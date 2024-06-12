package proxy

import (
	"context"

	"github.com/pb33f/libopenapi"
)

func Compile(ctx context.Context, specPath string) (newspec []byte, doc libopenapi.Document, err error) {
	pe, err := NewProxyExtension(ctx, specPath)
	if err != nil {
		return nil, nil, err
	}

	newspec, doc, _, err = pe.CreateProxyDoc()
	return
}
