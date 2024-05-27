package config

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	baselow "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	src := "./testdata/spec-proxy.yml"
	sf, _ := filepath.Abs(src)
	specDir, _ := filepath.Abs(filepath.Dir(src))
	specBytes, _ := os.ReadFile(sf)

	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(specBytes), &datamodel.DocumentConfiguration{
		BasePath:                specDir,
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	})
	require.NoError(t, err, "should not return error")

	v3, errs := doc.BuildV3Model()
	require.Len(t, errs, 0, "should not return errors")

	path, ok := v3.Model.Paths.PathItems.Get("/profiles/{profile-id}")
	require.True(t, ok, "path should be present")
	ex, ok := path.Get.Extensions.Get("x-proxy")
	require.True(t, ok, "extension should exists")
	var proxy Proxy
	require.NoError(t, ex.Decode(&proxy), "should load extension")
}

func TestCompile(t *testing.T) {
	src := "./testdata/spec-proxy.yml"
	sf, _ := filepath.Abs(src)
	specDir, _ := filepath.Abs(filepath.Dir(src))
	specBytes, _ := os.ReadFile(sf)

	docProxy, err := libopenapi.NewDocument([]byte(specBytes))
	require.NoError(t, err, "should not return error")
	modelProxy, errs := docProxy.BuildV3Model()
	require.Len(t, errs, 0, "should not return errors")

	pathProxy, ok := modelProxy.Model.Paths.PathItems.Get("/profiles/{profile-id}")
	require.True(t, ok, "path should be present")
	opProxy := pathProxy.Get
	ex, ok := opProxy.Extensions.Get("x-proxy")
	require.True(t, ok, "extension should exists")
	var proxy Proxy
	require.NoError(t, ex.Decode(&proxy), "should load extension")

	specProfile, err := os.ReadFile(path.Join(specDir, proxy.Spec))
	require.NoError(t, err)
	docProfile, err := libopenapi.NewDocument(specProfile)
	require.NoError(t, err)
	modelProfile, errs := docProfile.BuildV3Model()
	require.NoError(t, errors.Join(errs...))
	pathProfile, ok := modelProfile.Model.Paths.PathItems.Get(proxy.Path)
	require.True(t, ok)
	require.NotNil(t, pathProfile.Get)

	switch {
	case !strings.EqualFold(proxy.Method, "get"):
		pathProfile.Get = nil
		fallthrough
	case !strings.EqualFold(proxy.Method, "post"):
		pathProfile.Post = nil
		fallthrough
	case !strings.EqualFold(proxy.Method, "put"):
		pathProfile.Put = nil
		fallthrough
	case !strings.EqualFold(proxy.Method, "patch"):
		pathProfile.Patch = nil
		fallthrough
	case !strings.EqualFold(proxy.Method, "delete"):
		pathProfile.Delete = nil
		fallthrough
	case !strings.EqualFold(proxy.Method, "head"):
		pathProfile.Head = nil
		fallthrough
	case !strings.EqualFold(proxy.Method, "options"):
		pathProfile.Options = nil
		fallthrough
	case !strings.EqualFold(proxy.Method, "trace"):
		pathProfile.Trace = nil
		fallthrough
	default:
	}
	require.NotNil(t, pathProfile.Get)
	require.Nil(t, pathProfile.Delete)

	for pi := range orderedmap.Iterate(context.Background(), modelProfile.Model.Paths.PathItems) {
		if pathProfile == pi.Value() {
			continue
		}

		modelProfile.Model.Paths.PathItems.Delete(pi.Key())
	}
	_, docProfile, modelProfile, errs = docProfile.RenderAndReload()
	require.NoError(t, errors.Join(errs...))
	pathProfile, ok = modelProfile.Model.Paths.PathItems.Get(proxy.Path)
	require.True(t, ok)
	opProfile := pathProfile.Get

	for _, r := range modelProfile.Index.GetRawReferencesSequenced() {
		ref := ""
		switch {
		case strings.HasPrefix(r.Definition, "#/components/schemas"):
			name := r.Name + proxy.GetSuffix()
			ref = "#/components/schemas/" + name
			schema := &baselow.Schema{}
			schema.Build(context.Background(), r.Node, r.Index)
			r.Node.Content = base.CreateSchemaProxyRef(ref).GetReferenceNode().Content

			modelProxy.Model.Components.Schemas.Set(name, base.CreateSchemaProxy(base.NewSchema(schema)))
		}
		if ref == "" {
			continue
		}
	}
	params := opProxy.Parameters
	pathProxy.Get = opProfile
	opProxy = pathProxy.Get
	opProxy.Parameters = []*v3.Parameter{}
	for _, param := range opProfile.Parameters {
		for _, inject := range proxy.Inject.Parameters {
			t.Log("param", param.Name, param.In)
			if inject.Name == param.Name && inject.In == param.In {
				continue
			}

			t.Log("adding param", param.Name, param.In)
			opProxy.Parameters = append(opProxy.Parameters, param)
			break
		}
	}
	for _, param := range params {
		opProxy.Parameters = append(opProxy.Parameters, param)
	}

	by, docProxy, modelProxy, errs := docProxy.RenderAndReload()
	require.NoError(t, errors.Join(errs...))
	t.Log(string(by))
}
