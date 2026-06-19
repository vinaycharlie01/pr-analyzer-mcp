//go:build wireinject
// +build wireinject

package config

import (
	"context"

	"github.com/google/wire"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/bitbucket"
	githubadapter "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/github"
	mcpadapter "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/mcp"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/application/usecase"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/analysis"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/architecture"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/migration"
	pkgconfig "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/config"
	"github.com/vinaycharlie01/pr-analyzer-mcp/pkg/logger"
	"github.com/vinaycharlie01/pr-analyzer-mcp/pkg/observability"
)

var AnalysisSet = wire.NewSet(
	analysis.NewAnalyzer,
	migration.NewPlanner,
	architecture.NewMapper,
)

func InitializeMCPServer(ctx context.Context, cfg *pkgconfig.Config) (*mcpadapter.Server, func(), error) {
	wire.Build(
		// Logger
		provideLogger,
		// Observability
		provideObservabilityProvider,
		provideTracer,
		// GitHub client
		provideGitHubConfig,
		githubadapter.NewClient,
		provideGitHubVCS,
		// Bitbucket client
		provideBitbucketConfig,
		bitbucket.NewClient,
		provideBitbucketVCS,
		// Analysis services
		AnalysisSet,
		// Use case
		usecase.NewPRAnalyzerUseCase,
		// MCP server
		mcpadapter.NewServer,
	)
	return nil, nil, nil
}
