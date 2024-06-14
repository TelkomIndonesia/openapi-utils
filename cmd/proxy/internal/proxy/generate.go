package proxy

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/internal/util"
)

const prefixUpstream = "Upstream"

func Generate(ctx context.Context, specPath string) (err error) {
	pe, err := NewProxyExtension(ctx, specPath)
	if err != nil {
		return fmt.Errorf("fail to create proxy extension: %w", err)
	}

	{
		spec, _, _, err := pe.CreateProxyDoc()
		if err != nil {
			return fmt.Errorf("fail to create proxy doc: %w", err)
		}
		kinspec, err := loadKinDoc(spec)
		if err != nil {
			return fmt.Errorf("fail to reload proxy doc with kin: %w", err)
		}

		t, err := loadTemplates("proxy")
		if err != nil {
			return fmt.Errorf("fail to load template: %w", err)
		}
		codegen.TemplateFunctions["upstreamOperationID"] = func(opid string) string {
			for k, v := range pe.proxied {
				if opid != k.OperationId {
					continue
				}

				uop, _ := v.GetUpstreamOperation()
				return prefixUpstream + uop.OperationId
			}
			return ""
		}
		code, err := codegen.Generate(kinspec, codegen.Configuration{
			PackageName: "oapi",
			Generate: codegen.GenerateOptions{
				EchoServer: true,
				Strict:     true,
				Models:     true,
			},
			OutputOptions: codegen.OutputOptions{
				UserTemplates: t,
			},
		})
		if err != nil {
			return fmt.Errorf("fail to generate code: %w", err)
		}
		err = os.WriteFile("testdata/gen/oapi-proxy.go", []byte(code), 0o644)
		if err != nil {
			return fmt.Errorf("fail to write generated code: %w", err)
		}
	}

	{
		t, err := loadTemplates("upstream")
		if err != nil {
			return fmt.Errorf("fail to load template: %w", err)
		}

		generated := map[*libopenapi.DocumentModel[v3.Document]]struct{}{}
		for _, pop := range pe.Proxied() {
			doc, err := pop.GetOpenAPIDoc()
			if err != nil {
				return fmt.Errorf("fail to find upstream openapi doc: %w", err)
			}
			docv3, _ := doc.BuildV3Model()
			if _, ok := generated[docv3]; ok {
				continue
			}

			// add prefix
			for m := range orderedmap.Iterate(ctx, docv3.Model.Paths.PathItems) {
				for _, op := range util.GetOperationsMap(m.Value()) {
					op.OperationId = prefixUpstream + op.OperationId
				}
			}
			components := util.NewStubComponents()
			components.CopyComponents(docv3, "")
			components.CopyComponents(docv3, prefixUpstream)
			_, ndoc, ndocv3, _ := components.RenderAndReload(doc)
			components = util.NewStubComponents()
			components.CopyAndLocalizeComponents(ndocv3, prefixUpstream)
			spec, _, _, _ := components.RenderAndReload(ndoc)

			kinspec, err := loadKinDoc(spec)
			if err != nil {
				return fmt.Errorf("fail to reload proxy doc with kin: %w", err)
			}

			code, err := codegen.Generate(kinspec, codegen.Configuration{
				PackageName: "oapi",
				Generate: codegen.GenerateOptions{
					EchoServer: true,
					Strict:     true,
					Models:     true,
				},
				OutputOptions: codegen.OutputOptions{
					UserTemplates: t,
				},
			})
			if err != nil {
				return fmt.Errorf("fail to generate code: %w", err)
			}

			file := fmt.Sprintf("testdata/gen/oapi-upstream-%s.go", strings.ToLower(pop.GetName()))
			err = os.WriteFile(file, []byte(code), 0o644)
			if err != nil {
				return fmt.Errorf("fail to write generated code: %w", err)
			}

			generated[docv3] = struct{}{}
		}
	}

	return
}

func loadKinDoc(data []byte) (doc *openapi3.T, err error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	doc, err = loader.LoadFromData(data)
	return
}

//go:embed templates/*
var templates embed.FS

func loadTemplates(dir string) (t map[string]string, err error) {
	t = make(map[string]string)
	err = fs.WalkDir(templates, path.Join("templates", dir), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		buf, err := templates.ReadFile(p)
		if err != nil {
			return fmt.Errorf("error reading file '%s': %w", p, err)
		}

		templateName := strings.TrimPrefix(p, path.Join("templates", dir)+"/")
		t[templateName] = string(buf)
		return nil
	})
	return

}
