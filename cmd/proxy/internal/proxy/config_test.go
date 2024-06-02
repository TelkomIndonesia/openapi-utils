package proxy

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
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
	var proxy ProxyOperation
	require.NoError(t, ex.Decode(&proxy), "should load extension")
}
