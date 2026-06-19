package migration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
)

func TestPlanner_GeneratePlan_WithDatabaseDep(t *testing.T) {
	p := newTestPlanner()
	ctx := context.Background()
	pr := makePR(25)

	deps := []entity.Dependency{
		{Name: "github.com/jackc/pgx/v5", Type: entity.DependencyTypeDatabase, Required: true, Description: "PostgreSQL driver"},
		{Name: "github.com/redis/go-redis/v9", Type: entity.DependencyTypePackage, Required: true},
	}

	plan, err := p.GeneratePlan(ctx, pr, deps)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, entity.EffortLevelLarge, plan.Effort)

	// Should have risk for large PR
	assert.NotEmpty(t, plan.Risks)
	hasHighRisk := false
	for _, r := range plan.Risks {
		if r.Level == entity.RiskLevelHigh {
			hasHighRisk = true
		}
	}
	assert.True(t, hasHighRisk, "large PR should trigger high risk")
}

func TestPlanner_GeneratePlan_ManyDeps(t *testing.T) {
	p := newTestPlanner()
	ctx := context.Background()
	pr := makePR(5)

	deps := make([]entity.Dependency, 15)
	for i := range deps {
		deps[i] = entity.Dependency{
			Name:     "github.com/some/pkg",
			Type:     entity.DependencyTypePackage,
			Required: true,
		}
	}

	plan, err := p.GeneratePlan(ctx, pr, deps)
	require.NoError(t, err)
	hasMediumRisk := false
	for _, r := range plan.Risks {
		if r.Level == entity.RiskLevelMedium {
			hasMediumRisk = true
		}
	}
	assert.True(t, hasMediumRisk, "many deps should trigger medium risk")
}

func TestPlanner_GenerateChecklist_NoAnalysisDeps(t *testing.T) {
	p := newTestPlanner()
	ctx := context.Background()
	pr := makePR(2)

	result := &entity.AnalysisResult{
		PullRequest:     pr,
		Dependencies:    []entity.Dependency{},
		ValidationSteps: []entity.ValidationStep{},
	}

	checklist, err := p.GenerateChecklist(ctx, pr, result)
	require.NoError(t, err)
	assert.NotEmpty(t, checklist)

	// Must still have required categories
	categories := make(map[string]bool)
	for _, item := range checklist {
		categories[item.Category] = true
	}
	assert.True(t, categories["preparation"])
	assert.True(t, categories["testing"])
	assert.True(t, categories["rollback"])
}
