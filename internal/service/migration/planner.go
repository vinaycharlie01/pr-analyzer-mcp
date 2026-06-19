package migration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
)

// Planner generates migration plans from PR analysis results.
type Planner struct {
	logger *slog.Logger
}

func NewPlanner(logger *slog.Logger) *Planner {
	return &Planner{logger: logger}
}

func (p *Planner) GeneratePlan(ctx context.Context, pr *entity.PullRequest, deps []entity.Dependency) (*entity.MigrationPlan, error) {
	steps := p.buildMigrationSteps(pr, deps)
	risks := p.assessRisks(pr, deps)
	effort := p.estimateEffort(pr)

	plan := &entity.MigrationPlan{
		Title:       fmt.Sprintf("Migration Plan: %s", pr.Title),
		Description: p.buildDescription(pr),
		Steps:       steps,
		Risks:       risks,
		Effort:      effort,
		Timeline:    p.estimateTimeline(effort, len(steps)),
	}

	return plan, nil
}

func (p *Planner) GenerateChecklist(ctx context.Context, pr *entity.PullRequest, analysis *entity.AnalysisResult) ([]entity.ChecklistItem, error) {
	var items []entity.ChecklistItem
	id := 1

	items = append(items, entity.ChecklistItem{
		ID:          fmt.Sprintf("CHK-%03d", id),
		Title:       "Review PR description and context",
		Description: "Understand the business and technical purpose of this change",
		Required:    true,
		Category:    "preparation",
	})
	id++

	for _, dep := range analysis.Dependencies {
		items = append(items, entity.ChecklistItem{
			ID:          fmt.Sprintf("CHK-%03d", id),
			Title:       fmt.Sprintf("Verify dependency: %s", dep.Name),
			Description: fmt.Sprintf("Ensure %s is available in the target repository", dep.Name),
			Required:    dep.Required,
			Category:    "dependencies",
		})
		id++
	}

	for _, step := range analysis.ValidationSteps {
		items = append(items, entity.ChecklistItem{
			ID:          fmt.Sprintf("CHK-%03d", id),
			Title:       step.Title,
			Description: step.Description,
			Required:    true,
			Category:    "validation",
		})
		id++
	}

	items = append(items, entity.ChecklistItem{
		ID:          fmt.Sprintf("CHK-%03d", id),
		Title:       "Run full test suite",
		Description: "Execute all unit and integration tests after migration",
		Required:    true,
		Category:    "testing",
	})
	id++

	items = append(items, entity.ChecklistItem{
		ID:          fmt.Sprintf("CHK-%03d", id),
		Title:       "Verify rollback procedure",
		Description: "Test that rollback steps work correctly",
		Required:    true,
		Category:    "rollback",
	})

	return items, nil
}

func (p *Planner) GenerateRollbackStrategy(ctx context.Context, pr *entity.PullRequest) (*entity.RollbackStrategy, error) {
	return &entity.RollbackStrategy{
		Description: fmt.Sprintf("Rollback strategy for: %s", pr.Title),
		Steps: []string{
			"Revert the feature branch merge",
			"Restore previous configuration",
			"Rollback database migrations if applicable",
			"Restart affected services",
			"Verify system health after rollback",
		},
		Commands: []string{
			"git revert HEAD~1",
			"kubectl rollout undo deployment/app",
		},
	}, nil
}

func (p *Planner) buildMigrationSteps(pr *entity.PullRequest, deps []entity.Dependency) []entity.MigrationStep {
	steps := []entity.MigrationStep{
		{
			Order:       1,
			Title:       "Checkout target repository",
			Description: "Clone or update the target repository where features will be migrated",
			Commands:    []string{"git clone <target-repo-url>", "git checkout -b feature/migration"},
			Validation:  "Verify repository is clean: git status",
			Rollback:    "git checkout main && git branch -D feature/migration",
		},
		{
			Order:       2,
			Title:       "Install required dependencies",
			Description: fmt.Sprintf("Add %d dependencies identified in the source PR", len(deps)),
			Commands:    p.buildDependencyCommands(deps),
			Validation:  "Run: go mod tidy && go build ./...",
			Rollback:    "Restore go.mod and go.sum from backup",
		},
		{
			Order:       3,
			Title:       fmt.Sprintf("Migrate changed files (%d files)", len(pr.Files)),
			Description: "Copy and adapt the changed files to the target repository structure",
			Commands:    p.buildFileMigrationCommands(pr.Files),
			Validation:  "go vet ./...",
			Rollback:    "git checkout -- .",
		},
		{
			Order:       4,
			Title:       "Adapt imports and package names",
			Description: "Update import paths to match the target repository module name",
			Commands:    []string{"sed -i 's|<source-module>|<target-module>|g' **/*.go"},
			Validation:  "go build ./...",
			Rollback:    "git checkout -- .",
		},
		{
			Order:       5,
			Title:       "Run tests",
			Description: "Execute the test suite to verify migration correctness",
			Commands:    []string{"go test ./...", "go test -race ./..."},
			Validation:  "All tests pass with 0 failures",
			Rollback:    "Investigate and fix test failures before proceeding",
		},
		{
			Order:       6,
			Title:       "Create pull request in target repository",
			Description: "Open a PR with the migrated changes for review",
			Commands:    []string{"git add -A", "git commit -m \"feat: migrate <feature> from <source>\"", "git push origin feature/migration"},
			Validation:  "PR created and CI pipeline passes",
			Rollback:    "Close the PR and revert changes",
		},
	}
	return steps
}

func (p *Planner) buildDependencyCommands(deps []entity.Dependency) []string {
	cmds := []string{"cp go.mod go.mod.backup", "cp go.sum go.sum.backup"}
	for _, dep := range deps {
		if dep.Type == entity.DependencyTypePackage {
			cmds = append(cmds, fmt.Sprintf("go get %s", dep.Name))
		}
	}
	cmds = append(cmds, "go mod tidy")
	return cmds
}

func (p *Planner) buildFileMigrationCommands(files []entity.ChangedFile) []string {
	var cmds []string
	for _, f := range files {
		if f.Status == entity.FileStatusAdded {
			dir := directoryOf(f.Path)
			if dir != "" {
				cmds = append(cmds, fmt.Sprintf("mkdir -p %s", dir))
			}
			cmds = append(cmds, fmt.Sprintf("# Copy: %s", f.Path))
		}
	}
	return cmds
}

func (p *Planner) buildDescription(pr *entity.PullRequest) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("This migration plan covers the changes introduced in PR #%d: %s.\n\n", pr.Number, pr.Title))
	sb.WriteString(fmt.Sprintf("Source: %s/%s\n", pr.Repository.Owner, pr.Repository.Name))
	sb.WriteString(fmt.Sprintf("Branch: %s -> %s\n", pr.SourceBranch, pr.TargetBranch))
	sb.WriteString(fmt.Sprintf("Files changed: %d\n", len(pr.Files)))
	return sb.String()
}

func (p *Planner) assessRisks(pr *entity.PullRequest, deps []entity.Dependency) []entity.Risk {
	var risks []entity.Risk

	if len(pr.Files) > 20 {
		risks = append(risks, entity.Risk{
			Level:       entity.RiskLevelHigh,
			Description: fmt.Sprintf("Large changeset: %d files changed", len(pr.Files)),
			Mitigation:  "Break migration into smaller, incremental steps",
		})
	}

	for _, dep := range deps {
		if dep.Type == entity.DependencyTypeDatabase {
			risks = append(risks, entity.Risk{
				Level:       entity.RiskLevelHigh,
				Description: "Database dependency detected: migration may require schema changes",
				Mitigation:  "Review database migrations carefully; test on staging first",
			})
			break
		}
	}

	if len(deps) > 10 {
		risks = append(risks, entity.Risk{
			Level:       entity.RiskLevelMedium,
			Description: fmt.Sprintf("High dependency count: %d dependencies", len(deps)),
			Mitigation:  "Verify all dependencies are compatible with target repository",
		})
	}

	return risks
}

func (p *Planner) estimateEffort(pr *entity.PullRequest) entity.EffortLevel {
	fileCount := len(pr.Files)
	switch {
	case fileCount <= 5:
		return entity.EffortLevelSmall
	case fileCount <= 20:
		return entity.EffortLevelMedium
	default:
		return entity.EffortLevelLarge
	}
}

func (p *Planner) estimateTimeline(effort entity.EffortLevel, stepCount int) string {
	switch effort {
	case entity.EffortLevelSmall:
		return fmt.Sprintf("Estimated %d-4 hours for %d steps", stepCount/2+1, stepCount)
	case entity.EffortLevelMedium:
		return fmt.Sprintf("Estimated 1-3 days for %d steps", stepCount)
	default:
		return fmt.Sprintf("Estimated 3-7 days for %d steps", stepCount)
	}
}

func directoryOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}
