package proxy

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func Generate(ctx context.Context, specPath string) (err error) {
	pe, err := NewProxyExtension(ctx, specPath)
	if err != nil {
		return fmt.Errorf("fail to create proxy extension: %w", err)
	}

	t, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("fail to load template: %w", err)
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
		err = os.WriteFile("testdata/gen/oapi-server.go", []byte(code), 0o644)
		if err != nil {
			return fmt.Errorf("fail to write generated code: %w", err)
		}
	}

	{
		generated := map[*libopenapi.DocumentModel[v3.Document]]struct{}{}
		for _, pop := range pe.proxied {
			doc, err := pop.GetOpenAPIDoc()
			if err != nil {
				return fmt.Errorf("fail to find upstream openapi doc: %w", err)
			}
			docv3, _ := doc.BuildV3Model()
			if _, ok := generated[docv3]; ok {
				continue
			}

			spec, err := doc.Render()
			if err != nil {
				return fmt.Errorf("fail to render upstream openapi doc: %w", err)
			}
			kinspec, err := loadKinDoc(spec)
			if err != nil {
				return fmt.Errorf("fail to reload proxy doc with kin: %w", err)
			}
			code, err := codegen.Generate(kinspec, codegen.Configuration{
				PackageName: "oapi",
				Generate: codegen.GenerateOptions{
					Client: true,
					Models: true,
				},
				OutputOptions: codegen.OutputOptions{
					UserTemplates:  t,
					ClientTypeName: "Client",
				},
			})
			if err != nil {
				return fmt.Errorf("fail to generate code: %w", err)
			}

			file := fmt.Sprintf("testdata/gen/oapi-proxy-%s.go", pop.GetName())
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

func loadTemplates() (t map[string]string, err error) {
	t = make(map[string]string)
	err = fs.WalkDir(templates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		buf, err := templates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file '%s': %w", path, err)
		}

		templateName := strings.TrimPrefix(path, "templates/")
		t[templateName] = string(buf)
		return nil
	})
	return

}
