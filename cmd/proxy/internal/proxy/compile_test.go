package proxy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompile(t *testing.T) {
	src := "./testdata/spec-proxy.yml"

	by, _, err := CompileByte(context.Background(), src)
	require.NoError(t, err)
	t.Log("new\n", string(by))
}
