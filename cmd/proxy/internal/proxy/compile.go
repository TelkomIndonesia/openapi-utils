package proxy

import (
	"context"
	"errors"
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

	// copy components to proxy doc
	proxyOperationUpstreamDocs := map[*ProxyOperation]libopenapi.Document{}
	for doc, uopPopMap := range pe.upstream {
		docV3, _ := doc.BuildV3Model()

		// delete unused operation
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

		// copy components with new prefix
		prefix := util.MapFirstEntry(util.MapFirstEntry(uopPopMap).Value).Key.GetName()
		doc, err := util.CopyDocComponentsAndRenameRef(ctx, doc, prefix, pe.doc)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to copy components : %w", err)
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
	newspec, doc, _, errs := pe.doc.RenderAndReload()
	err = errors.Join(errs...)
	return
}
