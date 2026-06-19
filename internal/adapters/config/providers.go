package config

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	bitbucketadapter "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/bitbucket"
	githubadapter "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/github"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
	pkgconfig "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/config"
	pkglogger "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/logger"
	"github.com/vinaycharlie01/pr-analyzer-mcp/pkg/observability"
)

func provideLogger(cfg *pkgconfig.Config) *slog.Logger {
	return pkglogger.New(pkglogger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	})
}

func provideObservabilityProvider(ctx context.Context, cfg *pkgconfig.Config, log *slog.Logger) (*observability.Provider, func(), error) {
	provider, err := observability.NewProvider(ctx, observability.Config{
		ServiceName: cfg.Observability.ServiceName,
		Enabled:     cfg.Observability.Enabled,
	})
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			log.Error("shutting down observability provider", slog.String("error", err.Error()))
		}
	}
	return provider, cleanup, nil
}

func provideTracer(provider *observability.Provider) trace.Tracer {
	return provider.Tracer()
}

func provideGitHubConfig(cfg *pkgconfig.Config) githubadapter.Config {
	return githubadapter.Config{
		Token:   cfg.GitHub.Token,
		BaseURL: cfg.GitHub.BaseURL,
	}
}

func provideGitHubVCS(client *githubadapter.Client) (outbound.VCSPort, error) {
	return client, nil
}

func provideBitbucketConfig(cfg *pkgconfig.Config) bitbucketadapter.Config {
	return bitbucketadapter.Config{
		BaseURL:       cfg.Bitbucket.BaseURL,
		DatacenterURL: cfg.Bitbucket.DatacenterURL,
		Token:         cfg.Bitbucket.Token,
		Username:      cfg.Bitbucket.Username,
		AppPassword:   cfg.Bitbucket.AppPassword,
	}
}

func provideBitbucketVCS(client *bitbucketadapter.Client) (outbound.VCSPort, error) {
	return client, nil
}
