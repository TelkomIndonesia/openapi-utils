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

	doc      libopenapi.Document
	docv3    *libopenapi.DocumentModel[v3.Document]
	proxied  map[*v3.Operation]*ProxyOperation
	upstream map[libopenapi.Document]map[*v3.Operation]map[*ProxyOperation]struct{}
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
	pe.upstream = make(map[libopenapi.Document]map[*v3.Operation]map[*ProxyOperation]struct{})

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
			if _, ok := pe.upstream[doc]; !ok {
				pe.upstream[doc] = map[*v3.Operation]map[*ProxyOperation]struct{}{}
			}
			if _, ok := pe.upstream[doc][uop]; !ok {
				pe.upstream[doc][uop] = map[*ProxyOperation]struct{}{}
			}
			pe.upstream[doc][uop][&pop] = struct{}{}
		}
	}
	return
}
