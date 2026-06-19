package migration_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/migration"
)

func newTestPlanner() *migration.Planner {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return migration.NewPlanner(log)
}

func makePR(fileCount int) *entity.PullRequest {
	files := make([]entity.ChangedFile, fileCount)
	for i := range files {
		files[i] = entity.ChangedFile{
			Path:      "internal/some/file.go",
			Status:    entity.FileStatusModified,
			Additions: 10,
			Deletions: 5,
		}
	}
	return &entity.PullRequest{
		ID:           "1",
		Number:       42,
		Title:        "Add new feature",
		Description:  "This PR adds an important feature",
		SourceBranch: "feature/new-thing",
		TargetBranch: "main",
		Platform:     entity.PlatformGitHub,
		Files:        files,
		Author:       entity.Author{Username: "dev"},
		CreatedAt:    time.Now(),
		Repository:   entity.Repository{Name: "myrepo", Owner: "myorg"},
	}
}

func TestPlanner_GeneratePlan(t *testing.T) {
	p := newTestPlanner()
	ctx := context.Background()

	tests := []struct {
		name       string
		fileCount  int
		wantEffort entity.EffortLevel
	}{
		{name: "small PR", fileCount: 3, wantEffort: entity.EffortLevelSmall},
		{name: "medium PR", fileCount: 10, wantEffort: entity.EffortLevelMedium},
		{name: "large PR", fileCount: 30, wantEffort: entity.EffortLevelLarge},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pr := makePR(tc.fileCount)
			plan, err := p.GeneratePlan(ctx, pr, nil)
			require.NoError(t, err)
			assert.NotNil(t, plan)
			assert.Equal(t, tc.wantEffort, plan.Effort)
			assert.NotEmpty(t, plan.Steps)
			assert.NotEmpty(t, plan.Title)
			assert.NotEmpty(t, plan.Timeline)
		})
	}
}

func TestPlanner_GenerateChecklist(t *testing.T) {
	p := newTestPlanner()
	ctx := context.Background()
	pr := makePR(5)

	analysis := &entity.AnalysisResult{
		PullRequest: pr,
		Dependencies: []entity.Dependency{
			{Name: "github.com/some/package", Type: entity.DependencyTypePackage, Required: true},
		},
		ValidationSteps: []entity.ValidationStep{
			{Order: 1, Title: "Build", Command: "go build ./..."},
		},
	}

	checklist, err := p.GenerateChecklist(ctx, pr, analysis)
	require.NoError(t, err)
	assert.NotEmpty(t, checklist)

	// Must have preparation, dependency, validation, testing, rollback items
	categories := make(map[string]bool)
	for _, item := range checklist {
		categories[item.Category] = true
		assert.NotEmpty(t, item.ID)
		assert.NotEmpty(t, item.Title)
	}
	assert.True(t, categories["preparation"])
	assert.True(t, categories["dependencies"])
	assert.True(t, categories["testing"])
}

func TestPlanner_GenerateRollbackStrategy(t *testing.T) {
	p := newTestPlanner()
	ctx := context.Background()
	pr := makePR(2)

	strategy, err := p.GenerateRollbackStrategy(ctx, pr)
	require.NoError(t, err)
	assert.NotNil(t, strategy)
	assert.NotEmpty(t, strategy.Description)
	assert.NotEmpty(t, strategy.Steps)
	assert.NotEmpty(t, strategy.Commands)
}

func BenchmarkPlanner_GeneratePlan(b *testing.B) {
	p := newTestPlanner()
	ctx := context.Background()
	pr := makePR(15)
	deps := []entity.Dependency{
		{Name: "github.com/some/pkg", Type: entity.DependencyTypePackage, Required: true},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.GeneratePlan(ctx, pr, deps)
	}
}
