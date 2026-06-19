package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
)

func TestDetectLayerFromPath_AllLayers(t *testing.T) {
	tests := []struct {
		path  string
		layer string
	}{
		{"internal/domain/entity/foo.go", "domain"},
		{"internal/application/usecase/foo.go", "application"},
		{"internal/adapters/github/client.go", "adapter"},
		{"internal/port/inbound/port.go", "port"},
		{"internal/service/analysis/analyzer.go", "service"},
		{"some/path/cmd/server/main.go", "presentation"},
		{"internal/repository/impl.go", "repository"},
		{"some/path/pkg/util.go", "shared"},
		{"configs/config.yaml", ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.layer, detectLayerFromPath(tc.path))
		})
	}
}

func TestInferFilePurpose_AllCases(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"user_test.go", "Test file"},
		{"api_handler.go", "HTTP or MCP request handler"},
		{"user_repository.go", "Data access layer"},
		{"create_usecase.go", "Business logic use case"},
		{"user_entity.go", "Domain entity"},
		{"app_config.go", "Configuration"},
		{"0001_migration.go", "Database migration"},
		{"service.go", "Go source file"},
		{"README.md", "Source file"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.expected, inferFilePurpose(tc.path))
		})
	}
}

func TestAssessFileImpact(t *testing.T) {
	uc := &PRAnalyzerUseCase{}

	tests := []struct {
		additions int
		deletions int
		contains  string
	}{
		{150, 100, "High impact"},
		{30, 25, "Medium impact"},
		{5, 3, "Low impact"},
	}

	for _, tc := range tests {
		f := &entity.ChangedFile{Additions: tc.additions, Deletions: tc.deletions}
		result := uc.assessFileImpact(f)
		assert.Contains(t, result, tc.contains)
	}
}

func TestDescribeHowToMigrate(t *testing.T) {
	uc := &PRAnalyzerUseCase{}

	tests := []struct {
		file     *entity.ChangedFile
		contains string
	}{
		{&entity.ChangedFile{Path: "new.go", Status: entity.FileStatusAdded}, "Copy"},
		{&entity.ChangedFile{Path: "old.go", Status: entity.FileStatusDeleted}, "Remove"},
		{&entity.ChangedFile{Path: "new.go", OldPath: "old.go", Status: entity.FileStatusRenamed}, "Rename"},
		{&entity.ChangedFile{Path: "existing.go", Status: entity.FileStatusModified}, "Apply"},
	}

	for _, tc := range tests {
		result := uc.describeHowToMigrate(tc.file)
		assert.Contains(t, result, tc.contains)
	}
}

func TestDetectEntryFunction(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"cmd/server/main.go", "main()"},
		{"api/handler.go", "Handle()"},
		{"create_usecase.go", "Execute()"},
		{"analysis_service.go", "Run()"},
		{"other.go", "New()"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.expected, detectEntryFunction(tc.path))
		})
	}
}

func TestAppendUnique(t *testing.T) {
	s := []string{"a", "b"}
	s = appendUnique(s, "c")
	assert.Equal(t, []string{"a", "b", "c"}, s)

	s = appendUnique(s, "a")
	assert.Equal(t, []string{"a", "b", "c"}, s)
}

func TestContainsAny(t *testing.T) {
	assert.True(t, containsAny("fix: bug fix", "fix", "feat"))
	assert.True(t, containsAny("feat: new feature", "fix", "feat"))
	assert.False(t, containsAny("chore: cleanup", "fix", "feat"))
}
