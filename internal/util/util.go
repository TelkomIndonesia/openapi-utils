package util

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

func InitComponents(doc *libopenapi.DocumentModel[v3.Document]) {
	comp := doc.Model.Components
	if comp == nil {
		comp = &v3.Components{}
	}
	if comp.Schemas == nil {
		comp.Schemas = orderedmap.New[string, *base.SchemaProxy]()
	}
	if comp.Parameters == nil {
		comp.Parameters = orderedmap.New[string, *v3.Parameter]()
	}
	if comp.RequestBodies == nil {
		comp.RequestBodies = orderedmap.New[string, *v3.RequestBody]()
	}
	if comp.Responses == nil {
		comp.Responses = orderedmap.New[string, *v3.Response]()
	}
	if comp.Headers == nil {
		comp.Headers = orderedmap.New[string, *v3.Header]()
	}
	if comp.Links == nil {
		comp.Links = orderedmap.New[string, *v3.Link]()
	}
	if comp.SecuritySchemes == nil {
		comp.SecuritySchemes = orderedmap.New[string, *v3.SecurityScheme]()
	}
	if comp.Examples == nil {
		comp.Examples = orderedmap.New[string, *base.Example]()
	}
	if comp.Extensions == nil {
		comp.Extensions = orderedmap.New[string, *yaml.Node]()
	}
	if comp.Callbacks == nil {
		comp.Callbacks = orderedmap.New[string, *v3.Callback]()
	}
	doc.Model.Components = comp
}

func GetOperationsMap(p *v3.PathItem) (ops map[string]*v3.Operation) {
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

func GetOperation(p *v3.PathItem, method string) *v3.Operation {
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

func SetOperation(p *v3.PathItem, method string, val *v3.Operation) {
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

func CopyDocComponentsAndRenameRef(ctx context.Context, src libopenapi.Document, prefix string, dst libopenapi.Document) (nsrc libopenapi.Document, err error) {
	srcv3, errs := src.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("fail to build src v3 model: %w", err)
	}

	// duplicate schema on source doc with added prefix
	for _, ref := range srcv3.Index.GetRawReferencesSequenced() {
		if !strings.HasPrefix(ref.Definition, "#/components/schemas/") {
			continue
		}

		copySchema(ctx, ref, prefix, srcv3.Model.Components.Schemas)
	}

	// rerender
	_, src, srcv3, errs = src.RenderAndReload()
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("faill to render and reload openapi doc: %w", err)
	}

	// copy all components
	dstv3, errs := dst.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("fail to build dst v3 model: %w", err)
	}

	InitComponents(dstv3)
	for _, ref := range srcv3.Index.GetRawReferencesSequenced() {
		err = CopyComponentAndRenameRef(ctx, ref, prefix, dstv3)
	}
	return src, err
}

func copySchema(ctx context.Context,
	ref *index.Reference,
	prefix string,
	dst *orderedmap.Map[string, *base.SchemaProxy],
) (err error) {
	schemaProxy, err := baselow.ExtractSchema(ctx, ref.Node, ref.Index)
	if err != nil {
		return fmt.Errorf("fail to recreate schema: %w", err)
	}

	name := prefix + ref.Name
	dst.Set(name, base.NewSchemaProxy(schemaProxy))
	return
}

func CopyComponentAndRenameRef(ctx context.Context,
	ref *index.Reference,
	prefix string,
	dstv3 *libopenapi.DocumentModel[v3.Document],
) (err error) {
	switch {
	case strings.HasPrefix(ref.Definition, "#/components/schemas/"):
		return copySchemaAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.Schemas)

	case strings.HasPrefix(ref.Definition, "#/components/parameters/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.Parameters, v3.NewParameter)

	case strings.HasPrefix(ref.Definition, "#/components/requestBodies/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.RequestBodies, v3.NewRequestBody)

	case strings.HasPrefix(ref.Definition, "#/components/headers/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.Headers, v3.NewHeader)

	case strings.HasPrefix(ref.Definition, "#/components/responses/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.Responses, v3.NewResponse)

	case strings.HasPrefix(ref.Definition, "#/components/securitySchemes/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.SecuritySchemes, v3.NewSecurityScheme)

	case strings.HasPrefix(ref.Definition, "#/components/examples/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.Examples, base.NewExample)

	case strings.HasPrefix(ref.Definition, "#/components/links/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.Links, v3.NewLink)

	case strings.HasPrefix(ref.Definition, "#/components/callbacks/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components.Callbacks, v3.NewCallback)
	}

	return nil
}

func copySchemaAndRenameRef(ctx context.Context,
	ref *index.Reference,
	prefix string,
	dst *orderedmap.Map[string, *base.SchemaProxy],
) (err error) {
	if err = copySchema(ctx, ref, prefix, dst); err != nil {
		return
	}

	name := prefix + ref.Name
	refname := strings.TrimSuffix(ref.Definition, ref.Name) + name
	ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
	return
}

func copyComponentAndRenameRef[B any, L low.Buildable[B], H high.GoesLow[L]](
	ctx context.Context,
	ref *index.Reference,
	prefix string,
	dst *orderedmap.Map[string, H],
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
	dst.Set(name, fnew(v.Value))

	refname := strings.TrimSuffix(ref.Definition, ref.Name) + name
	ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
	return
}

type ParameterKey struct {
	name string
	in   string
}

func NewParameterKey(name string, in string) ParameterKey {
	return ParameterKey{
		name: name,
		in:   in,
	}
}

func CopyParameters(src []*v3.Parameter, add ...*v3.Parameter) (dst []*v3.Parameter) {
	copied := map[ParameterKey]struct{}{}
	dst = make([]*v3.Parameter, 0, len(src)+len(add))
	for _, p := range src {
		dst = append(dst, p)
		copied[ParameterKey{name: p.Name, in: p.In}] = struct{}{}
	}
	for _, p := range add {
		if _, ok := copied[ParameterKey{name: p.Name, in: p.In}]; ok {
			continue
		}
		dst = append(dst, p)
	}
	return
}

func MapFirstEntry[K comparable, V any](m map[K]V) (e struct {
	Key   K
	Value V
}) {
	for k, v := range m {
		return struct {
			Key   K
			Value V
		}{
			Key:   k,
			Value: v,
		}
	}
	return
}
