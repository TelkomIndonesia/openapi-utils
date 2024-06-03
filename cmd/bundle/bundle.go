package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/telkomindonesia/openapi-utils/internal/util"
	"gopkg.in/yaml.v3"
)

func bundleFile(p string) (bytes []byte, err error) {
	f, _ := filepath.Abs(p)
	d, _ := filepath.Abs(filepath.Dir(p))
	by, _ := os.ReadFile(f)

	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(by), &datamodel.DocumentConfiguration{
		BasePath:                d,
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	})
	if err != nil {
		return nil, fmt.Errorf("fail to load openapi spec: %w", err)
	}

	bytes, err = bundle(doc, false)
	if err != nil {
		return nil, fmt.Errorf("fail to bundle: %w", err)
	}
	// docv3, errs := doc.BuildV3Model()
	// if len(errs) > 0 {
	// 	return nil, fmt.Errorf("fail to re-build openapi spec: %w", errors.Join(errs...))
	// }
	// bytes, err = bundler.BundleDocument(&docv3.Model)
	// if err != nil {
	// 	return nil, fmt.Errorf("fail to bundle: %w", err)
	// }
	return
}

func bundle(doc libopenapi.Document, inline bool) (b []byte, err error) {
	docv3, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		return nil, fmt.Errorf("fail to re-build openapi spec: %w", errors.Join(errs...))
	}
	util.InitComponents(docv3)

	shouldSkip := func(sequenced *index.Reference, idx *index.SpecIndex, root bool) bool {
		mappedReferences := idx.GetMappedReferences()

		// if we're in the root document, don't bundle anything.
		refExp := strings.Split(sequenced.FullDefinition, "#/")
		if len(refExp) == 2 {
			if refExp[0] == sequenced.Index.GetSpecAbsolutePath() || refExp[0] == "" {
				if root && !inline {
					idx.GetLogger().Debug("[bundler] skipping local root reference",
						"ref", sequenced.Definition)
					return true
				}
			}
		}

		mappedReference := mappedReferences[sequenced.FullDefinition]
		if mappedReference == nil {
			return true
		}

		if mappedReference.Circular {
			if idx.GetLogger() != nil {
				idx.GetLogger().Warn("[bundler] skipping circular reference",
					"ref", sequenced.FullDefinition)
			}
			return true
		}

		return false
	}

	compact := func(idx *index.SpecIndex, root bool) (err error) {
		for _, ref := range idx.GetRawReferencesSequenced() {
			if shouldSkip(ref, idx, root) {
				continue
			}

			err := util.CopyComponentAndRenameRef(context.Background(), ref, "", docv3.Model.Components)
			if err != nil {
				return fmt.Errorf("fail to copy components: %w", err)
			}
		}
		return
	}

	rolodex := docv3.Model.Rolodex
	indexes := rolodex.GetIndexes()
	for _, idx := range indexes {
		if err = compact(idx, false); err != nil {
			return nil, fmt.Errorf("fail to compact: %w", err)
		}
	}
	if err = compact(rolodex.GetRootIndex(), true); err != nil {
		return nil, fmt.Errorf("fail to compact: %w", err)
	}

	// copy components into root node in case new references need to be resolved, e.g. reference inside `allOf`
	components, err := toYamlNode("components", *docv3.Model.Components)
	if err != nil {
		return nil, fmt.Errorf("fail to convert components into `*node.Yaml`: %w", err)
	}
	for _, idx := range append(indexes, rolodex.GetRootIndex()) {
		idx.GetRootNode().Content = components.Content
	}

	return docv3.Model.Render()
}

func toYamlNode(key string, v interface{}) (n *yaml.Node, err error) {
	b, err := yaml.Marshal(map[string]interface{}{
		key: v,
	})
	if err != nil {
		return nil, err
	}
	y := yaml.Node{}
	return &y, yaml.Unmarshal(b, &y)
}
