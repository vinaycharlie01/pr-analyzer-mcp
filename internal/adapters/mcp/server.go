package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/inbound"
)

// Server wraps the MCP server with all 12 PR analysis tools.
type Server struct {
	mcpServer *server.MCPServer
	analyzer  inbound.PRAnalyzerPort
	logger    *slog.Logger
}

func NewServer(analyzer inbound.PRAnalyzerPort, logger *slog.Logger) *Server {
	s := &Server{
		analyzer: analyzer,
		logger:   logger,
	}

	mcpSrv := server.NewMCPServer(
		"PR Analyzer MCP",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.mcpServer = mcpSrv
	s.registerTools()
	return s
}

func (s *Server) ServeStdio(ctx context.Context) error {
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	s.mcpServer.AddTool(s.analyzePRTool())
	s.mcpServer.AddTool(s.explainChangeTool())
	s.mcpServer.AddTool(s.generateMigrationPlanTool())
	s.mcpServer.AddTool(s.dependencyAnalysisTool())
	s.mcpServer.AddTool(s.architectureMapTool())
	s.mcpServer.AddTool(s.findRelatedFilesTool())
	s.mcpServer.AddTool(s.findRequiredDependenciesTool())
	s.mcpServer.AddTool(s.compareRepositoriesTool())
	s.mcpServer.AddTool(s.generateMigrationChecklistTool())
	s.mcpServer.AddTool(s.explainCodeFlowTool())
	s.mcpServer.AddTool(s.generateFeatureSummaryTool())
	s.mcpServer.AddTool(s.generateMigrationDocumentationTool())
}

// --- helpers for parsing common arguments ---

func parsePlatformRepoNumber(req mcp.CallToolRequest) (entity.PlatformType, valueobject.RepositoryRef, valueobject.PRNumber, error) {
	platform := req.GetString("platform", "github")
	repoStr := req.GetString("repository", "")
	prNum := req.GetFloat("pr_number", 0)

	repo, err := valueobject.ParseRepositoryRef(repoStr)
	if err != nil {
		return "", valueobject.RepositoryRef{}, valueobject.PRNumber{}, fmt.Errorf("invalid repository: %w", err)
	}
	prNumber, err := valueobject.NewPRNumber(int(prNum))
	if err != nil {
		return "", valueobject.RepositoryRef{}, valueobject.PRNumber{}, fmt.Errorf("invalid PR number: %w", err)
	}
	return entity.PlatformType(platform), repo, prNumber, nil
}

// --- Tool: analyze_pr ---

func (s *Server) analyzePRTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("analyze_pr",
		mcp.WithDescription("Perform a comprehensive analysis of a pull request. Returns executive summary, business purpose, technical purpose, files changed, dependencies, migration plan, architecture impact, database impact, configuration impact, Kubernetes impact, validation steps, and rollback strategy."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format (e.g. 'myorg/myrepo')")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		result, err := s.analyzer.AnalyzePR(ctx, inbound.AnalyzePRRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("analysis failed: %v", err)), nil
		}

		return successResult(formatAnalysisResult(result)), nil
	}

	return tool, handler
}

// --- Tool: explain_change ---

func (s *Server) explainChangeTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("explain_change",
		mcp.WithDescription("Explain WHY a specific file was changed in a PR, WHAT changed, HOW to migrate it, and WHAT can break."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Path of the file to explain")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		filePath := req.GetString("file_path", "")

		result, err := s.analyzer.ExplainChange(ctx, inbound.ExplainChangeRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
			FilePath:   filePath,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("explain change failed: %v", err)), nil
		}

		return successResult(formatChangeExplanation(result)), nil
	}

	return tool, handler
}

// --- Tool: generate_migration_plan ---

func (s *Server) generateMigrationPlanTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("generate_migration_plan",
		mcp.WithDescription("Generate a step-by-step migration plan to port changes from a PR to a target repository. Includes required dependencies, file copy instructions, import path updates, and validation commands."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("Source VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("source_repository", mcp.Required(), mcp.Description("Source repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
		mcp.WithString("target_repository", mcp.Required(), mcp.Description("Target repository in owner/name format")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform := entity.PlatformType(req.GetString("platform", "github"))
		sourceRepoStr := req.GetString("source_repository", "")
		prNum := req.GetFloat("pr_number", 0)
		targetRepoStr := req.GetString("target_repository", "")

		sourceRepo, err := valueobject.ParseRepositoryRef(sourceRepoStr)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid source repository: %v", err)), nil
		}
		targetRepo, err := valueobject.ParseRepositoryRef(targetRepoStr)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid target repository: %v", err)), nil
		}
		prNumber, err := valueobject.NewPRNumber(int(prNum))
		if err != nil {
			return errorResult(fmt.Sprintf("invalid PR number: %v", err)), nil
		}

		plan, err := s.analyzer.GenerateMigrationPlan(ctx, inbound.MigrationPlanRequest{
			SourcePlatform: platform,
			SourceRepo:     sourceRepo,
			PRNumber:       prNumber,
			TargetRepo:     targetRepo,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("migration plan generation failed: %v", err)), nil
		}

		return successResult(formatMigrationPlan(plan)), nil
	}

	return tool, handler
}

// --- Tool: dependency_analysis ---

func (s *Server) dependencyAnalysisTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("dependency_analysis",
		mcp.WithDescription("Analyze all dependencies introduced or modified by a PR. Returns package dependencies, service dependencies, database dependencies, and API dependencies with descriptions."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		deps, err := s.analyzer.AnalyzeDependencies(ctx, inbound.DependencyRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("dependency analysis failed: %v", err)), nil
		}

		return successResult(formatDependencies(deps)), nil
	}

	return tool, handler
}

// --- Tool: architecture_map ---

func (s *Server) architectureMapTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("architecture_map",
		mcp.WithDescription("Generate an architecture impact map for a PR showing which layers, patterns, and components are affected."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		impact, err := s.analyzer.GenerateArchitectureMap(ctx, inbound.ArchitectureRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("architecture map failed: %v", err)), nil
		}

		return successResult(formatArchitectureImpact(impact)), nil
	}

	return tool, handler
}

// --- Tool: find_related_files ---

func (s *Server) findRelatedFilesTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("find_related_files",
		mcp.WithDescription("Find files related to the PR changes that may also need to be migrated (tests, interfaces, config files, etc.)."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		files, err := s.analyzer.FindRelatedFiles(ctx, inbound.RelatedFilesRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("find related files failed: %v", err)), nil
		}

		out := "# Related Files\n\nThe following files may need to be considered during migration:\n\n"
		for i, f := range files {
			out += fmt.Sprintf("%d. `%s`\n", i+1, f)
		}
		if len(files) == 0 {
			out += "No additional related files detected.\n"
		}
		return successResult(out), nil
	}

	return tool, handler
}

// --- Tool: find_required_dependencies ---

func (s *Server) findRequiredDependenciesTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("find_required_dependencies",
		mcp.WithDescription("Find all required dependencies that must be present in the target repository for the migration to succeed."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		deps, err := s.analyzer.FindRequiredDependencies(ctx, inbound.RequiredDepsRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("find required dependencies failed: %v", err)), nil
		}

		return successResult("# Required Dependencies\n\n" + formatDependencies(deps)), nil
	}

	return tool, handler
}

// --- Tool: compare_repositories ---

func (s *Server) compareRepositoriesTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("compare_repositories",
		mcp.WithDescription("Compare a source and target repository to identify differences, common files, and migration suggestions."),
		mcp.WithString("source_platform", mcp.Required(), mcp.Description("Source platform: 'github' or 'bitbucket'")),
		mcp.WithString("source_repository", mcp.Required(), mcp.Description("Source repository in owner/name format")),
		mcp.WithString("target_platform", mcp.Required(), mcp.Description("Target platform: 'github' or 'bitbucket'")),
		mcp.WithString("target_repository", mcp.Required(), mcp.Description("Target repository in owner/name format")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sourcePlatform := req.GetString("source_platform", "github")
		sourceRepoStr := req.GetString("source_repository", "")
		targetPlatform := req.GetString("target_platform", "github")
		targetRepoStr := req.GetString("target_repository", "")

		sourceRepo, err := valueobject.ParseRepositoryRef(sourceRepoStr)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid source repository: %v", err)), nil
		}
		targetRepo, err := valueobject.ParseRepositoryRef(targetRepoStr)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid target repository: %v", err)), nil
		}

		result, err := s.analyzer.CompareRepositories(ctx, inbound.CompareReposRequest{
			SourcePlatform: entity.PlatformType(sourcePlatform),
			SourceRepo:     sourceRepo,
			TargetPlatform: entity.PlatformType(targetPlatform),
			TargetRepo:     targetRepo,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("repository comparison failed: %v", err)), nil
		}

		out := fmt.Sprintf("# Repository Comparison\n\n**Source:** %s (%s)\n**Target:** %s (%s)\n\n",
			result.Source.FullName, result.Source.Platform,
			result.Target.FullName, result.Target.Platform)
		if len(result.CommonFiles) > 0 {
			out += fmt.Sprintf("## Common Files (%d)\n", len(result.CommonFiles))
			for _, f := range result.CommonFiles {
				out += fmt.Sprintf("- `%s`\n", f)
			}
			out += "\n"
		}
		if len(result.Differences) > 0 {
			out += fmt.Sprintf("## Differences (%d)\n", len(result.Differences))
			for _, d := range result.Differences {
				out += fmt.Sprintf("- `%s`: %s\n", d.Path, d.Description)
			}
			out += "\n"
		}
		if len(result.Suggestions) > 0 {
			out += "## Suggestions\n"
			for _, sug := range result.Suggestions {
				out += fmt.Sprintf("- %s\n", sug)
			}
		}
		return successResult(out), nil
	}

	return tool, handler
}

// --- Tool: generate_migration_checklist ---

func (s *Server) generateMigrationChecklistTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("generate_migration_checklist",
		mcp.WithDescription("Generate a comprehensive migration checklist for safely porting PR changes to a target repository."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		checklist, err := s.analyzer.GenerateMigrationChecklist(ctx, inbound.ChecklistRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("checklist generation failed: %v", err)), nil
		}

		out := "# Migration Checklist\n\n"
		categories := make(map[string][]entity.ChecklistItem)
		for _, item := range checklist {
			categories[item.Category] = append(categories[item.Category], item)
		}
		for cat, items := range categories {
			out += fmt.Sprintf("## %s\n", cat)
			for _, item := range items {
				required := ""
				if item.Required {
					required = " *(required)*"
				}
				out += fmt.Sprintf("- [ ] **[%s]** %s%s\n  > %s\n", item.ID, item.Title, required, item.Description)
			}
			out += "\n"
		}
		return successResult(out), nil
	}

	return tool, handler
}

// --- Tool: explain_code_flow ---

func (s *Server) explainCodeFlowTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("explain_code_flow",
		mcp.WithDescription("Explain the code flow and execution path introduced by PR changes, from entry point through all modified components."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
		mcp.WithString("entry_point", mcp.Description("Optional entry point function or file to trace flow from")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		entryPoint := req.GetString("entry_point", "")

		flow, err := s.analyzer.ExplainCodeFlow(ctx, inbound.CodeFlowRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
			EntryPoint: entryPoint,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("code flow explanation failed: %v", err)), nil
		}

		out := fmt.Sprintf("# Code Flow Analysis\n\n**Entry Point:** %s\n\n%s\n\n## Flow Steps\n\n",
			flow.EntryPoint, flow.Description)
		for _, step := range flow.Flow {
			out += fmt.Sprintf("%d. **%s** (`%s`)\n   %s\n\n", step.Order, step.Function, step.File, step.Description)
		}
		if len(flow.Flow) == 0 {
			out += "No Go files detected in this PR.\n"
		}
		return successResult(out), nil
	}

	return tool, handler
}

// --- Tool: generate_feature_summary ---

func (s *Server) generateFeatureSummaryTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("generate_feature_summary",
		mcp.WithDescription("Generate a business and technical feature summary for a PR, explaining what the feature does and its value."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		summary, err := s.analyzer.GenerateFeatureSummary(ctx, inbound.FeatureSummaryRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("feature summary failed: %v", err)), nil
		}

		out := fmt.Sprintf("# Feature Summary: %s\n\n", summary.Title)
		out += fmt.Sprintf("## Business Value\n%s\n\n", summary.BusinessValue)
		out += fmt.Sprintf("## Technical Details\n%s\n\n", summary.TechnicalDetail)
		if len(summary.AffectedAreas) > 0 {
			out += "## Affected Areas\n"
			for _, area := range summary.AffectedAreas {
				out += fmt.Sprintf("- %s\n", area)
			}
		}
		return successResult(out), nil
	}

	return tool, handler
}

// --- Tool: generate_migration_documentation ---

func (s *Server) generateMigrationDocumentationTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("generate_migration_documentation",
		mcp.WithDescription("Generate comprehensive migration documentation for porting PR changes, suitable for a senior engineer handoff document."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("VCS platform: 'github' or 'bitbucket'")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository in owner/name format")),
		mcp.WithNumber("pr_number", mcp.Required(), mcp.Description("Pull request number")),
		mcp.WithString("target_repository", mcp.Description("Optional target repository for the migration (owner/name format)")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		platform, repo, prNumber, err := parsePlatformRepoNumber(req)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		targetRepoStr := req.GetString("target_repository", "")

		var targetRepo valueobject.RepositoryRef
		if targetRepoStr != "" {
			targetRepo, err = valueobject.ParseRepositoryRef(targetRepoStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid target repository: %v", err)), nil
			}
		}

		doc, err := s.analyzer.GenerateMigrationDocumentation(ctx, inbound.MigrationDocRequest{
			Platform:   platform,
			Repository: repo,
			PRNumber:   prNumber,
			TargetRepo: targetRepo,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("documentation generation failed: %v", err)), nil
		}

		out := fmt.Sprintf("# %s\n\n", doc.Title)
		out += fmt.Sprintf("## Executive Summary\n%s\n\n", doc.ExecutiveSummary)
		out += fmt.Sprintf("## Technical Overview\n%s\n\n", doc.TechnicalOverview)

		if len(doc.MigrationSteps) > 0 {
			out += "## Migration Steps\n\n"
			for _, step := range doc.MigrationSteps {
				out += fmt.Sprintf("### Step %d: %s\n%s\n\n", step.Order, step.Title, step.Description)
				if len(step.Commands) > 0 {
					out += "```bash\n"
					for _, cmd := range step.Commands {
						out += cmd + "\n"
					}
					out += "```\n\n"
				}
				if step.Validation != "" {
					out += fmt.Sprintf("**Validation:** %s\n\n", step.Validation)
				}
				if step.Rollback != "" {
					out += fmt.Sprintf("**Rollback:** %s\n\n", step.Rollback)
				}
			}
		}

		if len(doc.Checklist) > 0 {
			out += "## Migration Checklist\n\n"
			for _, item := range doc.Checklist {
				out += fmt.Sprintf("- [ ] [%s] %s\n", item.ID, item.Title)
			}
			out += "\n"
		}

		if len(doc.ValidationSteps) > 0 {
			out += "## Validation Steps\n\n"
			for _, step := range doc.ValidationSteps {
				out += fmt.Sprintf("%d. **%s**: `%s`\n", step.Order, step.Title, step.Command)
				out += fmt.Sprintf("   Expected: %s\n\n", step.Expected)
			}
		}

		if doc.RollbackPlan != nil {
			out += fmt.Sprintf("## Rollback Plan\n%s\n\n", doc.RollbackPlan.Description)
			for i, step := range doc.RollbackPlan.Steps {
				out += fmt.Sprintf("%d. %s\n", i+1, step)
			}
		}

		return successResult(out), nil
	}

	return tool, handler
}

// --- Format helpers ---

func formatAnalysisResult(r *entity.AnalysisResult) string {
	out := fmt.Sprintf("# PR Analysis: %s\n\n", r.PullRequest.Title)
	out += fmt.Sprintf("## Executive Summary\n%s\n\n", r.ExecutiveSummary)
	out += fmt.Sprintf("## Business Purpose\n%s\n\n", r.BusinessPurpose)
	out += fmt.Sprintf("## Technical Purpose\n%s\n\n", r.TechnicalPurpose)

	out += fmt.Sprintf("## Files Changed (%d)\n\n", len(r.FilesChanged))
	for _, f := range r.FilesChanged {
		out += fmt.Sprintf("- `%s` (%s) — %s\n", f.Path, f.ChangeType, f.Impact)
	}
	out += "\n"

	if len(r.Dependencies) > 0 {
		out += fmt.Sprintf("## Dependencies (%d)\n\n", len(r.Dependencies))
		out += formatDependencies(r.Dependencies)
	}

	if r.ArchitectureImpact != nil {
		out += "## Architecture Impact\n\n"
		out += formatArchitectureImpact(r.ArchitectureImpact)
	}

	if r.DatabaseImpact != nil {
		out += fmt.Sprintf("## Database Impact\n%s\n\n", r.DatabaseImpact.Description)
	}

	if r.ConfigurationImpact != nil {
		out += fmt.Sprintf("## Configuration Impact\n%s\n\n", r.ConfigurationImpact.Description)
	}

	if r.KubernetesImpact != nil {
		out += fmt.Sprintf("## Kubernetes Impact\n%s\n\n", r.KubernetesImpact.Description)
	}

	if len(r.ValidationSteps) > 0 {
		out += "## Validation Steps\n\n"
		for _, step := range r.ValidationSteps {
			out += fmt.Sprintf("%d. **%s**: `%s`\n", step.Order, step.Title, step.Command)
		}
		out += "\n"
	}

	if r.RollbackStrategy != nil {
		out += fmt.Sprintf("## Rollback Strategy\n%s\n\n", r.RollbackStrategy.Description)
		for i, step := range r.RollbackStrategy.Steps {
			out += fmt.Sprintf("%d. %s\n", i+1, step)
		}
	}

	return out
}

func formatChangeExplanation(e *inbound.ChangeExplanation) string {
	out := fmt.Sprintf("# Change Explanation: `%s`\n\n", e.FilePath)
	out += fmt.Sprintf("## WHY was this changed?\n%s\n\n", e.Why)
	out += fmt.Sprintf("## WHAT changed?\n%s\n\n", e.What)
	out += fmt.Sprintf("## HOW to migrate it?\n%s\n\n", e.How)
	if e.Impact != "" {
		out += fmt.Sprintf("## Impact\n%s\n\n", e.Impact)
	}
	if len(e.Risks) > 0 {
		out += "## Risks\n"
		for _, r := range e.Risks {
			out += fmt.Sprintf("- **[%s]** %s\n  *Mitigation:* %s\n", r.Level, r.Description, r.Mitigation)
		}
	}
	return out
}

func formatMigrationPlan(p *entity.MigrationPlan) string {
	out := fmt.Sprintf("# %s\n\n%s\n\n", p.Title, p.Description)
	out += fmt.Sprintf("**Effort:** %s | **Timeline:** %s\n\n", p.Effort, p.Timeline)

	if len(p.Risks) > 0 {
		out += "## Risks\n\n"
		for _, r := range p.Risks {
			out += fmt.Sprintf("- **[%s]** %s\n  *Mitigation:* %s\n", r.Level, r.Description, r.Mitigation)
		}
		out += "\n"
	}

	out += fmt.Sprintf("## Steps (%d)\n\n", len(p.Steps))
	for _, step := range p.Steps {
		out += fmt.Sprintf("### Step %d: %s\n%s\n\n", step.Order, step.Title, step.Description)
		if len(step.Commands) > 0 {
			out += "```bash\n"
			for _, cmd := range step.Commands {
				out += cmd + "\n"
			}
			out += "```\n\n"
		}
		if step.Validation != "" {
			out += fmt.Sprintf("Validation: %s\n\n", step.Validation)
		}
		if step.Rollback != "" {
			out += fmt.Sprintf("Rollback: %s\n\n", step.Rollback)
		}
	}
	return out
}

func formatDependencies(deps []entity.Dependency) string {
	if len(deps) == 0 {
		return "No dependencies detected.\n"
	}
	out := ""
	for _, d := range deps {
		required := ""
		if d.Required {
			required = " *(required)*"
		}
		out += fmt.Sprintf("- **%s** [%s]%s\n  %s\n", d.Name, d.Type, required, d.Description)
	}
	return out + "\n"
}

func formatArchitectureImpact(impact *entity.ArchitectureImpact) string {
	out := fmt.Sprintf("%s\n\n", impact.Description)
	if len(impact.LayersAffected) > 0 {
		out += fmt.Sprintf("**Layers:** %s\n\n", joinStrings(impact.LayersAffected))
	}
	if len(impact.PatternsUsed) > 0 {
		out += fmt.Sprintf("**Patterns:** %s\n\n", joinStrings(impact.PatternsUsed))
	}
	if len(impact.NewComponents) > 0 {
		out += fmt.Sprintf("**New Components (%d):**\n", len(impact.NewComponents))
		for _, c := range impact.NewComponents {
			out += fmt.Sprintf("  - `%s`\n", c)
		}
		out += "\n"
	}
	if len(impact.ModifiedComponents) > 0 {
		out += fmt.Sprintf("**Modified Components (%d):**\n", len(impact.ModifiedComponents))
		for _, c := range impact.ModifiedComponents {
			out += fmt.Sprintf("  - `%s`\n", c)
		}
		out += "\n"
	}
	return out
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

func successResult(content string) *mcp.CallToolResult {
	return mcp.NewToolResultText(content)
}

func errorResult(msg string) *mcp.CallToolResult {
	return mcp.NewToolResultError(msg)
}
