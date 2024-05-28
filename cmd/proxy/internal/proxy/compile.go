package proxy

import (
	"context"
	"errors"
	"fmt"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/cmd/proxy/internal/proxy/config"
)

func CompileByte(ctx context.Context, specBytes []byte) (newspec []byte, err error) {
	doc, err := libopenapi.NewDocument([]byte(specBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create openapi document: %w", err)
	}
	model, errs := doc.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("failed to create openapi v3 document: %w", err)
	}

	proxies := map[string]*config.Proxy{}
	ex, ok := model.Model.Extensions.Get("x-proxy")
	if ok {
		err = ex.Decode(proxies)
		if err != nil {
			return nil, fmt.Errorf("fail to decode `x-proxy` component :%w", err)
		}
		for k, v := range proxies {
			v.Name = &k
			if _, err = v.GetOpenAPIModel(); err != nil {
				return nil, fmt.Errorf("fail to load `x-proxy` :%w", err)
			}
		}
	}

	proxiesOp := map[*v3.Operation]*config.ProxyOperation{}
	for m := range orderedmap.Iterate(ctx, model.Model.Paths.PathItems) {
		for _, op := range getOperations(m.Value()) {
			ex, ok := op.Extensions.Get("x-proxy")
			if !ok {
				continue
			}
			var pop config.ProxyOperation
			err = ex.Decode(&pop)
			if err != nil {
				return nil, fmt.Errorf("fail to decode Proxy Operation : %w", err)
			}
			if pop.Spec == "" && pop.Proxy != nil && pop.Proxy.Name != nil {
				pop.Proxy = proxies[*pop.Name]
			}
			proxiesOp[op] = &pop
		}
	}
	return
}

func getOperations(pi *v3.PathItem) (ops []*v3.Operation) {
	ops = append(ops, pi.Get)
	ops = append(ops, pi.Delete)
	ops = append(ops, pi.Post)
	ops = append(ops, pi.Patch)
	ops = append(ops, pi.Options)
	ops = append(ops, pi.Head)
	ops = append(ops, pi.Trace)
	return
}
