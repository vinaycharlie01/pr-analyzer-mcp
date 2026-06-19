package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/bitbucket"
	githubadapter "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/github"
	mcpadapter "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/mcp"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/application/usecase"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/analysis"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/architecture"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/migration"
	pkgconfig "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/config"
	pkglogger "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/logger"
	"github.com/vinaycharlie01/pr-analyzer-mcp/pkg/observability"
)

var (
	configPath string
	rootCmd    = &cobra.Command{
		Use:   "pr-analyzer-mcp",
		Short: "PR Analyzer MCP Server — analyze PRs and generate migration plans",
		Long: `PR Analyzer MCP Server

A Model Context Protocol (MCP) server that analyzes pull requests from GitHub
and Bitbucket, providing comprehensive migration assistance for senior engineers.

Tools available:
  - analyze_pr                       Full PR analysis with executive summary
  - explain_change                   Explain why/what/how a file changed
  - generate_migration_plan          Step-by-step migration plan
  - dependency_analysis              Identify all PR dependencies
  - architecture_map                 Architectural layer impact map
  - find_related_files               Files related to PR changes
  - find_required_dependencies       Required dependencies for migration
  - compare_repositories             Source vs target repository diff
  - generate_migration_checklist     Migration safety checklist
  - explain_code_flow                Code execution flow analysis
  - generate_feature_summary         Business and technical feature summary
  - generate_migration_documentation Full migration documentation`,
		RunE: runServer,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (default: ./configs/config.yaml)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := pkgconfig.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	log := pkglogger.New(pkglogger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	})

	log.InfoContext(ctx, "starting PR Analyzer MCP server",
		slog.String("mode", cfg.Server.Mode),
		slog.Int("port", cfg.Server.Port),
	)

	otelProvider, err := observability.NewProvider(ctx, observability.Config{
		ServiceName: cfg.Observability.ServiceName,
		Enabled:     cfg.Observability.Enabled,
	})
	if err != nil {
		return fmt.Errorf("initializing observability: %w", err)
	}
	defer func() {
		if err := otelProvider.Shutdown(context.Background()); err != nil {
			log.Error("shutting down observability", slog.String("error", err.Error()))
		}
	}()

	tracer := otelProvider.Tracer()

	var githubVCS *githubadapter.Client
	if cfg.GitHub.Token != "" {
		githubVCS, err = githubadapter.NewClient(githubadapter.Config{
			Token:   cfg.GitHub.Token,
			BaseURL: cfg.GitHub.BaseURL,
		}, log)
		if err != nil {
			log.WarnContext(ctx, "GitHub client initialization failed", slog.String("error", err.Error()))
		}
	} else {
		log.WarnContext(ctx, "GITHUB_TOKEN not set — GitHub integration disabled")
	}

	var bbVCS *bitbucket.Client
	if cfg.Bitbucket.Token != "" || cfg.Bitbucket.AppPassword != "" {
		bbVCS, err = bitbucket.NewClient(bitbucket.Config{
			BaseURL:       cfg.Bitbucket.BaseURL,
			DatacenterURL: cfg.Bitbucket.DatacenterURL,
			Token:         cfg.Bitbucket.Token,
			Username:      cfg.Bitbucket.Username,
			AppPassword:   cfg.Bitbucket.AppPassword,
		}, log)
		if err != nil {
			log.WarnContext(ctx, "Bitbucket client initialization failed", slog.String("error", err.Error()))
		}
	} else {
		log.WarnContext(ctx, "Bitbucket credentials not set — Bitbucket integration disabled")
	}

	// Fall back to a noop VCS if clients are not configured
	var ghPort, bbPort = noopVCS{}, noopVCS{}
	if githubVCS != nil {
		ghPort = noopVCS{delegate: githubVCS}
	}
	if bbVCS != nil {
		bbPort = noopVCS{delegate: bbVCS}
	}

	analyzer := analysis.NewAnalyzer(log)
	planner := migration.NewPlanner(log)
	archMapper := architecture.NewMapper(log)

	uc := usecase.NewPRAnalyzerUseCase(
		ghPort,
		bbPort,
		analyzer,
		planner,
		archMapper,
		tracer,
		log,
	)

	mcpServer := mcpadapter.NewServer(uc, log)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.InfoContext(ctx, "received shutdown signal")
		cancel()
	}()

	log.InfoContext(ctx, "MCP server ready — listening on stdio")
	return mcpServer.ServeStdio(ctx)
}
