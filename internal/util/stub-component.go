package util

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type StubComponents struct {
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

func NewStubComponents() (c StubComponents) {
	c = StubComponents{
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
	return
}

func (c StubComponents) CopyLocalizedComponents(docv3 *libopenapi.DocumentModel[v3.Document], prefix string) (err error) {
	return c.copyComponents(docv3, prefix, true)
}

func (c StubComponents) CopyComponents(docv3 *libopenapi.DocumentModel[v3.Document], prefix string) (err error) {
	return c.copyComponents(docv3, prefix, false)
}

func (c StubComponents) copyComponents(docv3 *libopenapi.DocumentModel[v3.Document], prefix string, localized bool) (err error) {
	indexes := append(docv3.Index.GetRolodex().GetIndexes(), docv3.Index)
	for _, idx := range indexes {
		for _, ref := range idx.GetRawReferencesSequenced() {
			if low.IsCircular(ref.Node, ref.Index) {
				if idx.GetLogger() != nil {
					idx.GetLogger().Warn("skipping circular reference",
						"ref", ref.FullDefinition)
				}
				continue
			}

			err := c.copyComponentNode(ref, prefix)
			if err != nil {
				return fmt.Errorf("fail to locate component: %w", err)
			}

			if localized {
				LocalizeReference(ref, prefix)
			}
		}
	}

	for m := range orderedmap.Iterate(context.Background(), docv3.Model.Components.Extensions) {
		c.Extensions.Set(m.Key(), m.Value())
	}

	err = c.replaceRootNodes(docv3)
	return
}

func (c StubComponents) copyComponentNode(src *index.Reference, prefix string) (err error) {
	node, err := locateNode(src)
	if err != nil {
		return fmt.Errorf("fail to locate component: %w", err)
	}

	name := prefix + src.Name
	switch {
	case strings.HasPrefix(src.Definition, "#/components/schemas/"):
		c.Schemas.Set(name, node)

	case strings.HasPrefix(src.Definition, "#/components/parameters/"):
		c.Parameters.Set(name, node)

	case strings.HasPrefix(src.Definition, "#/components/requestBodies/"):
		c.RequestBodies.Set(name, node)

	case strings.HasPrefix(src.Definition, "#/components/headers/"):
		c.Headers.Set(name, node)

	case strings.HasPrefix(src.Definition, "#/components/responses/"):
		c.Responses.Set(name, node)

	case strings.HasPrefix(src.Definition, "#/components/securitySchemes/"):
		c.SecuritySchemes.Set(name, node)

	case strings.HasPrefix(src.Definition, "#/components/examples/"):
		c.Examples.Set(name, node)

	case strings.HasPrefix(src.Definition, "#/components/links/"):
		c.Links.Set(name, node)

	case strings.HasPrefix(src.Definition, "#/components/callbacks/"):
		c.Callbacks.Set(name, node)
	}

	return nil
}

func locateNode(ref *index.Reference) (node *yaml.Node, err error) {
	idx := ref.Index
	if r := getFromMap(idx.GetAllComponentSchemas(), ref.Definition); r != nil {
		return r.Node, nil
	}
	if r := getFromMap(idx.GetAllParameters(), ref.Definition); r != nil {
		return r.Node, nil
	}
	if r := getFromMap(idx.GetAllRequestBodies(), ref.Definition); r != nil {
		return r.Node, nil
	}
	if r := getFromMap(idx.GetAllResponses(), ref.Definition); r != nil {
		return r.Node, nil
	}
	if r := getFromMap(idx.GetAllSecuritySchemes(), ref.Definition); r != nil {
		return r.Node, nil
	}
	if r := getFromMap(idx.GetAllExamples(), ref.Definition); r != nil {
		return r.Node, nil
	}
	if r := getFromMap(idx.GetAllLinks(), ref.Definition); r != nil {
		return r.Node, nil
	}
	if r := getFromMap(idx.GetAllCallbacks(), ref.Definition); r != nil {
		return r.Node, nil
	}

	node, _, err = low.LocateRefNode(ref.Node, ref.Index)
	if err != nil {
		return nil, fmt.Errorf("fail to locate component: %w", err)
	}
	return
}

func (c StubComponents) replaceRootNodes(docv3 *libopenapi.DocumentModel[v3.Document]) (err error) {
	y, err := c.ToYamlNode()
	if err != nil {
		return fmt.Errorf("fail to convert components into `*node.Yaml`: %w", err)
	}

	rolodex := docv3.Index.GetRolodex()
	indexes := append(rolodex.GetIndexes(), rolodex.GetRootIndex())
	for _, idx := range append(indexes, rolodex.GetRootIndex()) {
		idx.GetRootNode().Content = y.Content
	}
	return
}

func (c StubComponents) RenderAndReload(doc libopenapi.Document) (b []byte, ndoc libopenapi.Document, docv3 *libopenapi.DocumentModel[v3.Document], err error) {
	docv3, errs := doc.BuildV3Model()
	if err := errors.Join(errs...); err != nil {
		return nil, nil, nil, fmt.Errorf("fail to build v3 model: %w", err)
	}
	b, err = c.Render(docv3)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to render doc: %w", err)
	}

	ndoc, err = libopenapi.NewDocumentWithConfiguration(b, doc.GetConfiguration())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to parse new doc: %w", err)
	}
	docv3, errs = ndoc.BuildV3Model()
	if err := errors.Join(errs...); err != nil {
		return nil, nil, nil, fmt.Errorf("fail to build v3 model from new doc: %w", err)
	}

	return
}

func (c StubComponents) Render(docv3 *libopenapi.DocumentModel[v3.Document]) ([]byte, error) {
	comp, err := c.ToYamlNode()
	if err != nil {
		return nil, fmt.Errorf("fail to encode stub-components to yaml: %w", err)
	}

	y, err := docv3.Model.MarshalYAML()
	if err != nil {
		return nil, fmt.Errorf("fail to marshal modified doc to yaml :%w", err)
	}
	root := y.(*yaml.Node)

	_, rootComp := utils.FindKeyNode(v3low.ComponentsLabel, root.Content)
	if rootComp == nil {
		root.Content = append(root.Content, comp.Content...)
	} else {
		rootComp.Content = comp.Content[0].Content[1].Content
	}

	return yaml.Marshal(root)
}

func (c StubComponents) ToYamlNode() (n *yaml.Node, err error) {
	m := orderedmap.New[string, any]()
	m.Set("schemas", c.Schemas)
	m.Set("responses", c.Responses)
	m.Set("parameters", c.Parameters)
	m.Set("examples", c.Examples)
	m.Set("requestBodies", c.RequestBodies)
	m.Set("headers", c.Headers)
	m.Set("securitySchemes", c.SecuritySchemes)
	m.Set("links", c.Links)
	m.Set("callbacks", c.Callbacks)
	for item := range orderedmap.Iterate(context.Background(), c.Extensions) {
		m.Set(item.Key(), item.Value())
	}
	b, err := yaml.Marshal(map[string]interface{}{
		v3low.ComponentsLabel: m,
	})
	if err != nil {
		return nil, err
	}

	y := yaml.Node{}
	return &y, yaml.Unmarshal(b, &y)
}
