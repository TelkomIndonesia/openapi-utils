package proxy

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/cmd/proxy/internal/proxy/config"
)

func CompileByte(ctx context.Context, specBytes []byte, specDir string) (newspec []byte, doc libopenapi.Document, docv3 *libopenapi.DocumentModel[v3.Document], err error) {
	proxyDoc, err := libopenapi.NewDocument([]byte(specBytes))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create openapi document: %w", err)
	}
	proxyDocv3, errs := proxyDoc.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create openapi v3 document: %w", err)
	}
	initComponents(proxyDocv3)

	// build the proxy
	proxies := map[string]*config.Proxy{}
	if proxyDocv3.Model.Components.Extensions != nil {
		ex, ok := proxyDocv3.Model.Components.Extensions.Get("x-proxy")
		if ok {
			if err = ex.Decode(proxies); err != nil {
				return nil, nil, nil, fmt.Errorf("fail to decode `x-proxy` component :%w", err)
			}
			for k, v := range proxies {
				v.Name = k
				v.Spec = path.Join(specDir, v.Spec)
			}
		}
	}
	proxyOperations := map[*config.ProxyOperation]*v3.Operation{}
	upstreamDocs := map[libopenapi.Document]map[*config.ProxyOperation]*v3.Operation{}
	for m := range orderedmap.Iterate(ctx, proxyDocv3.Model.Paths.PathItems) {
		for _, op := range getOperationsMap(m.Value()) {
			if op.Extensions == nil {
				continue
			}

			ex, ok := op.Extensions.Get("x-proxy")
			if !ok {
				continue
			}

			var pop config.ProxyOperation
			err = ex.Decode(&pop)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("fail to decode Proxy Operation : %w", err)
			}

			if pop.Spec == "" && pop.Proxy != nil && pop.Proxy.Name != "" {
				pop.Proxy, ok = proxies[pop.Name]
				if !ok {
					return nil, nil, nil, fmt.Errorf("invalid proxy definition for %s: no spec is provided", pop.Proxy.Name)
				}
			} else {
				pop.Spec = path.Join(specDir, pop.Spec)
			}

			doc, err := pop.GetOpenAPIDoc()
			if err != nil {
				return nil, nil, nil, fmt.Errorf("fail to load `x-proxy` :%w", err)
			}
			if _, ok := upstreamDocs[doc]; !ok {
				upstreamDocs[doc] = map[*config.ProxyOperation]*v3.Operation{}
			}
			docv3, _ := doc.BuildV3Model()
			val, ok := docv3.Model.Paths.PathItems.Get(pop.Path)
			if !ok {
				continue
			}
			uop := getOperation(val, pop.Method)
			if uop == nil {
				continue
			}

			proxyOperations[&pop] = op
			upstreamDocs[doc][&pop] = uop
		}
	}

	// copy components to proxy doc
	proxyOperationUpstreamDocs := map[*config.ProxyOperation]libopenapi.Document{}
	for doc, popmap := range upstreamDocs {
		docV3, _ := doc.BuildV3Model()

		// delete unused operation
		opmap := map[*v3.Operation]struct{}{}
		for _, v := range popmap {
			opmap[v] = struct{}{}
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
		prefix := firstKey(popmap).GetName()
		doc, err := copyComponents(ctx, doc, prefix, proxyDoc)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to copy components : %w", err)
		}

		// store the result
		for pop := range popmap {
			proxyOperationUpstreamDocs[pop] = doc
		}
	}

	// compile proxy document
	for pop, op := range proxyOperations {
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
	by, proxyDoc, proxyDocv3, errs := proxyDoc.RenderAndReload()

	return by, proxyDoc, proxyDocv3, errors.Join(errs...)
}
