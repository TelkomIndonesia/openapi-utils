package proxy

import (
	"context"
	"errors"
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

func getOperationsMap(p *v3.PathItem) (ops map[string]*v3.Operation) {
	ops = map[string]*v3.Operation{}
	if p.Get != nil {
		ops["get"] = p.Get
	}
	if p.Delete != nil {
		ops["delete"] = p.Delete
	}
	if p.Post != nil {
		ops["post"] = p.Post
	}
	if p.Put != nil {
		ops["put"] = p.Put
	}
	if p.Patch != nil {
		ops["patch"] = p.Patch
	}
	if p.Options != nil {
		ops["options"] = p.Options
	}
	if p.Head != nil {
		ops["head"] = p.Head
	}
	if p.Trace != nil {
		ops["trace"] = p.Trace
	}
	return
}

func getOperation(p *v3.PathItem, method string) *v3.Operation {
	switch {
	case strings.EqualFold("get", method):
		return p.Get
	case strings.EqualFold("delete", method):
		return p.Delete
	case strings.EqualFold("post", method):
		return p.Post
	case strings.EqualFold("put", method):
		return p.Put
	case strings.EqualFold("patch", method):
		return p.Patch
	case strings.EqualFold("options", method):
		return p.Options
	case strings.EqualFold("head", method):
		return p.Head
	case strings.EqualFold("trace", method):
		return p.Trace
	}
	return nil
}

func setOperation(p *v3.PathItem, method string, val *v3.Operation) {
	switch {
	case strings.EqualFold("get", method):
		p.Get = val
	case strings.EqualFold("delete", method):
		p.Delete = val
	case strings.EqualFold("post", method):
		p.Post = val
	case strings.EqualFold("put", method):
		p.Put = val
	case strings.EqualFold("patch", method):
		p.Patch = val
	case strings.EqualFold("options", method):
		p.Options = val
	case strings.EqualFold("head", method):
		p.Head = val
	case strings.EqualFold("trace", method):
		p.Trace = val
	}
}

func modCopyComponents(ctx context.Context, src libopenapi.Document, prefix string, dst libopenapi.Document) (nsrc libopenapi.Document, err error) {
	srcv3, errs := src.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("fail to build v3 model: %w", err)
	}

	// duplicate schema on source doc with added prefix
	for _, ref := range srcv3.Index.GetRawReferencesSequenced() {
		if !strings.HasPrefix(ref.Definition, "#/components/schemas/") {
			continue
		}

		duplicateSchema(ctx, ref, prefix, srcv3.Model.Components.Schemas)
	}

	// rerender
	_, src, srcv3, errs = src.RenderAndReload()
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("faill to render and reload openapi doc: %w", err)
	}

	// copy all components
	dstv3, errs := dst.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("fail to build v3 model: %w", err)
	}
	for _, ref := range srcv3.Index.GetRawReferencesSequenced() {
		switch {
		case strings.HasPrefix(ref.Definition, "#/components/schemas/"):
			modCopySchema(ctx, ref, prefix, dstv3.Model.Components.Schemas)

		case strings.HasPrefix(ref.Definition, "#/components/parameters/"):
			modCopyComponent(ctx, ref, prefix, dstv3.Model.Components.Parameters, v3.NewParameter)

		case strings.HasPrefix(ref.Definition, "#/components/requestBodies/"):
			modCopyComponent(ctx, ref, prefix, dstv3.Model.Components.RequestBodies, v3.NewRequestBody)

		case strings.HasPrefix(ref.Definition, "#/components/headers/"):
			modCopyComponent(ctx, ref, prefix, dstv3.Model.Components.Headers, v3.NewHeader)

		case strings.HasPrefix(ref.Definition, "#/components/responses/"):
			modCopyComponent(ctx, ref, prefix, dstv3.Model.Components.Responses, v3.NewResponse)

		case strings.HasPrefix(ref.Definition, "#/components/securitySchemes/"):
			modCopyComponent(ctx, ref, prefix, dstv3.Model.Components.SecuritySchemes, v3.NewSecurityScheme)

		case strings.HasPrefix(ref.Definition, "#/components/examples/"):
			modCopyComponent(ctx, ref, prefix, dstv3.Model.Components.Examples, base.NewExample)

		case strings.HasPrefix(ref.Definition, "#/components/links/"):
			modCopyComponent(ctx, ref, prefix, dstv3.Model.Components.Links, v3.NewLink)

		case strings.HasPrefix(ref.Definition, "#/components/callbacks/"):
			modCopyComponent(ctx, ref, prefix, dstv3.Model.Components.Callbacks, v3.NewCallback)
		}
	}

	return src, nil
}

func duplicateSchema(ctx context.Context,
	ref *index.Reference,
	prefix string,
	m *orderedmap.Map[string, *base.SchemaProxy],
) (err error) {
	schemaProxy, err := baselow.ExtractSchema(ctx, ref.Node, ref.Index)
	if err != nil {
		return fmt.Errorf("fail to recreate schema: %w", err)
	}

	name := prefix + ref.Name
	m.Set(name, base.NewSchemaProxy(schemaProxy))
	return
}

func modCopySchema(ctx context.Context,
	ref *index.Reference,
	prefix string,
	m *orderedmap.Map[string, *base.SchemaProxy],
) (err error) {
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

func modCopyComponent[B any, L low.Buildable[B], H high.GoesLow[L]](
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
	err = v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)
	if err != nil {
		return fmt.Errorf("fail to build object: %w", err)
	}

	name := prefix + ref.Name
	refname := strings.TrimSuffix(ref.Definition, ref.Name) + name
	ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
	m.Set(name, fnew(v.Value))
	return
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

func mapFirstEntry[K comparable, V any](m map[K]V) (e struct {
	key   K
	value V
}) {
	for k, v := range m {
		return struct {
			key   K
			value V
		}{
			key:   k,
			value: v,
		}
	}
	return
}
