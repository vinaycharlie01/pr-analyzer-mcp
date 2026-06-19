package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bitbucketadapter "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/bitbucket"
	githubadapter "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/github"
	pkgconfig "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/config"
)

func testConfig() *pkgconfig.Config {
	cfg, _ := pkgconfig.Load("")
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
	cfg.Observability.ServiceName = "test"
	cfg.Observability.Enabled = false
	cfg.GitHub.Token = "test-token"
	cfg.Bitbucket.Token = "test-token"
	return cfg
}

func TestProvideLogger(t *testing.T) {
	cfg := testConfig()
	log := provideLogger(cfg)
	assert.NotNil(t, log)
}

func TestProvideObservabilityProvider(t *testing.T) {
	cfg := testConfig()
	log := provideLogger(cfg)
	provider, cleanup, err := provideObservabilityProvider(context.Background(), cfg, log)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, cleanup)
	cleanup()
}

func TestProvideTracer(t *testing.T) {
	cfg := testConfig()
	log := provideLogger(cfg)
	provider, cleanup, err := provideObservabilityProvider(context.Background(), cfg, log)
	require.NoError(t, err)
	defer cleanup()

	tracer := provideTracer(provider)
	assert.NotNil(t, tracer)
}

func TestProvideGitHubConfig(t *testing.T) {
	cfg := testConfig()
	ghCfg := provideGitHubConfig(cfg)
	assert.Equal(t, "test-token", ghCfg.Token)
}

func TestProvideGitHubVCS(t *testing.T) {
	cfg := testConfig()
	log := provideLogger(cfg)
	ghCfg := provideGitHubConfig(cfg)
	client, err := githubadapter.NewClient(ghCfg, log)
	require.NoError(t, err)
	vcs, err := provideGitHubVCS(client)
	require.NoError(t, err)
	assert.NotNil(t, vcs)
}

func TestProvideBitbucketConfig(t *testing.T) {
	cfg := testConfig()
	bbCfg := provideBitbucketConfig(cfg)
	assert.Equal(t, "test-token", bbCfg.Token)
}

func TestProvideBitbucketVCS(t *testing.T) {
	cfg := testConfig()
	log := provideLogger(cfg)
	bbCfg := provideBitbucketConfig(cfg)
	client, err := bitbucketadapter.NewClient(bbCfg, log)
	require.NoError(t, err)
	vcs, err := provideBitbucketVCS(client)
	require.NoError(t, err)
	assert.NotNil(t, vcs)
}
