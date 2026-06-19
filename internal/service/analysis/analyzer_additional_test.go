package analysis_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
)

func TestAnalyzer_AnalyzeDependencies_GoImports(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	files := []entity.ChangedFile{
		{
			Path:   "service.go",
			Status: entity.FileStatusAdded,
			Patch: `+import "github.com/single/package"
`,
		},
	}

	deps, err := a.AnalyzeDependencies(ctx, files)
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "github.com/single/package", deps[0].Name)
}

func TestAnalyzer_AnalyzeDependencies_Deduplication(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	files := []entity.ChangedFile{
		{
			Path:   "a.go",
			Status: entity.FileStatusAdded,
			Patch: `+import (
+	"github.com/shared/package"
+)`,
		},
		{
			Path:   "b.go",
			Status: entity.FileStatusModified,
			Patch: `+import (
+	"github.com/shared/package"
+)`,
		},
	}

	deps, err := a.AnalyzeDependencies(ctx, files)
	require.NoError(t, err)
	assert.Len(t, deps, 1, "duplicate imports should be deduplicated")
}

func TestAnalyzer_AnalyzeArchitecture_NoFiles(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	impact, err := a.AnalyzeArchitecture(ctx, []entity.ChangedFile{})
	require.NoError(t, err)
	assert.NotNil(t, impact)
	assert.Empty(t, impact.LayersAffected)
	assert.Empty(t, impact.NewComponents)
}

func TestAnalyzer_AnalyzeArchitecture_DeletedFiles(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	files := []entity.ChangedFile{
		{Path: "internal/service/old.go", Status: entity.FileStatusDeleted},
	}

	impact, err := a.AnalyzeArchitecture(ctx, files)
	require.NoError(t, err)
	assert.Contains(t, impact.LayersAffected, "service")
	assert.Empty(t, impact.NewComponents)
	assert.Empty(t, impact.ModifiedComponents)
}

func TestAnalyzer_BuildDependencyGraph(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	packages := []string{
		"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity",
		"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/analysis",
	}

	graph, err := a.BuildDependencyGraph(ctx, packages)
	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Nodes, 2)
}

func TestAnalyzer_FindRelatedFiles_NonGoFiles(t *testing.T) {
	a := newTestAnalyzer()
	ctx := context.Background()

	files := []entity.ChangedFile{
		{Path: "configs/config.yaml", Status: entity.FileStatusModified},
		{Path: "README.md", Status: entity.FileStatusModified},
	}

	related, err := a.FindRelatedFiles(ctx, files, "")
	require.NoError(t, err)
	// Non-Go files don't generate related suggestions
	assert.NotNil(t, related)
}
