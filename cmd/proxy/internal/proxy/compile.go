package proxy

import (
	"context"
	"errors"
	"fmt"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

func CompileByte(ctx context.Context, specPath string) (newspec []byte, doc libopenapi.Document, docv3 *libopenapi.DocumentModel[v3.Document], err error) {
	pe, err := NewProxyExtension(ctx, specPath)
	if err != nil {
		return nil, nil, nil, err
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
			for method, op := range getOperationsMap(m.Value()) {
				if _, ok := opmap[op]; ok {
					continue
				}
				setOperation(pathItem, method, nil)
			}
		}

		// copy components with new prefix
		prefix := firstEntry(firstEntry(uopPopMap).value).key.GetName()
		doc, err := modCopyComponents(ctx, doc, prefix, pe.doc)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to copy components : %w", err)
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
		ud, ok := proxyOperationUpstreamDocs[pop]
		if !ok {
			continue
		}

		// find upstream path operation, with parameter inherited from upstream path
		udv3, _ := ud.BuildV3Model()
		up, ok := udv3.Model.Paths.PathItems.Get(pop.Path)
		if !ok {
			continue
		}
		uop := getOperation(up, pop.Method)
		if uop == nil {
			continue
		}
		var uParams []*v3.Parameter
		injectedParamMap := map[parameterKey]struct{}{}
		for _, p := range pop.Inject.Parameters {
			injectedParamMap[parameterKey{name: p.Name, in: p.In}] = struct{}{}
		}
		for _, p := range copyParameters(up.Parameters, uop.Parameters...) {
			if _, ok := injectedParamMap[parameterKey{name: p.Name, in: p.In}]; ok {
				continue
			}
			uParams = append(uParams, p)
		}

		// copy operation
		opParam := copyParameters(op.Parameters, uParams...)
		opID := op.OperationId
		opSecurity := op.Security
		*op = *uop
		op.Parameters = opParam
		op.OperationId = opID
		op.Security = opSecurity
	}
	by, proxyDoc, proxyDocv3, errs := pe.doc.RenderAndReload()

	return by, proxyDoc, proxyDocv3, errors.Join(errs...)
}
