package util

import (
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
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

func NewStubComponents() StubComponents {
	return StubComponents{
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

func (c StubComponents) CopyToRootNode(idx *index.SpecIndex, prefix string) (err error) {
	rolodex := idx.GetRolodex()
	indexes := rolodex.GetIndexes()
	for _, idx := range indexes {
		if err = c.copyNodesAndRenameRefs(idx, prefix); err != nil {
			return fmt.Errorf("fail to compact: %w", err)
		}
	}
	if err = c.copyNodesAndRenameRefs(rolodex.GetRootIndex(), prefix); err != nil {
		return fmt.Errorf("fail to compact: %w", err)
	}

	y, err := c.ToYamlNode()
	if err != nil {
		return fmt.Errorf("fail to convert components into `*node.Yaml`: %w", err)
	}
	for _, idx := range append(indexes, rolodex.GetRootIndex()) {
		idx.GetRootNode().Content = y.Content
	}
	return
}

func (c StubComponents) copyNodesAndRenameRefs(idx *index.SpecIndex, prefix string) (err error) {
	for _, ref := range idx.GetRawReferencesSequenced() {
		if !shouldCopy(ref, idx) {
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
	return
}

func shouldCopy(sequenced *index.Reference, idx *index.SpecIndex) bool {
	mappedReferences := idx.GetMappedReferences()
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

func (c StubComponents) ToYamlNode() (n *yaml.Node, err error) {
	b, err := yaml.Marshal(map[string]interface{}{
		v3.ComponentsLabel: c,
	})
	if err != nil {
		return nil, err
	}

	y := yaml.Node{}
	return &y, yaml.Unmarshal(b, &y)
}
