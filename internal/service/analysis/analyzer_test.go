package analysis_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/analysis"
)

func newTestAnalyzer() *analysis.Analyzer {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return analysis.NewAnalyzer(log)
}

func TestAnalyzer_AnalyzeDependencies(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	tests := []struct {
		name     string
		files    []entity.ChangedFile
		wantDeps int
	}{
		{
			name: "no go files",
			files: []entity.ChangedFile{
				{Path: "README.md", Status: entity.FileStatusModified, Patch: "some content"},
			},
			wantDeps: 0,
		},
		{
			name: "go file with imports",
			files: []entity.ChangedFile{
				{
					Path:   "main.go",
					Status: entity.FileStatusModified,
					Patch: `+import (
+	"github.com/some/package"
+	"golang.org/x/tools/go/packages"
+)`,
				},
			},
			wantDeps: 2,
		},
		{
			name:     "empty files",
			files:    []entity.ChangedFile{},
			wantDeps: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps, err := a.AnalyzeDependencies(ctx, tc.files)
			require.NoError(t, err)
			assert.Len(t, deps, tc.wantDeps)
		})
	}
}

func TestAnalyzer_AnalyzeArchitecture(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	files := []entity.ChangedFile{
		{Path: "internal/domain/entity/user.go", Status: entity.FileStatusAdded},
		{Path: "internal/adapters/github/client.go", Status: entity.FileStatusModified},
		{Path: "internal/service/analysis/analyzer.go", Status: entity.FileStatusAdded},
	}

	impact, err := a.AnalyzeArchitecture(ctx, files)
	require.NoError(t, err)
	assert.NotNil(t, impact)
	assert.Contains(t, impact.LayersAffected, "domain")
	assert.Contains(t, impact.LayersAffected, "adapter")
	assert.Contains(t, impact.LayersAffected, "service")
	assert.Len(t, impact.NewComponents, 2)
	assert.Len(t, impact.ModifiedComponents, 1)
}

func TestAnalyzer_FindRelatedFiles(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	files := []entity.ChangedFile{
		{Path: "internal/service/analysis/analyzer.go", Status: entity.FileStatusModified},
	}

	related, err := a.FindRelatedFiles(ctx, files, "")
	require.NoError(t, err)
	assert.NotEmpty(t, related)
	// Should suggest test file
	found := false
	for _, r := range related {
		if r == "internal/service/analysis/analyzer_test.go" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected test file in related files")
}

func BenchmarkAnalyzer_AnalyzeDependencies(b *testing.B) {
	a := newTestAnalyzer()
	ctx := context.Background()
	files := make([]entity.ChangedFile, 50)
	for i := range files {
		files[i] = entity.ChangedFile{
			Path:   "file.go",
			Status: entity.FileStatusModified,
			Patch: `+import (
+	"github.com/some/package"
+	"golang.org/x/tools/go/packages"
+)`,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.AnalyzeDependencies(ctx, files)
	}
}

func BenchmarkAnalyzer_AnalyzeArchitecture(b *testing.B) {
	a := newTestAnalyzer()
	ctx := context.Background()
	files := make([]entity.ChangedFile, 20)
	for i := range files {
		files[i] = entity.ChangedFile{
			Path:   "internal/service/some/file.go",
			Status: entity.FileStatusModified,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.AnalyzeArchitecture(ctx, files)
	}
}
