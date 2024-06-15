package proxy

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	err := Generate(context.Background(), "./testdata/spec-proxy.yml")
	require.NoError(t, err)

	cmd := exec.Command("go", "test", ".", "-v")
	cmd.Dir = "testgen"
	out, err := cmd.Output()
	require.NoError(t, err)
	t.Log(string(out))
	assert.Equal(t, 0, cmd.ProcessState.ExitCode())
}
