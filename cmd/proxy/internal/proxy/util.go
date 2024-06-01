package proxy

import (
	"context"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	baselow "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

func initComponents(doc *libopenapi.DocumentModel[v3.Document]) {
	if doc.Model.Components == nil {
		doc.Model.Components = &v3.Components{}
	}
	if doc.Model.Components.Schemas == nil {
		doc.Model.Components.Schemas = &orderedmap.Map[string, *base.SchemaProxy]{}
	}
	if doc.Model.Components.Parameters == nil {
		doc.Model.Components.Parameters = &orderedmap.Map[string, *v3.Parameter]{}
	}
	if doc.Model.Components.RequestBodies == nil {
		doc.Model.Components.RequestBodies = &orderedmap.Map[string, *v3.RequestBody]{}
	}
	if doc.Model.Components.Responses == nil {
		doc.Model.Components.Responses = &orderedmap.Map[string, *v3.Response]{}
	}
	if doc.Model.Components.Headers == nil {
		doc.Model.Components.Headers = &orderedmap.Map[string, *v3.Header]{}
	}
	if doc.Model.Components.Links == nil {
		doc.Model.Components.Links = &orderedmap.Map[string, *v3.Link]{}
	}
	if doc.Model.Components.SecuritySchemes == nil {
		doc.Model.Components.SecuritySchemes = &orderedmap.Map[string, *v3.SecurityScheme]{}
	}
	if doc.Model.Components.Examples == nil {
		doc.Model.Components.Examples = &orderedmap.Map[string, *base.Example]{}
	}
	if doc.Model.Components.Extensions == nil {
		doc.Model.Components.Extensions = &orderedmap.Map[string, *yaml.Node]{}
	}
	if doc.Model.Components.Callbacks == nil {
		doc.Model.Components.Callbacks = &orderedmap.Map[string, *v3.Callback]{}
	}
}

func getOperationsMap(pi *v3.PathItem) (ops map[string]*v3.Operation) {
	ops = map[string]*v3.Operation{}
	if pi.Get != nil {
		ops["get"] = pi.Get
	}
	if pi.Delete != nil {
		ops["delete"] = pi.Delete
	}
	if pi.Post != nil {
		ops["post"] = pi.Post
	}
	if pi.Put != nil {
		ops["put"] = pi.Put
	}
	if pi.Patch != nil {
		ops["patch"] = pi.Patch
	}
	if pi.Options != nil {
		ops["options"] = pi.Options
	}
	if pi.Head != nil {
		ops["head"] = pi.Head
	}
	if pi.Trace != nil {
		ops["trace"] = pi.Trace
	}
	return
}

func getOperation(pi *v3.PathItem, method string) *v3.Operation {
	switch {
	case strings.EqualFold("get", method):
		return pi.Get
	case strings.EqualFold("delete", method):
		return pi.Delete
	case strings.EqualFold("post", method):
		return pi.Post
	case strings.EqualFold("put", method):
		return pi.Put
	case strings.EqualFold("patch", method):
		return pi.Patch
	case strings.EqualFold("options", method):
		return pi.Options
	case strings.EqualFold("head", method):
		return pi.Head
	case strings.EqualFold("trace", method):
		return pi.Trace
	}
	return nil
}

func setOperation(pi *v3.PathItem, method string, val *v3.Operation) {
	switch {
	case strings.EqualFold("get", method):
		pi.Get = val
	case strings.EqualFold("delete", method):
		pi.Delete = val
	case strings.EqualFold("post", method):
		pi.Post = val
	case strings.EqualFold("put", method):
		pi.Put = val
	case strings.EqualFold("patch", method):
		pi.Patch = val
	case strings.EqualFold("options", method):
		pi.Options = val
	case strings.EqualFold("head", method):
		pi.Head = val
	case strings.EqualFold("trace", method):
		pi.Trace = val
	}
}

type parameterKey struct {
	name string
	in   string
}

func copyParameters(src []*v3.Parameter, add ...*v3.Parameter) (dst []*v3.Parameter) {
	copied := map[parameterKey]struct{}{}
	dst = make([]*v3.Parameter, 0, len(src)+len(add))
	for _, p := range src {
		dst = append(dst, p)
		copied[parameterKey{name: p.Name, in: p.In}] = struct{}{}
	}
	for _, p := range add {
		if _, ok := copied[parameterKey{name: p.Name, in: p.In}]; ok {
			continue
		}
		dst = append(dst, p)
	}
	return
}

func duplicateSchema(ctx context.Context, ref *index.Reference, prefix string, m *orderedmap.Map[string, *base.SchemaProxy]) (err error) {
	schemaProxy, err := baselow.ExtractSchema(ctx, ref.Node, ref.Index)
	if err != nil {
		return fmt.Errorf("fail to recreate schema: %w", err)
	}

	name := prefix + ref.Name
	m.Set(name, base.NewSchemaProxy(schemaProxy))
	return
}

func copyComponents(ctx context.Context, src *libopenapi.DocumentModel[v3.Document], prefix string, dst *libopenapi.DocumentModel[v3.Document]) {
	for _, ref := range src.Index.GetRawReferencesSequenced() {
		switch {
		case strings.HasPrefix(ref.Definition, "#/components/schemas/"):
			copySchema(ctx, ref, prefix, dst.Model.Components.Schemas)

		case strings.HasPrefix(ref.Definition, "#/components/parameters/"):
			copyComponent(ctx, ref, prefix, dst.Model.Components.Parameters, v3.NewParameter)

		case strings.HasPrefix(ref.Definition, "#/components/requestBodies/"):
			copyComponent(ctx, ref, prefix, dst.Model.Components.RequestBodies, v3.NewRequestBody)

		case strings.HasPrefix(ref.Definition, "#/components/headers/"):
			copyComponent(ctx, ref, prefix, dst.Model.Components.Headers, v3.NewHeader)

		case strings.HasPrefix(ref.Definition, "#/components/responses/"):
			copyComponent(ctx, ref, prefix, dst.Model.Components.Responses, v3.NewResponse)

		case strings.HasPrefix(ref.Definition, "#/components/securitySchemes/"):
			copyComponent(ctx, ref, prefix, dst.Model.Components.SecuritySchemes, v3.NewSecurityScheme)

		case strings.HasPrefix(ref.Definition, "#/components/examples/"):
			copyComponent(ctx, ref, prefix, dst.Model.Components.Examples, base.NewExample)

		case strings.HasPrefix(ref.Definition, "#/components/links/"):
			copyComponent(ctx, ref, prefix, dst.Model.Components.Links, v3.NewLink)

		case strings.HasPrefix(ref.Definition, "#/components/callbacks/"):
			copyComponent(ctx, ref, prefix, dst.Model.Components.Callbacks, v3.NewCallback)
		}
	}
}

func copySchema(ctx context.Context, ref *index.Reference, prefix string, m *orderedmap.Map[string, *base.SchemaProxy]) (err error) {
	schemaProxy, err := baselow.ExtractSchema(ctx, ref.Node, ref.Index)
	if err != nil {
		return fmt.Errorf("fail to recreate schema: %w", err)
	}

	name := prefix + ref.Name
	refname := strings.TrimSuffix(ref.Definition, ref.Name) + name
	ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
	m.Set(name, base.NewSchemaProxy(schemaProxy))
	return
}

func copyComponent[B any, L low.Buildable[B], H high.GoesLow[L]](
	ctx context.Context,
	ref *index.Reference,
	prefix string,
	m *orderedmap.Map[string, H],
	fnew func(L) H,
) (err error) {
	v, err := low.ExtractObject[L](ctx, "", ref.Node, ref.Index)
	if err != nil {
		return fmt.Errorf("fail to extract object: %w", err)
	}
	v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

	name := prefix + ref.Name
	refname := strings.TrimSuffix(ref.Definition, ref.Name) + name
	ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
	m.Set(name, fnew(v.Value))

	return
}
