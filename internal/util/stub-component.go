package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
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

	doc libopenapi.Document
}

func NewStubComponents(doc libopenapi.Document) (c StubComponents) {
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

		doc: doc,
	}

	return
}

func (c StubComponents) CopyNodesAndRenameRefs(prefix string) (err error) {
	docv3, errs := c.doc.BuildV3Model()
	if err := errors.Join(errs...); err != nil {
		return err
	}

	rolodex := docv3.Index.GetRolodex()
	indexes := append(rolodex.GetIndexes(), rolodex.GetRootIndex())
	for _, idx := range indexes {
		for _, ref := range idx.GetRawReferencesSequenced() {
			if isCircular(ref) {
				if idx.GetLogger() != nil {
					idx.GetLogger().Warn("skipping circular reference",
						"ref", ref.FullDefinition)
				}
				continue
			}

			err := c.copyNode(ref, prefix)
			if err != nil {
				return fmt.Errorf("fail to locate component: %w", err)
			}

			name := prefix + ref.Name
			refdef := strings.TrimSuffix(ref.Definition, ref.Name) + name
			ref.Node.Content = base.CreateSchemaProxyRef(refdef).GetReferenceNode().Content
		}
	}

	err = c.copyToRootNode(docv3)
	return
}

func isCircular(sequenced *index.Reference) bool {
	idx := sequenced.Index
	mappedReferences := idx.GetMappedReferences()
	mappedReference := mappedReferences[sequenced.FullDefinition]
	if mappedReference == nil {
		return true
	}

	return mappedReference.Circular
}

func (c StubComponents) copyNode(src *index.Reference, prefix string) (err error) {
	node, _, err := low.LocateRefNode(src.Node, src.Index)
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

func (c StubComponents) copyToRootNode(docv3 *libopenapi.DocumentModel[v3.Document]) (err error) {
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

func (c StubComponents) Render() ([]byte, error) {
	docv3, errs := c.doc.BuildV3Model()
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

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
	b, err := yaml.Marshal(map[string]interface{}{
		v3low.ComponentsLabel: c,
	})
	if err != nil {
		return nil, err
	}

	y := yaml.Node{}
	return &y, yaml.Unmarshal(b, &y)
}
