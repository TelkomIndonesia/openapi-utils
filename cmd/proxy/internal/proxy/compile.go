package proxy

import (
	"context"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/internal/util"
)

func CompileByte(ctx context.Context, specPath string) (newspec []byte, doc libopenapi.Document, err error) {
	pe, err := NewProxyExtension(ctx, specPath)
	if err != nil {
		return nil, nil, err
	}

	// compile proxy document
	for op, pop := range pe.Proxied() {
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

	newspec, doc, _, err = pe.RenderAndReload()
	return
}
