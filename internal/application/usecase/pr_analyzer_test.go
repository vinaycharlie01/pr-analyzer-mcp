package usecase_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/application/usecase"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/inbound"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/analysis"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/architecture"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/migration"
)

// stubVCS implements outbound.VCSPort returning deterministic test data.
type stubVCS struct{}

func (s stubVCS) GetPullRequest(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) (*entity.PullRequest, error) {
	return &entity.PullRequest{
		ID:           "1",
		Number:       number.Value(),
		Title:        "Add new feature",
		Description:  "This PR adds an important feature",
		Platform:     entity.PlatformGitHub,
		SourceBranch: "feature/new-thing",
		TargetBranch: "main",
		Status:       entity.PRStatusOpen,
		Author:       entity.Author{Username: "dev", Name: "Dev User"},
		Repository:   entity.Repository{Name: repo.Name(), Owner: repo.Owner(), Platform: entity.PlatformGitHub},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (s stubVCS) ListPullRequests(ctx context.Context, repo valueobject.RepositoryRef, opts outbound.ListPROptions) ([]*entity.PullRequest, error) {
	return []*entity.PullRequest{}, nil
}

func (s stubVCS) GetPullRequestFiles(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.ChangedFile, error) {
	return []entity.ChangedFile{
		{Path: "internal/domain/entity/user.go", Status: entity.FileStatusAdded, Additions: 50},
		{Path: "internal/service/user.go", Status: entity.FileStatusModified, Additions: 20, Deletions: 5},
		{Path: "internal/adapters/github/client.go", Status: entity.FileStatusModified, Additions: 10},
	}, nil
}

func (s stubVCS) GetPullRequestCommits(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Commit, error) {
	return []entity.Commit{
		{SHA: "abc123", Message: "feat: add user entity", Author: entity.Author{Username: "dev"}},
	}, nil
}

func (s stubVCS) GetPullRequestReviews(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Review, error) {
	return []entity.Review{}, nil
}

func (s stubVCS) GetPullRequestComments(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Comment, error) {
	return []entity.Comment{}, nil
}

func (s stubVCS) GetFileContent(ctx context.Context, repo valueobject.RepositoryRef, path, ref string) (string, error) {
	return "package main\n", nil
}

func (s stubVCS) ListBranches(ctx context.Context, repo valueobject.RepositoryRef) ([]string, error) {
	return []string{"main", "develop"}, nil
}

func (s stubVCS) CompareRepositories(ctx context.Context, source, target valueobject.RepositoryRef) (*outbound.RepositoryComparison, error) {
	return &outbound.RepositoryComparison{
		SourceRepo: entity.Repository{Name: source.Name(), Owner: source.Owner(), Platform: entity.PlatformGitHub},
		TargetRepo: entity.Repository{Name: target.Name(), Owner: target.Owner(), Platform: entity.PlatformGitHub},
	}, nil
}

func newTestUseCase() *usecase.PRAnalyzerUseCase {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	tracer := noop.NewTracerProvider().Tracer("test")
	vcs := stubVCS{}
	return usecase.NewPRAnalyzerUseCase(
		vcs, vcs,
		analysis.NewAnalyzer(log),
		migration.NewPlanner(log),
		architecture.NewMapper(log),
		tracer,
		log,
	)
}

func testRepo(t *testing.T) valueobject.RepositoryRef {
	t.Helper()
	repo, err := valueobject.ParseRepositoryRef("myorg/myrepo")
	require.NoError(t, err)
	return repo
}

func testPRNumber(t *testing.T) valueobject.PRNumber {
	t.Helper()
	n, err := valueobject.NewPRNumber(42)
	require.NoError(t, err)
	return n
}

func TestPRAnalyzerUseCase_AnalyzePR(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	result, err := uc.AnalyzePR(ctx, inbound.AnalyzePRRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.ExecutiveSummary)
	assert.NotEmpty(t, result.BusinessPurpose)
	assert.NotEmpty(t, result.TechnicalPurpose)
	assert.NotNil(t, result.PullRequest)
	assert.NotEmpty(t, result.FilesChanged)
	assert.NotNil(t, result.MigrationPlan)
	assert.NotNil(t, result.ArchitectureImpact)
	assert.NotNil(t, result.DatabaseImpact)
	assert.NotNil(t, result.ConfigurationImpact)
	assert.NotNil(t, result.KubernetesImpact)
	assert.NotEmpty(t, result.ValidationSteps)
	assert.NotNil(t, result.RollbackStrategy)
	assert.NotEmpty(t, result.MigrationChecklist)
}

func TestPRAnalyzerUseCase_ExplainChange(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	explanation, err := uc.ExplainChange(ctx, inbound.ExplainChangeRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
		FilePath:   "internal/domain/entity/user.go",
	})

	require.NoError(t, err)
	assert.NotNil(t, explanation)
	assert.NotEmpty(t, explanation.Why)
	assert.NotEmpty(t, explanation.What)
	assert.NotEmpty(t, explanation.How)
}

func TestPRAnalyzerUseCase_GenerateMigrationPlan(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	target, err := valueobject.ParseRepositoryRef("myorg/target-repo")
	require.NoError(t, err)

	plan, err := uc.GenerateMigrationPlan(ctx, inbound.MigrationPlanRequest{
		SourcePlatform: entity.PlatformGitHub,
		SourceRepo:     testRepo(t),
		PRNumber:       testPRNumber(t),
		TargetRepo:     target,
	})

	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.NotEmpty(t, plan.Steps)
	assert.NotEmpty(t, plan.Title)
}

func TestPRAnalyzerUseCase_AnalyzeDependencies(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	deps, err := uc.AnalyzeDependencies(ctx, inbound.DependencyRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.NotNil(t, deps)
}

func TestPRAnalyzerUseCase_GenerateArchitectureMap(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	impact, err := uc.GenerateArchitectureMap(ctx, inbound.ArchitectureRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.NotNil(t, impact)
	assert.Contains(t, impact.LayersAffected, "domain")
}

func TestPRAnalyzerUseCase_FindRelatedFiles(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	files, err := uc.FindRelatedFiles(ctx, inbound.RelatedFilesRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.NotNil(t, files)
}

func TestPRAnalyzerUseCase_CompareRepositories(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	target, err := valueobject.ParseRepositoryRef("myorg/target-repo")
	require.NoError(t, err)

	result, err := uc.CompareRepositories(ctx, inbound.CompareReposRequest{
		SourcePlatform: entity.PlatformGitHub,
		SourceRepo:     testRepo(t),
		TargetPlatform: entity.PlatformGitHub,
		TargetRepo:     target,
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Suggestions)
}

func TestPRAnalyzerUseCase_GenerateMigrationChecklist(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	checklist, err := uc.GenerateMigrationChecklist(ctx, inbound.ChecklistRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.NotEmpty(t, checklist)
}

func TestPRAnalyzerUseCase_ExplainCodeFlow(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	flow, err := uc.ExplainCodeFlow(ctx, inbound.CodeFlowRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
		EntryPoint: "main()",
	})

	require.NoError(t, err)
	assert.NotNil(t, flow)
	assert.NotEmpty(t, flow.Description)
}

func TestPRAnalyzerUseCase_GenerateFeatureSummary(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	summary, err := uc.GenerateFeatureSummary(ctx, inbound.FeatureSummaryRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.NotNil(t, summary)
	assert.NotEmpty(t, summary.Title)
}

func TestPRAnalyzerUseCase_GenerateMigrationDocumentation(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	doc, err := uc.GenerateMigrationDocumentation(ctx, inbound.MigrationDocRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.NotEmpty(t, doc.Title)
	assert.NotEmpty(t, doc.ExecutiveSummary)
}

func TestPRAnalyzerUseCase_FindRequiredDependencies(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	deps, err := uc.FindRequiredDependencies(ctx, inbound.RequiredDepsRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	// Returns nil or empty slice when no Go import patches are present
	require.NoError(t, err)
	_ = deps
}

func TestPRAnalyzerUseCase_AnalyzePR_Bitbucket(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	result, err := uc.AnalyzePR(ctx, inbound.AnalyzePRRequest{
		Platform:   entity.PlatformBitbucket,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPRAnalyzerUseCase_AnalyzePR_BusinessPurpose_Fix(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	// stubVCS returns "Add new feature" which triggers "feature" keyword
	result, err := uc.AnalyzePR(ctx, inbound.AnalyzePRRequest{
		Platform:   entity.PlatformGitHub,
		Repository: testRepo(t),
		PRNumber:   testPRNumber(t),
	})

	require.NoError(t, err)
	assert.Contains(t, result.BusinessPurpose, "Feature")
}

func BenchmarkPRAnalyzerUseCase_AnalyzePR(b *testing.B) {
	uc := newTestUseCase()
	ctx := context.Background()
	repo, _ := valueobject.ParseRepositoryRef("myorg/myrepo")
	prNum, _ := valueobject.NewPRNumber(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uc.AnalyzePR(ctx, inbound.AnalyzePRRequest{
			Platform:   entity.PlatformGitHub,
			Repository: repo,
			PRNumber:   prNum,
		})
	}
}
