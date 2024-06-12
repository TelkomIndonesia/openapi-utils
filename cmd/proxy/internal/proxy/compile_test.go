package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestCompile(t *testing.T) {
	src := "./testdata/spec-proxy.yml"
	bytes, _, err := Compile(context.Background(), src)
	require.NoError(t, err)
	doc, err := libopenapi.NewDocument(bytes)
	require.NoError(t, err)
	_, errs := doc.BuildV3Model()
	require.NoError(t, errors.Join(errs...))
}
