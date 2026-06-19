package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/pkg/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Pass an empty path so viper uses search paths and produces a ConfigFileNotFoundError,
	// which Load silently ignores, returning a config populated with defaults only.
	cfg, err := config.Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, "stdio", cfg.Server.Mode)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, 10, cfg.Analysis.MaxDepth)
	assert.Equal(t, 30, cfg.Analysis.TimeoutSeconds)
	assert.True(t, cfg.Analysis.Architecture)
	assert.True(t, cfg.Analysis.DependencyAnalysis)
	assert.True(t, cfg.Analysis.MigrationAnalysis)
	assert.True(t, cfg.Observability.Enabled)
	assert.Equal(t, "pr-analyzer-mcp", cfg.Observability.ServiceName)
}

func TestLoad_FromFile(t *testing.T) {
	cfg, err := config.Load("../../configs/config.yaml")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
}
