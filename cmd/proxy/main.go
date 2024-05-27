package main

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func main() {
	src := os.Args[1]
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
	if err != nil {
		log.Fatalln("fail to load openapi spec:", err)
	}
	docv3, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		log.Fatalln("fail to load openapi spec as v3:", errs)
	}

	v, p := docv3.Model.Paths.PathItems.Delete("/tenants/{tenant-id}/profiles")
	if !p {
		log.Fatalln("path doesn't exists")
	}
	var idx int
	for i, param := range v.Post.Parameters {
		if param.Name != "tenant-id" {
			continue
		}
		idx = i
	}
	v.Post.Parameters = append(v.Post.Parameters[:idx], v.Post.Parameters[idx+1:]...)
	v.Post.Parameters = append(v.Post.Parameters, &v3.Parameter{
		Name: "test",
		In:   "path",
	})
	docv3.Model.Paths.PathItems.Set("/profiles", v)

	b, doc, docv3, errs := doc.RenderAndReload()
	if len(errs) > 0 {
		log.Fatalln("fail to rerender openapi spec:", errs)
	}
	os.Stdout.Write(b)
}
