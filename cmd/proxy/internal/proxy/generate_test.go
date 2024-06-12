package proxy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	err := Generate(context.Background(), "./testdata/spec-proxy.yml")
	require.NoError(t, err)
}
