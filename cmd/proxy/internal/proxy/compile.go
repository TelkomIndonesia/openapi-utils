package proxy

import (
	"context"
	"fmt"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/internal/util"
)

func CompileByte(ctx context.Context, specPath string) (newspec []byte, doc libopenapi.Document, err error) {
	pe, err := NewProxyExtension(ctx, specPath)
	if err != nil {
		return nil, nil, err
	}

	components := util.NewStubComponents()
	// copy components to proxy doc
	proxyOperationUpstreamDocs := map[*ProxyOperation]libopenapi.Document{}
	for doc, uopPopMap := range pe.upstream {
		docV3, _ := doc.BuildV3Model()
		prefix := util.MapFirstEntry(util.MapFirstEntry(uopPopMap).Value).Key.GetName()

		// delete unused operation
		allcomponents := util.NewStubComponents()
		err := allcomponents.CopyLocalizedComponents(docV3, "")
		if err != nil {
			return nil, nil, fmt.Errorf("fail to copy and localized all components: %w", err)
		}
		opmap := map[*v3.Operation]struct{}{}
		for k := range uopPopMap {
			opmap[k] = struct{}{}
		}
		for m := range orderedmap.Iterate(ctx, docV3.Model.Paths.PathItems) {
			pathItem := m.Value()
			for method, op := range util.GetOperationsMap(m.Value()) {
				if _, ok := opmap[op]; ok {
					continue
				}
				util.SetOperation(pathItem, method, nil)
			}
		}
		_, doc, docV3, err = allcomponents.RenderAndReload(doc)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to render and reload upstream doc: %w", err)
		}

		// copy components with new prefix
		err = components.CopyLocalizedComponents(docV3, prefix)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to copy and rename localized components: %w", err)
		}

		// store the new upstream doc
		for _, popmap := range uopPopMap {
			for pop := range popmap {
				proxyOperationUpstreamDocs[pop] = doc
			}
		}
	}

	// compile proxy document
	for op, pop := range pe.proxied {
		doc, ok := proxyOperationUpstreamDocs[pop]
		if !ok {
			continue
		}
		*pop = pop.WithReloadedDoc(doc)
		uop, err := pop.GetUpstreamOperation()
		if err != nil {
			return nil, nil, err
		}
		params, err := pop.GetProxiedParameters()
		if err != nil {
			return nil, nil, err
		}

		// copy operation
		opParam := util.CopyParameters(op.Parameters, params...)
		opID := op.OperationId
		opSecurity := op.Security
		*op = *uop
		op.Parameters = opParam
		op.OperationId = opID
		op.Security = opSecurity
	}

	err = components.CopyLocalizedComponents(pe.docv3, "")
	if err != nil {
		return nil, nil, fmt.Errorf("fail to copy components on proxy doc: %w", err)
	}
	newspec, doc, _, err = components.RenderAndReload(pe.doc)
	return
}
