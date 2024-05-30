package proxy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompile(t *testing.T) {
	src := "./testdata/spec-proxy.yml"
	sf, _ := filepath.Abs(src)
	specDir, _ := filepath.Abs(filepath.Dir(src))
	specBytes, _ := os.ReadFile(sf)

	t.Log("original\n", string(specBytes))

	by, _, _, err := CompileByte(context.Background(), specBytes, specDir)
	require.NoError(t, err)
	t.Log("new\n", string(by))
}
