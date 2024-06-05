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
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/internal/util"
	"gopkg.in/yaml.v3"
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

	bytes, err = bundle(doc, false)
	if err != nil {
		return nil, fmt.Errorf("fail to bundle: %w", err)
	}

	return
}

func bundle(doc libopenapi.Document, inline bool) (b []byte, err error) {
	docv3, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		return nil, fmt.Errorf("fail to re-build openapi spec: %w", errors.Join(errs...))
	}

	// create dummy components and localize all references
	components := newDummyComponents()
	rolodex := docv3.Model.Rolodex
	indexes := rolodex.GetIndexes()
	for _, idx := range indexes {
		if err = components.copyComponentsAndLocalize(idx, false, inline); err != nil {
			return nil, fmt.Errorf("fail to compact: %w", err)
		}
	}
	if err = components.copyComponentsAndLocalize(rolodex.GetRootIndex(), true, inline); err != nil {
		return nil, fmt.Errorf("fail to compact: %w", err)
	}

	// copy marshalled components into root node of all index to avoid lookup error
	y, err := components.toYamlNode()
	if err != nil {
		return nil, fmt.Errorf("fail to convert components into `*node.Yaml`: %w", err)
	}
	for _, idx := range append(indexes, rolodex.GetRootIndex()) {
		idx.GetRootNode().Content = y.Content
	}

	// copy all components
	util.InitComponents(docv3)
	for _, idx := range indexes {
		for _, ref := range idx.GetRawReferencesSequenced() {
			util.CopyComponent(context.Background(), ref, "", docv3.Model.Components)
		}
	}

	return docv3.Model.Render()
}

type dummyComponents struct {
	Schemas         *orderedmap.Map[string, *yaml.Node] `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Responses       *orderedmap.Map[string, *yaml.Node] `json:"responses,omitempty" yaml:"responses,omitempty"`
	Parameters      *orderedmap.Map[string, *yaml.Node] `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Examples        *orderedmap.Map[string, *yaml.Node] `json:"examples,omitempty" yaml:"examples,omitempty"`
	RequestBodies   *orderedmap.Map[string, *yaml.Node] `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Headers         *orderedmap.Map[string, *yaml.Node] `json:"headers,omitempty" yaml:"headers,omitempty"`
	SecuritySchemes *orderedmap.Map[string, *yaml.Node] `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Links           *orderedmap.Map[string, *yaml.Node] `json:"links,omitempty" yaml:"links,omitempty"`
	Callbacks       *orderedmap.Map[string, *yaml.Node] `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
	Extensions      *orderedmap.Map[string, *yaml.Node] `json:"-" yaml:"-"`
}

func newDummyComponents() dummyComponents {
	return dummyComponents{
		Schemas:         orderedmap.New[string, *yaml.Node](),
		Responses:       orderedmap.New[string, *yaml.Node](),
		Parameters:      orderedmap.New[string, *yaml.Node](),
		Examples:        orderedmap.New[string, *yaml.Node](),
		RequestBodies:   orderedmap.New[string, *yaml.Node](),
		Headers:         orderedmap.New[string, *yaml.Node](),
		SecuritySchemes: orderedmap.New[string, *yaml.Node](),
		Links:           orderedmap.New[string, *yaml.Node](),
		Callbacks:       orderedmap.New[string, *yaml.Node](),
		Extensions:      orderedmap.New[string, *yaml.Node](),
	}
}

func (c dummyComponents) copyComponentsAndLocalize(idx *index.SpecIndex, root bool, inline bool) (err error) {
	for _, ref := range idx.GetRawReferencesSequenced() {
		if !shouldCopy(ref, idx, root, inline) {
			continue
		}

		err := c.copyNode(ref)
		if err != nil {
			return fmt.Errorf("fail to locate component: %w", err)
		}

		ref.Node.Content = base.CreateSchemaProxyRef(ref.Definition).GetReferenceNode().Content
	}
	return
}

func shouldCopy(sequenced *index.Reference, idx *index.SpecIndex, root bool, inline bool) bool {
	mappedReferences := idx.GetMappedReferences()

	// if we're in the root document, don't bundle anything.
	refExp := strings.Split(sequenced.FullDefinition, "#/")
	if len(refExp) == 2 {
		if refExp[0] == sequenced.Index.GetSpecAbsolutePath() || refExp[0] == "" {
			if root && !inline {
				idx.GetLogger().Debug("[bundler] skipping local root reference",
					"ref", sequenced.Definition)
				return false
			}
		}
	}

	mappedReference := mappedReferences[sequenced.FullDefinition]
	if mappedReference == nil {
		return false
	}

	if mappedReference.Circular {
		if idx.GetLogger() != nil {
			idx.GetLogger().Warn("[bundler] skipping circular reference",
				"ref", sequenced.FullDefinition)
		}
		return false
	}

	return true
}

func (c dummyComponents) copyNode(src *index.Reference) (err error) {
	node, _, err := low.LocateRefNode(src.Node, src.Index)
	if err != nil {
		return fmt.Errorf("fail to locate component: %w", err)
	}

	switch {
	case strings.HasPrefix(src.Definition, "#/components/schemas/"):
		c.Schemas.Set(src.Name, node)

	case strings.HasPrefix(src.Definition, "#/components/parameters/"):
		c.Parameters.Set(src.Name, node)

	case strings.HasPrefix(src.Definition, "#/components/requestBodies/"):
		c.RequestBodies.Set(src.Name, node)

	case strings.HasPrefix(src.Definition, "#/components/headers/"):
		c.Headers.Set(src.Name, node)

	case strings.HasPrefix(src.Definition, "#/components/responses/"):
		c.Responses.Set(src.Name, node)

	case strings.HasPrefix(src.Definition, "#/components/securitySchemes/"):
		c.SecuritySchemes.Set(src.Name, node)

	case strings.HasPrefix(src.Definition, "#/components/examples/"):
		c.Examples.Set(src.Name, node)

	case strings.HasPrefix(src.Definition, "#/components/links/"):
		c.Links.Set(src.Name, node)

	case strings.HasPrefix(src.Definition, "#/components/callbacks/"):
		c.Callbacks.Set(src.Name, node)
	}

	return nil
}

func (c dummyComponents) toYamlNode() (n *yaml.Node, err error) {
	b, err := yaml.Marshal(map[string]interface{}{
		"components": c,
	})
	if err != nil {
		return nil, err
	}

	y := yaml.Node{}
	return &y, yaml.Unmarshal(b, &y)
}
