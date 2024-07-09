package middleware_test

import (
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/middleware"
	"testing"
)

func TestMakeTracingOptions(t *testing.T) {
	opts := middleware.MakeTracingOptions(middleware.WithExcludedPaths([]string{"/test", "/TEST", "/test1"}))

	require.NotEmpty(t, opts)
	require.NotEmpty(t, opts.ExcludedPaths)
	require.Len(t, opts.ExcludedPaths, 2)
	require.Contains(t, opts.ExcludedPaths, "/test")
	require.Contains(t, opts.ExcludedPaths, "/test1")
}
