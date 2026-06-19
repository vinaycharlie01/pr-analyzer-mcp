package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/inbound"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/analysis"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/architecture"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/migration"
	"github.com/vinaycharlie01/pr-analyzer-mcp/pkg/logger"
)

// PRAnalyzerUseCase implements the inbound PRAnalyzerPort interface.
type PRAnalyzerUseCase struct {
	githubVCS    outbound.VCSPort
	bitbucketVCS outbound.VCSPort
	analyzer     *analysis.Analyzer
	planner      *migration.Planner
	archMapper   *architecture.Mapper
	tracer       trace.Tracer
	logger       *slog.Logger
}

func NewPRAnalyzerUseCase(
	githubVCS outbound.VCSPort,
	bitbucketVCS outbound.VCSPort,
	analyzer *analysis.Analyzer,
	planner *migration.Planner,
	archMapper *architecture.Mapper,
	tracer trace.Tracer,
	log *slog.Logger,
) *PRAnalyzerUseCase {
	return &PRAnalyzerUseCase{
		githubVCS:    githubVCS,
		bitbucketVCS: bitbucketVCS,
		analyzer:     analyzer,
		planner:      planner,
		archMapper:   archMapper,
		tracer:       tracer,
		logger:       log,
	}
}

// Ensure interface implementation at compile time.
var _ inbound.PRAnalyzerPort = (*PRAnalyzerUseCase)(nil)

func (uc *PRAnalyzerUseCase) AnalyzePR(ctx context.Context, req inbound.AnalyzePRRequest) (*entity.AnalysisResult, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.AnalyzePR",
		trace.WithAttributes(
			attribute.String("platform", string(req.Platform)),
			attribute.String("repository", req.Repository.String()),
			attribute.Int("pr_number", req.PRNumber.Value()),
		),
	)
	defer span.End()

	log := logger.FromContext(ctx, uc.logger)
	log.InfoContext(ctx, "analyzing pull request",
		slog.String("repository", req.Repository.String()),
		slog.Int("pr_number", req.PRNumber.Value()),
	)

	vcs := uc.selectVCS(req.Platform)

	pr, err := vcs.GetPullRequest(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting pull request: %w", err)
	}

	files, err := vcs.GetPullRequestFiles(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting PR files: %w", err)
	}
	pr.Files = files

	commits, err := vcs.GetPullRequestCommits(ctx, req.Repository, req.PRNumber)
	if err != nil {
		log.WarnContext(ctx, "failed to get commits", slog.String("error", err.Error()))
	} else {
		pr.Commits = commits
	}

	reviews, err := vcs.GetPullRequestReviews(ctx, req.Repository, req.PRNumber)
	if err != nil {
		log.WarnContext(ctx, "failed to get reviews", slog.String("error", err.Error()))
	} else {
		pr.Reviews = reviews
	}

	deps, err := uc.analyzer.AnalyzeDependencies(ctx, files)
	if err != nil {
		log.WarnContext(ctx, "dependency analysis failed", slog.String("error", err.Error()))
	}

	archImpact, err := uc.archMapper.MapImpact(ctx, pr)
	if err != nil {
		log.WarnContext(ctx, "architecture analysis failed", slog.String("error", err.Error()))
	}

	migrationPlan, err := uc.planner.GeneratePlan(ctx, pr, deps)
	if err != nil {
		log.WarnContext(ctx, "migration plan generation failed", slog.String("error", err.Error()))
	}

	result := &entity.AnalysisResult{
		ID:                  fmt.Sprintf("analysis-%s-%d", req.Repository.String(), req.PRNumber.Value()),
		PullRequest:         pr,
		ExecutiveSummary:    uc.buildExecutiveSummary(pr, deps),
		BusinessPurpose:     uc.inferBusinessPurpose(pr),
		TechnicalPurpose:    uc.inferTechnicalPurpose(pr, files),
		FilesChanged:        uc.buildFileAnalysis(files),
		Dependencies:        deps,
		MigrationPlan:       migrationPlan,
		ArchitectureImpact:  archImpact,
		DatabaseImpact:      uc.archMapper.DetectDatabaseImpact(files),
		ConfigurationImpact: uc.archMapper.DetectConfigImpact(files),
		KubernetesImpact:    uc.archMapper.DetectKubernetesImpact(files),
		ValidationSteps:     uc.buildValidationSteps(pr, files),
		CreatedAt:           time.Now(),
	}

	rollback, err := uc.planner.GenerateRollbackStrategy(ctx, pr)
	if err == nil {
		result.RollbackStrategy = rollback
	}

	checklist, err := uc.planner.GenerateChecklist(ctx, pr, result)
	if err == nil {
		result.MigrationChecklist = checklist
	}

	log.InfoContext(ctx, "pull request analysis completed",
		slog.String("repository", req.Repository.String()),
		slog.Int("pr_number", req.PRNumber.Value()),
		slog.Int("files_analyzed", len(files)),
		slog.Int("dependencies_found", len(deps)),
	)

	return result, nil
}

func (uc *PRAnalyzerUseCase) ExplainChange(ctx context.Context, req inbound.ExplainChangeRequest) (*inbound.ChangeExplanation, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.ExplainChange")
	defer span.End()

	vcs := uc.selectVCS(req.Platform)

	pr, err := vcs.GetPullRequest(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting pull request: %w", err)
	}

	files, err := vcs.GetPullRequestFiles(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting PR files: %w", err)
	}

	var targetFile *entity.ChangedFile
	for i, f := range files {
		if f.Path == req.FilePath || strings.HasSuffix(f.Path, req.FilePath) {
			targetFile = &files[i]
			break
		}
	}

	if targetFile == nil {
		return &inbound.ChangeExplanation{
			FilePath: req.FilePath,
			Why:      "File not found in this PR",
			What:     "No changes detected for this file path",
			How:      "Verify the file path is correct",
		}, nil
	}

	return &inbound.ChangeExplanation{
		FilePath: targetFile.Path,
		Why:      uc.inferWhyChanged(pr, targetFile),
		What:     uc.describeWhatChanged(targetFile),
		How:      uc.describeHowToMigrate(targetFile),
		Impact:   uc.assessFileImpact(targetFile),
		Risks: []entity.Risk{
			{
				Level:       entity.RiskLevelLow,
				Description: fmt.Sprintf("File has %d additions and %d deletions", targetFile.Additions, targetFile.Deletions),
				Mitigation:  "Review changes carefully before applying",
			},
		},
	}, nil
}

func (uc *PRAnalyzerUseCase) GenerateMigrationPlan(ctx context.Context, req inbound.MigrationPlanRequest) (*entity.MigrationPlan, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.GenerateMigrationPlan")
	defer span.End()

	vcs := uc.selectVCS(req.SourcePlatform)

	pr, err := vcs.GetPullRequest(ctx, req.SourceRepo, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting pull request: %w", err)
	}

	files, err := vcs.GetPullRequestFiles(ctx, req.SourceRepo, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting files: %w", err)
	}
	pr.Files = files

	deps, _ := uc.analyzer.AnalyzeDependencies(ctx, files)

	return uc.planner.GeneratePlan(ctx, pr, deps)
}

func (uc *PRAnalyzerUseCase) AnalyzeDependencies(ctx context.Context, req inbound.DependencyRequest) ([]entity.Dependency, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.AnalyzeDependencies")
	defer span.End()

	vcs := uc.selectVCS(req.Platform)

	files, err := vcs.GetPullRequestFiles(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting PR files: %w", err)
	}

	return uc.analyzer.AnalyzeDependencies(ctx, files)
}

func (uc *PRAnalyzerUseCase) GenerateArchitectureMap(ctx context.Context, req inbound.ArchitectureRequest) (*entity.ArchitectureImpact, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.GenerateArchitectureMap")
	defer span.End()

	vcs := uc.selectVCS(req.Platform)

	pr, err := vcs.GetPullRequest(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting pull request: %w", err)
	}

	files, err := vcs.GetPullRequestFiles(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting files: %w", err)
	}
	pr.Files = files

	return uc.archMapper.MapImpact(ctx, pr)
}

func (uc *PRAnalyzerUseCase) FindRelatedFiles(ctx context.Context, req inbound.RelatedFilesRequest) ([]string, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.FindRelatedFiles")
	defer span.End()

	vcs := uc.selectVCS(req.Platform)

	files, err := vcs.GetPullRequestFiles(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting files: %w", err)
	}

	return uc.analyzer.FindRelatedFiles(ctx, files, "")
}

func (uc *PRAnalyzerUseCase) FindRequiredDependencies(ctx context.Context, req inbound.RequiredDepsRequest) ([]entity.Dependency, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.FindRequiredDependencies")
	defer span.End()

	vcs := uc.selectVCS(req.Platform)

	files, err := vcs.GetPullRequestFiles(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting files: %w", err)
	}

	deps, err := uc.analyzer.AnalyzeDependencies(ctx, files)
	if err != nil {
		return nil, err
	}

	// Filter to only required dependencies
	var required []entity.Dependency
	for _, d := range deps {
		if d.Required {
			required = append(required, d)
		}
	}
	return required, nil
}

func (uc *PRAnalyzerUseCase) CompareRepositories(ctx context.Context, req inbound.CompareReposRequest) (*inbound.ComparisonResult, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.CompareRepositories")
	defer span.End()

	sourceVCS := uc.selectVCS(req.SourcePlatform)

	comparison, err := sourceVCS.CompareRepositories(ctx, req.SourceRepo, req.TargetRepo)
	if err != nil {
		return nil, fmt.Errorf("comparing repositories: %w", err)
	}

	diffs := make([]inbound.FileDifference, 0, len(comparison.Differences))
	for _, d := range comparison.Differences {
		diffs = append(diffs, inbound.FileDifference{
			Path:        d.Path,
			Description: fmt.Sprintf("Source: %d chars, Target: %d chars", len(d.Source), len(d.Target)),
		})
	}

	return &inbound.ComparisonResult{
		Source:      comparison.SourceRepo,
		Target:      comparison.TargetRepo,
		CommonFiles: comparison.CommonFiles,
		Differences: diffs,
		Suggestions: []string{
			"Review common files for compatibility",
			"Check for conflicting dependencies",
			"Verify module paths are correctly updated",
		},
	}, nil
}

func (uc *PRAnalyzerUseCase) GenerateMigrationChecklist(ctx context.Context, req inbound.ChecklistRequest) ([]entity.ChecklistItem, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.GenerateMigrationChecklist")
	defer span.End()

	analysisResult, err := uc.AnalyzePR(ctx, inbound.AnalyzePRRequest{
		Platform:   req.Platform,
		Repository: req.Repository,
		PRNumber:   req.PRNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("analyzing PR for checklist: %w", err)
	}

	return uc.planner.GenerateChecklist(ctx, analysisResult.PullRequest, analysisResult)
}

func (uc *PRAnalyzerUseCase) ExplainCodeFlow(ctx context.Context, req inbound.CodeFlowRequest) (*inbound.CodeFlowExplanation, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.ExplainCodeFlow")
	defer span.End()

	vcs := uc.selectVCS(req.Platform)

	files, err := vcs.GetPullRequestFiles(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting files: %w", err)
	}

	steps := make([]inbound.FlowStep, 0)
	for i, f := range files {
		if strings.HasSuffix(f.Path, ".go") {
			steps = append(steps, inbound.FlowStep{
				Order:       i + 1,
				Function:    detectEntryFunction(f.Path),
				File:        f.Path,
				Description: fmt.Sprintf("File %s (%s: +%d/-%d)", f.Path, f.Status, f.Additions, f.Deletions),
			})
		}
	}

	return &inbound.CodeFlowExplanation{
		EntryPoint:  req.EntryPoint,
		Flow:        steps,
		Description: fmt.Sprintf("Code flow analysis for PR involving %d Go files", len(steps)),
	}, nil
}

func (uc *PRAnalyzerUseCase) GenerateFeatureSummary(ctx context.Context, req inbound.FeatureSummaryRequest) (*inbound.FeatureSummary, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.GenerateFeatureSummary")
	defer span.End()

	vcs := uc.selectVCS(req.Platform)

	pr, err := vcs.GetPullRequest(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting pull request: %w", err)
	}

	files, err := vcs.GetPullRequestFiles(ctx, req.Repository, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("getting files: %w", err)
	}

	areas := make([]string, 0)
	for _, f := range files {
		if layer := detectLayerFromPath(f.Path); layer != "" {
			areas = appendUnique(areas, layer)
		}
	}

	return &inbound.FeatureSummary{
		Title:           pr.Title,
		BusinessValue:   uc.inferBusinessPurpose(pr),
		TechnicalDetail: uc.inferTechnicalPurpose(pr, files),
		AffectedAreas:   areas,
	}, nil
}

func (uc *PRAnalyzerUseCase) GenerateMigrationDocumentation(ctx context.Context, req inbound.MigrationDocRequest) (*inbound.MigrationDocumentation, error) {
	ctx, span := uc.tracer.Start(ctx, "PRAnalyzerUseCase.GenerateMigrationDocumentation")
	defer span.End()

	analysisResult, err := uc.AnalyzePR(ctx, inbound.AnalyzePRRequest{
		Platform:   req.Platform,
		Repository: req.Repository,
		PRNumber:   req.PRNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("analyzing PR: %w", err)
	}

	pr := analysisResult.PullRequest

	doc := &inbound.MigrationDocumentation{
		Title:             fmt.Sprintf("Migration Guide: %s", pr.Title),
		ExecutiveSummary:  analysisResult.ExecutiveSummary,
		TechnicalOverview: analysisResult.TechnicalPurpose,
		ValidationSteps:   analysisResult.ValidationSteps,
		RollbackPlan:      analysisResult.RollbackStrategy,
		Checklist:         analysisResult.MigrationChecklist,
	}

	if analysisResult.MigrationPlan != nil {
		doc.MigrationSteps = analysisResult.MigrationPlan.Steps
	}

	return doc, nil
}

func (uc *PRAnalyzerUseCase) selectVCS(platform entity.PlatformType) outbound.VCSPort {
	if platform == entity.PlatformBitbucket {
		return uc.bitbucketVCS
	}
	return uc.githubVCS
}

func (uc *PRAnalyzerUseCase) buildExecutiveSummary(pr *entity.PullRequest, deps []entity.Dependency) string {
	return fmt.Sprintf(
		"PR #%d '%s' by %s modifies %d file(s) with %d identified dependencies. "+
			"Source branch: %s -> Target branch: %s.",
		pr.Number, pr.Title, pr.Author.Username,
		len(pr.Files), len(deps),
		pr.SourceBranch, pr.TargetBranch,
	)
}

func (uc *PRAnalyzerUseCase) inferBusinessPurpose(pr *entity.PullRequest) string {
	title := strings.ToLower(pr.Title)
	desc := strings.ToLower(pr.Description)
	combined := title + " " + desc

	switch {
	case containsAny(combined, "fix", "bug", "patch", "hotfix"):
		return "Bug fix — resolves a defect or unexpected behavior in the system."
	case containsAny(combined, "feat", "feature", "add", "new", "implement"):
		return "Feature addition — introduces new functionality to the product."
	case containsAny(combined, "refactor", "clean", "improve", "optimize"):
		return "Code quality improvement — improves maintainability or performance without changing behavior."
	case containsAny(combined, "docs", "documentation", "readme"):
		return "Documentation update — improves developer or user documentation."
	case containsAny(combined, "test", "spec", "coverage"):
		return "Testing improvement — adds or improves automated test coverage."
	case containsAny(combined, "ci", "cd", "deploy", "release", "pipeline"):
		return "CI/CD improvement — enhances build, deployment, or release processes."
	case containsAny(combined, "security", "auth", "permission", "cve"):
		return "Security enhancement — addresses a security concern or vulnerability."
	default:
		return fmt.Sprintf("Change introduced via PR: %s", pr.Title)
	}
}

func (uc *PRAnalyzerUseCase) inferTechnicalPurpose(pr *entity.PullRequest, files []entity.ChangedFile) string {
	var layers []string
	layerSet := make(map[string]bool)
	for _, f := range files {
		if l := detectLayerFromPath(f.Path); l != "" && !layerSet[l] {
			layerSet[l] = true
			layers = append(layers, l)
		}
	}

	goFiles := 0
	testFiles := 0
	configFiles := 0
	for _, f := range files {
		switch {
		case strings.HasSuffix(f.Path, "_test.go"):
			testFiles++
		case strings.HasSuffix(f.Path, ".go"):
			goFiles++
		case strings.HasSuffix(f.Path, ".yaml") || strings.HasSuffix(f.Path, ".yml") || strings.HasSuffix(f.Path, ".json"):
			configFiles++
		}
	}

	return fmt.Sprintf(
		"Modifies %d Go file(s) and %d test file(s) across %s layer(s). %d config file(s) updated.",
		goFiles, testFiles, strings.Join(layers, ", "), configFiles,
	)
}

func (uc *PRAnalyzerUseCase) buildFileAnalysis(files []entity.ChangedFile) []entity.FileAnalysis {
	result := make([]entity.FileAnalysis, 0, len(files))
	for _, f := range files {
		result = append(result, entity.FileAnalysis{
			Path:       f.Path,
			Purpose:    inferFilePurpose(f.Path),
			ChangeType: f.Status,
			Impact:     fmt.Sprintf("+%d/-%d lines", f.Additions, f.Deletions),
		})
	}
	return result
}

func (uc *PRAnalyzerUseCase) buildValidationSteps(pr *entity.PullRequest, files []entity.ChangedFile) []entity.ValidationStep {
	steps := []entity.ValidationStep{
		{Order: 1, Title: "Build verification", Description: "Ensure the project compiles", Command: "go build ./...", Expected: "No compilation errors"},
		{Order: 2, Title: "Unit tests", Description: "Run unit tests", Command: "go test ./...", Expected: "All tests pass"},
		{Order: 3, Title: "Lint check", Description: "Run static analysis", Command: "go vet ./...", Expected: "No lint errors"},
	}

	hasGoFiles := false
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".go") {
			hasGoFiles = true
			break
		}
	}

	if hasGoFiles {
		steps = append(steps, entity.ValidationStep{
			Order:       4,
			Title:       "Race condition check",
			Description: "Run tests with race detector",
			Command:     "go test -race ./...",
			Expected:    "No race conditions detected",
		})
	}

	return steps
}

func (uc *PRAnalyzerUseCase) inferWhyChanged(pr *entity.PullRequest, f *entity.ChangedFile) string {
	return fmt.Sprintf("Changed as part of PR #%d: %s. %s", pr.Number, pr.Title, uc.inferBusinessPurpose(pr))
}

func (uc *PRAnalyzerUseCase) describeWhatChanged(f *entity.ChangedFile) string {
	switch f.Status {
	case entity.FileStatusAdded:
		return fmt.Sprintf("New file added with %d lines", f.Additions)
	case entity.FileStatusDeleted:
		return fmt.Sprintf("File deleted (%d lines removed)", f.Deletions)
	case entity.FileStatusRenamed:
		return fmt.Sprintf("File renamed from %s to %s", f.OldPath, f.Path)
	default:
		return fmt.Sprintf("File modified: +%d/-%d lines", f.Additions, f.Deletions)
	}
}

func (uc *PRAnalyzerUseCase) describeHowToMigrate(f *entity.ChangedFile) string {
	switch f.Status {
	case entity.FileStatusAdded:
		return fmt.Sprintf("Copy file %s to the target repository preserving directory structure", f.Path)
	case entity.FileStatusDeleted:
		return fmt.Sprintf("Remove file %s from the target repository", f.Path)
	case entity.FileStatusRenamed:
		return fmt.Sprintf("Rename %s to %s in the target repository", f.OldPath, f.Path)
	default:
		return fmt.Sprintf("Apply the patch changes to %s in the target repository", f.Path)
	}
}

func (uc *PRAnalyzerUseCase) assessFileImpact(f *entity.ChangedFile) string {
	totalLines := f.Additions + f.Deletions
	switch {
	case totalLines > 200:
		return "High impact — significant changes, review carefully"
	case totalLines > 50:
		return "Medium impact — moderate changes"
	default:
		return "Low impact — minor changes"
	}
}

func detectLayerFromPath(path string) string {
	switch {
	case strings.Contains(path, "/domain/"):
		return "domain"
	case strings.Contains(path, "/application/"):
		return "application"
	case strings.Contains(path, "/adapters/"):
		return "adapter"
	case strings.Contains(path, "/port/"):
		return "port"
	case strings.Contains(path, "/service/"):
		return "service"
	case strings.Contains(path, "/cmd/"):
		return "presentation"
	case strings.Contains(path, "/repository/"):
		return "repository"
	case strings.Contains(path, "/pkg/"):
		return "shared"
	default:
		return ""
	}
}

func inferFilePurpose(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, "_test.go"):
		return "Test file"
	case strings.Contains(lower, "handler"):
		return "HTTP or MCP request handler"
	case strings.Contains(lower, "repository"):
		return "Data access layer"
	case strings.Contains(lower, "usecase"):
		return "Business logic use case"
	case strings.Contains(lower, "entity"):
		return "Domain entity"
	case strings.Contains(lower, "config"):
		return "Configuration"
	case strings.Contains(lower, "migration"):
		return "Database migration"
	case strings.HasSuffix(lower, ".go"):
		return "Go source file"
	default:
		return "Source file"
	}
}

func detectEntryFunction(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "main"):
		return "main()"
	case strings.Contains(lower, "handler"):
		return "Handle()"
	case strings.Contains(lower, "usecase"):
		return "Execute()"
	case strings.Contains(lower, "service"):
		return "Run()"
	default:
		return "New()"
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
