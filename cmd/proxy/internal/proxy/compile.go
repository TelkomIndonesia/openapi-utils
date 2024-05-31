package proxy

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	baselow "github.com/pb33f/libopenapi/datamodel/low/base"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/cmd/proxy/internal/proxy/config"
)

type parameterKey struct {
	name string
	in   string
}

func CompileByte(ctx context.Context, specBytes []byte, specDir string) (newspec []byte, doc libopenapi.Document, docv3 *libopenapi.DocumentModel[v3.Document], err error) {
	proxyDoc, err := libopenapi.NewDocument([]byte(specBytes))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create openapi document: %w", err)
	}
	proxyDocv3, errs := proxyDoc.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create openapi v3 document: %w", err)
	}
	if proxyDocv3.Model.Components == nil {
		proxyDocv3.Model.Components = &v3.Components{}
	}
	if proxyDocv3.Model.Components.Schemas == nil {
		proxyDocv3.Model.Components.Schemas = &orderedmap.Map[string, *base.SchemaProxy]{}
	}
	if proxyDocv3.Model.Components.Responses == nil {
		proxyDocv3.Model.Components.Responses = &orderedmap.Map[string, *v3.Response]{}
	}

	// build the proxy
	proxies := map[string]*config.Proxy{}
	if proxyDocv3.Model.Components.Extensions != nil {
		ex, ok := proxyDocv3.Model.Components.Extensions.Get("x-proxy")
		if ok {
			if err = ex.Decode(proxies); err != nil {
				return nil, nil, nil, fmt.Errorf("fail to decode `x-proxy` component :%w", err)
			}
			for k, v := range proxies {
				v.Name = &k
				v.Spec = path.Join(specDir, v.Spec)
			}
		}
	}
	upstreamDocsOri := map[libopenapi.Document]map[*config.ProxyOperation]*v3.Operation{}
	proxyOperations := map[*v3.Operation]*config.ProxyOperation{}
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

			if pop.Spec == "" && pop.Proxy != nil && pop.Proxy.Name != nil {
				pop.Proxy, ok = proxies[*pop.Name]
				if !ok {
					return nil, nil, nil, fmt.Errorf("invalid proxy definition for %s: no spec is provided", *pop.Proxy.Name)
				}
			} else {
				pop.Spec = path.Join(specDir, pop.Spec)
			}

			proxyOperations[op] = &pop
			doc, err := pop.GetOpenAPIDoc()
			if err != nil {
				return nil, nil, nil, fmt.Errorf("fail to load `x-proxy` :%w", err)
			}
			if _, ok := upstreamDocsOri[doc]; !ok {
				upstreamDocsOri[doc] = map[*config.ProxyOperation]*v3.Operation{}
			}
			docv3, _ := doc.BuildV3Model()
			val, ok := docv3.Model.Paths.PathItems.Get(pop.Path)
			if !ok {
				continue
			}
			op := getOperation(val, pop.Method)
			if op == nil {
				continue
			}
			upstreamDocsOri[doc][&pop] = op
		}
	}

	// rerender upstream document to contains only used operation
	upstreamDocs := map[libopenapi.Document]string{}
	proxyOperationUpstreamDocs := map[*config.ProxyOperation]libopenapi.Document{}
	for doc, popmap := range upstreamDocsOri {
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

		// rerender
		_, doc, docV3, errs = doc.RenderAndReload()
		if err := errors.Join(errs...); err != nil {
			return nil, nil, nil, fmt.Errorf("faill to render and reload openapi doc: %w", err)
		}

		for pop := range popmap {
			upstreamDocs[doc] = pop.GetName()
			proxyOperationUpstreamDocs[pop] = doc
		}
	}

	// attach prefix to all components and copy them to proxy document
	for doc, docName := range upstreamDocs {
		docV3, _ := doc.BuildV3Model()

		// create duplicated schemas with prefix
		schemas := map[string]string{}
		for _, ref := range docV3.Index.GetRawReferencesSequenced() {
			switch {
			case strings.HasPrefix(ref.Definition, "#/components/schemas"):
				schema := &baselow.Schema{}
				err = schema.Build(context.Background(), ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to recreate schema: %w", err)
				}

				name := docName + ref.Name
				refname := "#/components/schemas/" + name
				schemas[ref.Definition] = refname
				docV3.Model.Components.Schemas.Set(name, base.CreateSchemaProxy(base.NewSchema(schema)))
			}
		}
		_, ndoc, _, errs := doc.RenderAndReload()
		if err := errors.Join(errs...); err != nil {
			return nil, nil, nil, fmt.Errorf("fail to render and reload: %w", err)
		}

		// copy components
		ndocV3, _ := ndoc.BuildV3Model()
		for _, ref := range ndocV3.Index.GetRawReferencesSequenced() {
			switch {
			case strings.HasPrefix(ref.Definition, "#/components/schemas"):
				if _, ok := schemas[ref.Definition]; !ok {
					continue
				}

				schema := &baselow.Schema{}
				err = schema.Build(context.Background(), ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to recreate schema: %w", err)
				}
				hschema := base.NewSchema(schema)

				name := docName + ref.Name
				refname := "#/components/schemas/" + name

				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.Schemas.Set(name, base.CreateSchemaProxy(hschema))

			case strings.HasPrefix(ref.Definition, "#/components/responses"):
				n, _, err, _ := low.LocateRefNodeWithContext(ctx, ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to find repsonse node: %w", err)
				}

				lres := &v3low.Response{}
				lres.Build(ctx, nil, n, ref.Index)
				res := v3.NewResponse(lres)
				for v := range orderedmap.Iterate(ctx, ndocV3.Model.Components.Responses) {
					if v.Value() == nil || v.Value().GoLow() == nil || v.Value().GoLow().RootNode != n {
						continue
					}
					res.Description = v.Value().Description
				}

				name := docName + ref.Name
				refname := "#/components/responses/" + name

				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				ndocV3.Model.Components.Responses.Set(name, res)
				proxyDocv3.Model.Components.Responses.Set(name, res)
			}
		}
		_, ndoc, _, errs = ndoc.RenderAndReload()
		if err := errors.Join(errs...); err != nil {
			return nil, nil, nil, fmt.Errorf("fail to render and reload: %w", err)
		}

		for pop, pdoc := range proxyOperationUpstreamDocs {
			if doc != pdoc {
				continue
			}
			proxyOperationUpstreamDocs[pop] = ndoc
		}
	}

	// compile proxy document
	for op, pop := range proxyOperations {
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
		uopParams := map[parameterKey]struct{}{}
		for _, p := range uop.Parameters {
			uopParams[parameterKey{name: p.Name, in: p.In}] = struct{}{}
		}
		for _, p := range up.Parameters {
			if _, ok := uopParams[parameterKey{name: p.Name, in: p.In}]; ok {
				continue
			}
			uop.Parameters = append(uop.Parameters, p)
		}

		// copy operation
		opParam := op.Parameters
		opID := op.OperationId

		opParamMap := map[parameterKey]struct{}{}
		for _, p := range op.Parameters {
			opParamMap[parameterKey{name: p.Name, in: p.In}] = struct{}{}
		}
		injectedParamMap := map[parameterKey]struct{}{}
		for _, p := range pop.Inject.Parameters {
			injectedParamMap[parameterKey{name: p.Name, in: p.In}] = struct{}{}
		}
		for _, p := range uop.Parameters {
			if _, ok := opParamMap[parameterKey{name: p.Name, in: p.In}]; ok {
				continue
			}
			if _, ok := injectedParamMap[parameterKey{name: p.Name, in: p.In}]; ok {
				continue
			}
			opParam = append(opParam, p)
		}
		*op = *uop

		op.Parameters = opParam
		op.OperationId = opID
	}
	by, proxyDoc, proxyDocv3, errs := proxyDoc.RenderAndReload()
	return by, proxyDoc, proxyDocv3, errors.Join(errs...)
}

func getOperationsMap(pi *v3.PathItem) (ops map[string]*v3.Operation) {
	ops = map[string]*v3.Operation{}
	if pi.Get != nil {
		ops["Get"] = pi.Get
	}
	if pi.Delete != nil {
		ops["Delete"] = pi.Delete
	}
	if pi.Post != nil {
		ops["Post"] = pi.Post
	}
	if pi.Patch != nil {
		ops["Patch"] = pi.Patch
	}
	if pi.Options != nil {
		ops["Options"] = pi.Options
	}
	if pi.Head != nil {
		ops["Head"] = pi.Head
	}
	if pi.Trace != nil {
		ops["Trace"] = pi.Trace
	}
	return
}

func getOperation(pi *v3.PathItem, method string) *v3.Operation {
	switch {
	case strings.EqualFold("Get", method):
		return pi.Get
	case strings.EqualFold("Delete", method):
		return pi.Delete
	case strings.EqualFold("Post", method):
		return pi.Post
	case strings.EqualFold("Patch", method):
		return pi.Patch
	case strings.EqualFold("Options", method):
		return pi.Options
	case strings.EqualFold("Head", method):
		return pi.Head
	case strings.EqualFold("Trace", method):
		return pi.Trace
	}
	return nil
}

func setOperation(pi *v3.PathItem, method string, val *v3.Operation) {
	switch {
	case strings.EqualFold("Get", method):
		pi.Get = val
	case strings.EqualFold("Delete", method):
		pi.Delete = val
	case strings.EqualFold("Post", method):
		pi.Post = val
	case strings.EqualFold("Patch", method):
		pi.Patch = val
	case strings.EqualFold("Options", method):
		pi.Options = val
	case strings.EqualFold("Head", method):
		pi.Head = val
	case strings.EqualFold("Trace", method):
		pi.Trace = val
	}
}
