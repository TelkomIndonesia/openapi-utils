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
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

func CopyComponentsAndRenameRefs(ctx context.Context, src libopenapi.Document, prefix string, dst libopenapi.Document) (nsrc libopenapi.Document, err error) {
	// duplicate components on source doc with added prefix and rerender
	srcv3, errs := src.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("fail to build src v3 model: %w", err)
	}
	for _, ref := range srcv3.Index.GetRawReferencesSequenced() {
		CopyComponent(ctx, ref, prefix, srcv3.Model.Components)
	}
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

func CopyComponentAndRenameRef(
	ctx context.Context,
	src *index.Reference,
	prefix string,
	dst *v3.Components,
) (err error) {
	err = CopyComponent(ctx, src, prefix, dst)
	if err != nil {
		return
	}

	name := prefix + src.Name
	refname := strings.TrimSuffix(src.Definition, src.Name) + name
	src.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
	return nil
}

func CopyComponent(
	ctx context.Context,
	src *index.Reference,
	prefix string,
	dst *v3.Components,
) (err error) {
	switch {
	case strings.HasPrefix(src.Definition, "#/components/schemas/"):
		return copySchema(ctx, src, prefix, dst.Schemas)

	case strings.HasPrefix(src.Definition, "#/components/parameters/"):
		return copyComponent(ctx, src, prefix, dst.Parameters, v3.NewParameter)

	case strings.HasPrefix(src.Definition, "#/components/requestBodies/"):
		return copyComponent(ctx, src, prefix, dst.RequestBodies, v3.NewRequestBody)

	case strings.HasPrefix(src.Definition, "#/components/headers/"):
		return copyComponent(ctx, src, prefix, dst.Headers, v3.NewHeader)

	case strings.HasPrefix(src.Definition, "#/components/responses/"):
		return copyComponent(ctx, src, prefix, dst.Responses, v3.NewResponse)

	case strings.HasPrefix(src.Definition, "#/components/securitySchemes/"):
		return copyComponent(ctx, src, prefix, dst.SecuritySchemes, v3.NewSecurityScheme)

	case strings.HasPrefix(src.Definition, "#/components/examples/"):
		return copyComponent(ctx, src, prefix, dst.Examples, base.NewExample)

	case strings.HasPrefix(src.Definition, "#/components/links/"):
		return copyComponent(ctx, src, prefix, dst.Links, v3.NewLink)

	case strings.HasPrefix(src.Definition, "#/components/callbacks/"):
		return copyComponent(ctx, src, prefix, dst.Callbacks, v3.NewCallback)
	}
	return nil
}

func copySchema(
	ctx context.Context,
	src *index.Reference,
	prefix string,
	dst *orderedmap.Map[string, *base.SchemaProxy],
) (err error) {
	schemaProxy, err := baselow.ExtractSchema(ctx, src.Node, src.Index)
	if err != nil {
		return fmt.Errorf("fail to recreate schema: %w", err)
	}

	name := prefix + src.Name
	dst.Set(name, base.NewSchemaProxy(schemaProxy))
	return
}

func copyComponent[B any, L low.Buildable[B], H high.GoesLow[L]](
	ctx context.Context,
	src *index.Reference,
	prefix string,
	dst *orderedmap.Map[string, H],
	fnew func(L) H,
) (err error) {
	node, index := src.Node, src.Index
	// loop till we find non-reference node
	// this will result in partial inline
	for {
		n, i, err, _ := low.LocateRefNodeWithContext(ctx, node, index)
		if err != nil {
			return err
		}
		if ok, _, _ := utils.IsNodeRefValue(n); !ok {
			break
		}
		node, index = n, i
	}

	v, err := low.ExtractObject[L](ctx, "", node, index)
	if err != nil {
		return fmt.Errorf("fail to extract object: %w", err)
	}
	err = v.Value.Build(ctx, v.KeyNode, v.ValueNode, src.Index)
	if err != nil {
		return fmt.Errorf("fail to build object: %w", err)
	}

	name := prefix + src.Name
	dst.Set(name, fnew(v.Value))
	return
}
