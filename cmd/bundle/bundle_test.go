package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBundle(t *testing.T) {
	bytes, err := bundleFile("./testdata/profile/profile.yml")
	require.NoError(t, err)
	t.Log(string(bytes))
}
