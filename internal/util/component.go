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

func CopyComponentsAndRenameRefs(ctx context.Context, src libopenapi.Document, prefix string, dst libopenapi.Document) (nsrc libopenapi.Document, err error) {
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
		err = CopyComponentAndRenameRef(ctx, ref, prefix, dstv3.Model.Components)
	}
	return src, err
}

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

func CopyComponent(ctx context.Context,
	ref *index.Reference,
	prefix string,
	dst *v3.Components,
) (err error) {
	switch {
	case strings.HasPrefix(ref.Definition, "#/components/schemas/"):
		return copySchema(ctx, ref, prefix, dst.Schemas)

	case strings.HasPrefix(ref.Definition, "#/components/parameters/"):
		return copyComponent(ctx, ref, prefix, dst.Parameters, v3.NewParameter)

	case strings.HasPrefix(ref.Definition, "#/components/requestBodies/"):
		return copyComponent(ctx, ref, prefix, dst.RequestBodies, v3.NewRequestBody)

	case strings.HasPrefix(ref.Definition, "#/components/headers/"):
		return copyComponent(ctx, ref, prefix, dst.Headers, v3.NewHeader)

	case strings.HasPrefix(ref.Definition, "#/components/responses/"):
		return copyComponent(ctx, ref, prefix, dst.Responses, v3.NewResponse)

	case strings.HasPrefix(ref.Definition, "#/components/securitySchemes/"):
		return copyComponent(ctx, ref, prefix, dst.SecuritySchemes, v3.NewSecurityScheme)

	case strings.HasPrefix(ref.Definition, "#/components/examples/"):
		return copyComponent(ctx, ref, prefix, dst.Examples, base.NewExample)

	case strings.HasPrefix(ref.Definition, "#/components/links/"):
		return copyComponent(ctx, ref, prefix, dst.Links, v3.NewLink)

	case strings.HasPrefix(ref.Definition, "#/components/callbacks/"):
		return copyComponent(ctx, ref, prefix, dst.Callbacks, v3.NewCallback)
	}
	return nil
}

func CopyComponentAndRenameRef(ctx context.Context,
	ref *index.Reference,
	prefix string,
	dst *v3.Components,
) (err error) {
	switch {
	case strings.HasPrefix(ref.Definition, "#/components/schemas/"):
		return copySchemaAndRenameRef(ctx, ref, prefix, dst.Schemas)

	case strings.HasPrefix(ref.Definition, "#/components/parameters/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dst.Parameters, v3.NewParameter)

	case strings.HasPrefix(ref.Definition, "#/components/requestBodies/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dst.RequestBodies, v3.NewRequestBody)

	case strings.HasPrefix(ref.Definition, "#/components/headers/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dst.Headers, v3.NewHeader)

	case strings.HasPrefix(ref.Definition, "#/components/responses/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dst.Responses, v3.NewResponse)

	case strings.HasPrefix(ref.Definition, "#/components/securitySchemes/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dst.SecuritySchemes, v3.NewSecurityScheme)

	case strings.HasPrefix(ref.Definition, "#/components/examples/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dst.Examples, base.NewExample)

	case strings.HasPrefix(ref.Definition, "#/components/links/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dst.Links, v3.NewLink)

	case strings.HasPrefix(ref.Definition, "#/components/callbacks/"):
		return copyComponentAndRenameRef(ctx, ref, prefix, dst.Callbacks, v3.NewCallback)
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

func copyComponentAndRenameRef[B any, L low.Buildable[B], H high.GoesLow[L]](
	ctx context.Context,
	ref *index.Reference,
	prefix string,
	dst *orderedmap.Map[string, H],
	fnew func(L) H,
) (err error) {
	if err = copyComponent(ctx, ref, prefix, dst, fnew); err != nil {
		return
	}

	name := prefix + ref.Name
	refname := strings.TrimSuffix(ref.Definition, ref.Name) + name
	ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
	return
}

func copyComponent[B any, L low.Buildable[B], H high.GoesLow[L]](
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
	return
}
