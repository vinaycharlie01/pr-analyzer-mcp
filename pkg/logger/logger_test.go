package logger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/pkg/logger"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		cfg  logger.Config
	}{
		{name: "json format", cfg: logger.Config{Level: "info", Format: "json", Output: "stdout"}},
		{name: "text format", cfg: logger.Config{Level: "debug", Format: "text", Output: "stdout"}},
		{name: "warn level", cfg: logger.Config{Level: "warn", Format: "json"}},
		{name: "error level", cfg: logger.Config{Level: "error", Format: "json"}},
		{name: "default level", cfg: logger.Config{Level: "", Format: "json"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log := logger.New(tc.cfg)
			require.NotNil(t, log)
		})
	}
}

func TestContextPropagation(t *testing.T) {
	ctx := context.Background()

	ctx = logger.WithRequestID(ctx, "req-123")
	ctx = logger.WithTraceID(ctx, "trace-456")

	log := logger.New(logger.Config{Level: "info", Format: "json"})
	enriched := logger.FromContext(ctx, log)
	require.NotNil(t, enriched)
}

func TestFromContext_EmptyContext(t *testing.T) {
	log := logger.New(logger.Config{Level: "info", Format: "json"})
	result := logger.FromContext(context.Background(), log)
	assert.NotNil(t, result)
}
