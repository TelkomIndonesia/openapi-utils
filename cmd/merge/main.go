package main

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <path-to-main-spec> [<path-to-new-spec>]\n", os.Args[0])
	}

	dst := ""
	if len(os.Args) > 2 {
		dst = os.Args[2]
	}
	s := os.Args[1]
	sf, _ := filepath.Abs(s)
	specDir, _ := filepath.Abs(filepath.Dir(s))
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

	v3Doc, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		log.Fatalln("fail to re-build openapi spec:", errs)
	}

	bytes, err := bundler.BundleDocument(&v3Doc.Model)
	if err != nil {
		log.Fatalln("fail to bundle openapi spec:", err)
	}

	if dst == "" {
		if _, err := os.Stdout.Write(bytes); err != nil {
			log.Fatalln("fail to write stdout:", err)
		}
		return
	}

	if err := os.WriteFile(dst, bytes, 0644); err != nil {
		log.Fatalln("fail to write file:", err)
	}

}
