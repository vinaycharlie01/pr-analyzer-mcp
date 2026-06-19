package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/inbound"
)

// stubAnalyzer implements inbound.PRAnalyzerPort for testing.
type stubAnalyzer struct{}

func (s *stubAnalyzer) AnalyzePR(_ context.Context, _ inbound.AnalyzePRRequest) (*entity.AnalysisResult, error) {
	pr := &entity.PullRequest{Title: "Test PR", Description: "desc"}
	return &entity.AnalysisResult{
		PullRequest:      pr,
		ExecutiveSummary: "Test summary",
		BusinessPurpose:  "Business purpose",
		TechnicalPurpose: "Technical purpose",
		FilesChanged: []entity.FileAnalysis{
			{Path: "main.go", ChangeType: "modified", Impact: "low"},
		},
		Dependencies: []entity.Dependency{
			{Name: "github.com/foo/bar", Type: entity.DependencyTypePackage},
		},
		ArchitectureImpact: &entity.ArchitectureImpact{
			Description:    "Arch desc",
			LayersAffected: []string{"service"},
		},
		DatabaseImpact: &entity.DatabaseImpact{
			Description: "DB desc",
		},
		ConfigurationImpact: &entity.ConfigurationImpact{
			Description: "Config desc",
		},
		KubernetesImpact: &entity.KubernetesImpact{
			Description: "K8s desc",
		},
		ValidationSteps: []entity.ValidationStep{
			{Order: 1, Title: "Build", Command: "go build ./..."},
		},
		RollbackStrategy: &entity.RollbackStrategy{
			Description: "Rollback desc",
			Steps:       []string{"git revert HEAD"},
		},
	}, nil
}

func (s *stubAnalyzer) ExplainChange(_ context.Context, req inbound.ExplainChangeRequest) (*inbound.ChangeExplanation, error) {
	return &inbound.ChangeExplanation{
		FilePath: req.FilePath,
		Why:      "Because of a bug fix",
		What:     "Fixed null pointer",
		How:      "Added nil check",
		Impact:   "Low",
		Risks: []entity.Risk{
			{Level: entity.RiskLevelMedium, Description: "May break existing tests", Mitigation: "Add regression tests"},
		},
	}, nil
}

func (s *stubAnalyzer) GenerateMigrationPlan(_ context.Context, _ inbound.MigrationPlanRequest) (*entity.MigrationPlan, error) {
	return &entity.MigrationPlan{
		Title:       "Migration Plan",
		Description: "Step by step plan",
		Effort:      entity.EffortLevelSmall,
		Timeline:    "1 day",
		Risks: []entity.Risk{
			{Level: entity.RiskLevelHigh, Description: "Large changeset", Mitigation: "Break into smaller steps"},
		},
		Steps: []entity.MigrationStep{
			{Order: 1, Title: "Copy files", Description: "Copy all changed files", Commands: []string{"cp -r src dst"}, Validation: "go build ./...", Rollback: "rm -rf dst"},
		},
	}, nil
}

func (s *stubAnalyzer) AnalyzeDependencies(_ context.Context, _ inbound.DependencyRequest) ([]entity.Dependency, error) {
	return []entity.Dependency{
		{Name: "github.com/some/pkg", Type: entity.DependencyTypePackage, Required: true},
	}, nil
}

func (s *stubAnalyzer) GenerateArchitectureMap(_ context.Context, _ inbound.ArchitectureRequest) (*entity.ArchitectureImpact, error) {
	return &entity.ArchitectureImpact{
		Description:    "Architecture impact description",
		LayersAffected: []string{"service", "domain"},
	}, nil
}

func (s *stubAnalyzer) FindRelatedFiles(_ context.Context, _ inbound.RelatedFilesRequest) ([]string, error) {
	return []string{"main_test.go", "internal/service/foo_test.go"}, nil
}

func (s *stubAnalyzer) FindRequiredDependencies(_ context.Context, _ inbound.RequiredDepsRequest) ([]entity.Dependency, error) {
	return []entity.Dependency{
		{Name: "github.com/required/dep", Type: entity.DependencyTypePackage, Required: true, Description: "Required dep"},
	}, nil
}

func (s *stubAnalyzer) CompareRepositories(_ context.Context, req inbound.CompareReposRequest) (*inbound.ComparisonResult, error) {
	return &inbound.ComparisonResult{
		Source:      entity.Repository{Name: req.SourceRepo.Name(), FullName: req.SourceRepo.String(), Platform: req.SourcePlatform},
		Target:      entity.Repository{Name: req.TargetRepo.Name(), FullName: req.TargetRepo.String(), Platform: req.TargetPlatform},
		CommonFiles: []string{"main.go", "go.mod"},
		Differences: []inbound.FileDifference{{Path: "README.md", Description: "different content"}},
		Suggestions: []string{"Review all imports", "Update module paths"},
	}, nil
}

func (s *stubAnalyzer) GenerateMigrationChecklist(_ context.Context, _ inbound.ChecklistRequest) ([]entity.ChecklistItem, error) {
	return []entity.ChecklistItem{
		{ID: "prep-1", Title: "Create branch", Category: "preparation", Required: true, Description: "Create a new branch"},
		{ID: "test-1", Title: "Run tests", Category: "testing", Required: true, Description: "Run all tests"},
	}, nil
}

func (s *stubAnalyzer) ExplainCodeFlow(_ context.Context, req inbound.CodeFlowRequest) (*inbound.CodeFlowExplanation, error) {
	return &inbound.CodeFlowExplanation{
		EntryPoint:  req.EntryPoint,
		Description: "Flow description",
		Flow: []inbound.FlowStep{
			{Order: 1, Function: "main", File: "main.go", Description: "Entry point"},
		},
	}, nil
}

func (s *stubAnalyzer) GenerateFeatureSummary(_ context.Context, _ inbound.FeatureSummaryRequest) (*inbound.FeatureSummary, error) {
	return &inbound.FeatureSummary{
		Title:           "New Feature",
		BusinessValue:   "Improves performance",
		TechnicalDetail: "Uses caching",
		AffectedAreas:   []string{"API", "Database"},
	}, nil
}

func (s *stubAnalyzer) GenerateMigrationDocumentation(_ context.Context, _ inbound.MigrationDocRequest) (*inbound.MigrationDocumentation, error) {
	return &inbound.MigrationDocumentation{
		Title:             "Migration Doc",
		ExecutiveSummary:  "Executive summary",
		TechnicalOverview: "Technical overview",
		MigrationSteps: []entity.MigrationStep{
			{Order: 1, Title: "Step 1", Description: "First step", Commands: []string{"go build"}},
		},
		Checklist: []entity.ChecklistItem{
			{ID: "c-1", Title: "Check tests", Category: "testing"},
		},
		ValidationSteps: []entity.ValidationStep{
			{Order: 1, Title: "Build", Command: "go build ./...", Expected: "no errors"},
		},
	}, nil
}

func newTestMCPServer(t *testing.T) *Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewServer(&stubAnalyzer{}, logger)
}

// callTool sends a tools/call JSON-RPC request and returns the decoded result content.
func callTool(t *testing.T, s *Server, toolName string, args map[string]any) map[string]any {
	t.Helper()
	argsBytes, err := json.Marshal(args)
	require.NoError(t, err)

	msg := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`,
		toolName, argsBytes)

	resp := s.mcpServer.HandleMessage(context.Background(), []byte(msg))
	require.NotNil(t, resp)

	respBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(respBytes, &decoded))
	return decoded
}

// getTextContent extracts the first text content value from a tools/call response.
func getTextContent(resp map[string]any) string {
	result, ok := resp["result"].(map[string]any)
	if !ok {
		return ""
	}
	contents, ok := result["content"].([]any)
	if !ok || len(contents) == 0 {
		return ""
	}
	first, ok := contents[0].(map[string]any)
	if !ok {
		return ""
	}
	text, _ := first["text"].(string)
	return text
}

func prArgs() map[string]any {
	return map[string]any{
		"platform":   "github",
		"repository": "myorg/myrepo",
		"pr_number":  float64(1),
	}
}

func TestMCPServer_NewServer(t *testing.T) {
	s := newTestMCPServer(t)
	assert.NotNil(t, s)
	assert.NotNil(t, s.mcpServer)
}

func TestMCPServer_AnalyzePR(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "analyze_pr", prArgs())
	text := getTextContent(resp)
	assert.Contains(t, text, "PR Analysis")
	assert.Contains(t, text, "Test PR")
	assert.Contains(t, text, "Business Purpose")
}

func TestMCPServer_AnalyzePR_InvalidRepo(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "analyze_pr", map[string]any{
		"platform":   "github",
		"repository": "invalid-no-slash",
		"pr_number":  float64(1),
	})
	text := getTextContent(resp)
	// Should return an error result
	assert.NotEmpty(t, text)
}

func TestMCPServer_AnalyzePR_InvalidPRNumber(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "analyze_pr", map[string]any{
		"platform":   "github",
		"repository": "myorg/myrepo",
		"pr_number":  float64(0),
	})
	text := getTextContent(resp)
	assert.NotEmpty(t, text)
}

func TestMCPServer_ExplainChange(t *testing.T) {
	s := newTestMCPServer(t)
	args := prArgs()
	args["file_path"] = "main.go"
	resp := callTool(t, s, "explain_change", args)
	text := getTextContent(resp)
	assert.Contains(t, text, "Change Explanation")
	assert.Contains(t, text, "main.go")
}

func TestMCPServer_GenerateMigrationPlan(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "generate_migration_plan", map[string]any{
		"platform":          "github",
		"source_repository": "myorg/source",
		"pr_number":         float64(1),
		"target_repository": "myorg/target",
	})
	text := getTextContent(resp)
	assert.Contains(t, text, "Migration Plan")
}

func TestMCPServer_DependencyAnalysis(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "dependency_analysis", prArgs())
	text := getTextContent(resp)
	assert.Contains(t, text, "github.com/some/pkg")
}

func TestMCPServer_ArchitectureMap(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "architecture_map", prArgs())
	text := getTextContent(resp)
	assert.Contains(t, text, "Architecture")
}

func TestMCPServer_FindRelatedFiles(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "find_related_files", prArgs())
	text := getTextContent(resp)
	assert.Contains(t, text, "Related Files")
	assert.Contains(t, text, "main_test.go")
}

func TestMCPServer_FindRequiredDependencies(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "find_required_dependencies", prArgs())
	text := getTextContent(resp)
	assert.Contains(t, text, "Required Dependencies")
}

func TestMCPServer_CompareRepositories(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "compare_repositories", map[string]any{
		"source_platform":   "github",
		"source_repository": "myorg/source",
		"target_platform":   "github",
		"target_repository": "myorg/target",
	})
	text := getTextContent(resp)
	assert.Contains(t, text, "Repository Comparison")
}

func TestMCPServer_GenerateMigrationChecklist(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "generate_migration_checklist", prArgs())
	text := getTextContent(resp)
	assert.Contains(t, text, "Migration Checklist")
	assert.Contains(t, text, "preparation")
}

func TestMCPServer_ExplainCodeFlow(t *testing.T) {
	s := newTestMCPServer(t)
	args := prArgs()
	args["entry_point"] = "main"
	resp := callTool(t, s, "explain_code_flow", args)
	text := getTextContent(resp)
	assert.Contains(t, text, "Code Flow")
}

func TestMCPServer_GenerateFeatureSummary(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "generate_feature_summary", prArgs())
	text := getTextContent(resp)
	assert.Contains(t, text, "Feature Summary")
	assert.Contains(t, text, "New Feature")
}

func TestMCPServer_GenerateMigrationDocumentation(t *testing.T) {
	s := newTestMCPServer(t)
	args := prArgs()
	args["target_repository"] = "myorg/target"
	resp := callTool(t, s, "generate_migration_documentation", args)
	text := getTextContent(resp)
	assert.Contains(t, text, "Migration Doc")
	assert.Contains(t, text, "Executive Summary")
}

func TestMCPServer_GenerateMigrationDocumentation_NoTarget(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "generate_migration_documentation", prArgs())
	text := getTextContent(resp)
	assert.Contains(t, text, "Migration Doc")
}

func TestMCPServer_GenerateMigrationPlan_InvalidSource(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "generate_migration_plan", map[string]any{
		"platform":          "github",
		"source_repository": "bad",
		"pr_number":         float64(1),
		"target_repository": "myorg/target",
	})
	text := getTextContent(resp)
	assert.NotEmpty(t, text)
}

func TestMCPServer_CompareRepositories_InvalidSource(t *testing.T) {
	s := newTestMCPServer(t)
	resp := callTool(t, s, "compare_repositories", map[string]any{
		"source_platform":   "github",
		"source_repository": "bad",
		"target_platform":   "github",
		"target_repository": "myorg/target",
	})
	text := getTextContent(resp)
	assert.NotEmpty(t, text)
}

func TestFormatDependencies_Empty(t *testing.T) {
	out := formatDependencies(nil)
	assert.Contains(t, out, "No dependencies")
}

func TestFormatDependencies_WithDeps(t *testing.T) {
	deps := []entity.Dependency{
		{Name: "github.com/foo/bar", Type: entity.DependencyTypePackage, Required: true, Description: "Foo package"},
	}
	out := formatDependencies(deps)
	assert.Contains(t, out, "github.com/foo/bar")
	assert.Contains(t, out, "*(required)*")
}

func TestFormatArchitectureImpact(t *testing.T) {
	impact := &entity.ArchitectureImpact{
		Description:        "Impact desc",
		LayersAffected:     []string{"service", "domain"},
		PatternsUsed:       []string{"repository"},
		NewComponents:      []string{"NewHandler"},
		ModifiedComponents: []string{"OldService"},
	}
	out := formatArchitectureImpact(impact)
	assert.Contains(t, out, "service")
	assert.Contains(t, out, "repository")
	assert.Contains(t, out, "NewHandler")
	assert.Contains(t, out, "OldService")
}

func TestJoinStrings(t *testing.T) {
	assert.Equal(t, "a, b, c", joinStrings([]string{"a", "b", "c"}))
	assert.Equal(t, "a", joinStrings([]string{"a"}))
	assert.Equal(t, "", joinStrings(nil))
}
