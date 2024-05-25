package main

import (
	"context"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	baselow "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
)

func bundle(model *v3.Document, inline bool) ([]byte, error) {
	rolodex := model.Rolodex
	model.Components = &v3.Components{
		Schemas: orderedmap.New[string, *base.SchemaProxy](),
	}
	compact := func(idx *index.SpecIndex, root bool) {
		sequencedReferences := idx.GetRawReferencesSequenced()
		for _, sequenced := range sequencedReferences {
			switch {
			case strings.HasPrefix(sequenced.Definition, "#/components/schemas"):
				schema := &baselow.Schema{}
				schema.Build(context.Background(), sequenced.Node, sequenced.Index)
				model.Components.Schemas.Set(sequenced.Name, base.CreateSchemaProxy(base.NewSchema(schema)))
			}
		}

		mappedReferences := idx.GetMappedReferences()
		for _, sequenced := range sequencedReferences {
			mappedReference := mappedReferences[sequenced.FullDefinition]

			// if we're in the root document, don't bundle anything.
			refExp := strings.Split(sequenced.FullDefinition, "#/")
			if len(refExp) == 2 {
				if refExp[0] == sequenced.Index.GetSpecAbsolutePath() || refExp[0] == "" {
					if root && !inline {
						idx.GetLogger().Debug("[bundler] skipping local root reference",
							"ref", sequenced.Definition)
						continue
					}
				}
			}

			if mappedReference != nil && !mappedReference.Circular {
				p := base.CreateSchemaProxyRef("#/components/schemas/" + sequenced.Name)
				sequenced.Node.Content = p.GetReferenceNode().Content
				continue
			}
			if mappedReference != nil && mappedReference.Circular {
				if idx.GetLogger() != nil {
					idx.GetLogger().Warn("[bundler] skipping circular reference",
						"ref", sequenced.FullDefinition)
				}
			}
		}
	}

	indexes := rolodex.GetIndexes()
	for _, idx := range indexes {
		compact(idx, false)
	}
	compact(rolodex.GetRootIndex(), true)
	return model.Render()
}
