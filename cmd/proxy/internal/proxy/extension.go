package proxy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/internal/util"
)

type ProxyExtension struct {
	specPath string
	specDir  string

	doc            libopenapi.Document
	docv3          *libopenapi.DocumentModel[v3.Document]
	proxied        map[*v3.Operation]*ProxyOperation
	loadedUpstream map[libopenapi.Document]map[*v3.Operation]map[*ProxyOperation]struct{}
	upstream       map[*ProxyOperation]libopenapi.Document
}

func NewProxyExtension(ctx context.Context, specPath string) (pe ProxyExtension, err error) {
	pe.specPath = specPath
	pe.specDir, err = filepath.Abs(filepath.Dir(specPath))
	if err != nil {
		return pe, fmt.Errorf("fail to determine spec file base directory: %w", err)
	}

	if err = pe.loadDoc(); err != nil {
		return
	}
	if err = pe.loadProxied(ctx); err != nil {
		return
	}
	if err = pe.pruneAndPrefixUpstreamDocs(ctx); err != nil {
		return
	}
	pe.updateProxied()

	return
}

func (pe *ProxyExtension) loadDoc() (err error) {
	specBytes, err := os.ReadFile(pe.specPath)
	if err != nil {
		return fmt.Errorf("fail to read spec file: %w", err)
	}

	doc, err := libopenapi.NewDocument([]byte(specBytes))
	if err != nil {
		return fmt.Errorf("failed to create openapi document: %w", err)
	}
	docv3, errs := doc.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return fmt.Errorf("failed to create openapi v3 document: %w", err)
	}

	pe.doc, pe.docv3 = doc, docv3
	return
}

func (pe *ProxyExtension) loadProxied(ctx context.Context) (err error) {
	pe.proxied = map[*v3.Operation]*ProxyOperation{}
	pe.loadedUpstream = make(map[libopenapi.Document]map[*v3.Operation]map[*ProxyOperation]struct{})

	proxies := map[string]*Proxy{}
	if pe.docv3.Model.Components.Extensions != nil {
		ex, ok := pe.docv3.Model.Components.Extensions.Get("x-proxy")
		if ok {
			if err = ex.Decode(proxies); err != nil {
				return fmt.Errorf("fail to decode `x-proxy` component :%w", err)
			}

			for k, v := range proxies {
				v.Name = k
				v.Spec = path.Join(pe.specDir, v.Spec)
			}
		}
	}

	for m := range orderedmap.Iterate(ctx, pe.docv3.Model.Paths.PathItems) {
		for _, op := range util.GetOperationsMap(m.Value()) {
			if op.Extensions == nil {
				continue
			}
			ex, ok := op.Extensions.Get("x-proxy")
			if !ok {
				continue
			}

			var pop ProxyOperation
			if err = ex.Decode(&pop); err != nil {
				return fmt.Errorf("fail to decode Proxy Operation : %w", err)
			}
			if pop.Spec == "" && pop.Proxy != nil && pop.Proxy.Name != "" {
				pop.Proxy, ok = proxies[pop.Name]
				if !ok {
					return fmt.Errorf("invalid proxy definition for %s: no spec is provided", pop.Proxy.Name)
				}
			} else {
				pop.Spec = path.Join(pe.specDir, pop.Spec)
			}

			doc, err := pop.GetOpenAPIDoc()
			if err != nil {
				return fmt.Errorf("fail to load upstream openapi spec: %w", err)
			}
			uop, err := pop.GetUpstreamOperation()
			if err != nil {
				return fmt.Errorf("fail to find upstream operation: %w", err)
			}

			pe.proxied[op] = &pop
			if _, ok := pe.loadedUpstream[doc]; !ok {
				pe.loadedUpstream[doc] = map[*v3.Operation]map[*ProxyOperation]struct{}{}
			}
			if _, ok := pe.loadedUpstream[doc][uop]; !ok {
				pe.loadedUpstream[doc][uop] = map[*ProxyOperation]struct{}{}
			}
			pe.loadedUpstream[doc][uop][&pop] = struct{}{}
		}
	}
	return
}

func (pe *ProxyExtension) pruneAndPrefixUpstreamDocs(ctx context.Context) (err error) {
	pe.upstream = map[*ProxyOperation]libopenapi.Document{}
	for doc, uopPopMap := range pe.loadedUpstream {
		docv3, _ := doc.BuildV3Model()
		prefix := util.MapFirstEntry(util.MapFirstEntry(uopPopMap).Value).Key.GetName()

		// delete unused operation
		opmap := map[*v3.Operation]struct{}{}
		for k := range uopPopMap {
			opmap[k] = struct{}{}
		}
		for m := range orderedmap.Iterate(ctx, docv3.Model.Paths.PathItems) {
			pathItem := m.Value()
			for method, op := range util.GetOperationsMap(m.Value()) {
				if _, ok := opmap[op]; ok {
					continue
				}
				util.SetOperation(pathItem, method, nil)
			}
		}

		// recreate the doc so that we could get references of used operations only
		// also add components with prefix so that it doesn't trigger error log from libopenapi
		components := util.NewStubComponents()
		err := components.CopyComponents(docv3, "")
		if err != nil {
			return fmt.Errorf("fail to copy components: %w", err)
		}
		err = components.CopyComponents(docv3, prefix)
		if err != nil {
			return fmt.Errorf("fail to copy components with prefix: %w", err)
		}
		_, doc, docv3, err = components.RenderAndReload(doc)
		if err != nil {
			return fmt.Errorf("fail to render and reload upstream doc: %w", err)
		}

		// rerender with prefixed added to all components
		components = util.NewStubComponents()
		err = components.CopyLocalizedComponents(docv3, prefix)
		if err != nil {
			return fmt.Errorf("fail to copy components with prefix: %w", err)
		}
		_, doc, docv3, err = components.RenderAndReload(doc)
		if err != nil {
			return fmt.Errorf("fail to render and reload upstream doc: %w", err)
		}

		// store it
		for _, popmap := range uopPopMap {
			for pop := range popmap {
				pe.upstream[pop] = doc
			}
		}
	}
	return
}

func (pe *ProxyExtension) updateProxied() {
	for _, pop := range pe.proxied {
		doc, ok := pe.upstream[pop]
		if !ok {
			continue
		}
		*pop = pop.WithReloadedDoc(doc)
	}
}

func (pe *ProxyExtension) Doc() libopenapi.Document {
	return pe.doc
}

func (pe *ProxyExtension) DocV3() *libopenapi.DocumentModel[v3.Document] {
	return pe.docv3
}

func (pe *ProxyExtension) Proxied() map[*v3.Operation]*ProxyOperation {
	return pe.proxied
}

func (pe *ProxyExtension) Upstream() map[*ProxyOperation]libopenapi.Document {
	return pe.upstream
}

func (pe *ProxyExtension) CreateProxyDoc() (b []byte, ndoc libopenapi.Document, docv3 *libopenapi.DocumentModel[v3.Document], err error) {
	// compile proxy document
	for op, pop := range pe.Proxied() {
		uop, err := pop.GetUpstreamOperation()
		if err != nil {
			return nil, nil, nil, err
		}
		params, err := pop.GetProxiedParameters()
		if err != nil {
			return nil, nil, nil, err
		}

		// copy operation
		opParam := util.CopyParameters(op.Parameters, params...)
		opID := op.OperationId
		opSecurity := op.Security
		opExt := op.Extensions
		*op = *uop
		op.Parameters = opParam
		op.OperationId = opID
		op.Security = opSecurity
		for m := range orderedmap.Iterate(context.Background(), op.Extensions) {
			opExt.Set(m.Key(), m.Value())
		}
		op.Extensions = opExt
	}

	components := util.NewStubComponents()
	docs := map[*libopenapi.DocumentModel[v3.Document]]struct{}{}
	for _, doc := range pe.Upstream() {
		docv3, _ := doc.BuildV3Model()
		docs[docv3] = struct{}{}
	}
	for docv3 := range docs {
		err := components.CopyLocalizedComponents(docv3, "")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to copy localized components: %w", err)
		}
	}

	err = components.CopyComponents(pe.docv3, "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to copy components on proxy doc: %w", err)
	}
	return components.RenderAndReload(pe.Doc())
}
