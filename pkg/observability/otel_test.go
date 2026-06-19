package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/pkg/observability"
)

func TestNewProvider_Disabled(t *testing.T) {
	ctx := context.Background()
	p, err := observability.NewProvider(ctx, observability.Config{
		ServiceName: "test-service",
		Enabled:     false,
	})
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.Tracer())
	assert.NotNil(t, p.Meter())
}

func TestNewProvider_Enabled(t *testing.T) {
	ctx := context.Background()
	p, err := observability.NewProvider(ctx, observability.Config{
		ServiceName: "test-service",
		Enabled:     true,
	})
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.Tracer())
	assert.NotNil(t, p.Meter())

	err = p.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestProvider_Shutdown_Nil(t *testing.T) {
	ctx := context.Background()
	p, err := observability.NewProvider(ctx, observability.Config{
		ServiceName: "test",
		Enabled:     false,
	})
	require.NoError(t, err)
	// Shutdown on a noop provider should not error
	err = p.Shutdown(ctx)
	assert.NoError(t, err)
}
