package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/telkomindonesia/openapi-utils/internal/util"
)

func bundleFile(p string) (bytes []byte, err error) {
	by, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("fail to read file :%w", err)
	}
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(by), &datamodel.DocumentConfiguration{
		BasePath:                filepath.Dir(p),
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	})
	if err != nil {
		return nil, fmt.Errorf("fail to load openapi spec: %w", err)
	}

	bytes, err = bundle(doc)
	if err != nil {
		return nil, fmt.Errorf("fail to bundle: %w", err)
	}

	return
}

func bundle(doc libopenapi.Document) (b []byte, err error) {
	docv3, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		return nil, fmt.Errorf("fail to re-build openapi spec: %w", errors.Join(errs...))
	}

	// create stub components and localize all references
	components := util.NewStubComponents()
	err = components.CopyNodesAndLocalizeRefs(docv3, "")
	if err != nil {
		return nil, fmt.Errorf("fail to copy stub components: %w", err)
	}

	// copy all high-level components
	// this might be unnecessary since we will replace them with stub-component
	// but just to be sure that the render won't fail
	util.InitComponents(docv3)
	for _, idx := range docv3.Index.GetRolodex().GetIndexes() {
		for _, ref := range idx.GetRawReferencesSequenced() {
			err = util.CopyComponent(context.Background(), ref, "", docv3.Model.Components)
			if err != nil {
				return nil, fmt.Errorf("fail to copy high-level components :%w", err)
			}
		}
	}

	return components.Render(docv3)
}
