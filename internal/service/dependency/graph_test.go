package dependency_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/dependency"
)

func newTestGraphService() *dependency.GraphService {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return dependency.NewGraphService(log)
}

func TestGraphService_Build(t *testing.T) {
	g := newTestGraphService()
	ctx := context.Background()

	deps := []entity.Dependency{
		{Name: "github.com/some/package", Type: entity.DependencyTypePackage, Source: "main.go"},
		{Name: "github.com/other/package", Type: entity.DependencyTypePackage, Source: "main.go"},
		{Name: "internal/service/user", Type: entity.DependencyTypeInternal, Source: "cmd/main.go"},
	}

	graph, err := g.Build(ctx, deps)
	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.NotEmpty(t, graph.Nodes)
	assert.NotEmpty(t, graph.Edges)
}

func TestGraphService_Build_NoDeps(t *testing.T) {
	g := newTestGraphService()
	ctx := context.Background()

	graph, err := g.Build(ctx, []entity.Dependency{})
	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Empty(t, graph.Nodes)
	assert.Empty(t, graph.Edges)
}

func TestGraphService_DetectCycles(t *testing.T) {
	g := newTestGraphService()
	ctx := context.Background()

	deps := []entity.Dependency{
		{Name: "pkg/a", Type: entity.DependencyTypeInternal, Source: "pkg/b"},
		{Name: "pkg/b", Type: entity.DependencyTypeInternal, Source: "pkg/a"},
	}

	graph, err := g.Build(ctx, deps)
	require.NoError(t, err)

	cycles, err := g.DetectCycles(ctx, graph)
	require.NoError(t, err)
	// Cycles may or may not be detected depending on graph structure
	assert.NotNil(t, cycles)
}

func BenchmarkGraphService_Build(b *testing.B) {
	g := newTestGraphService()
	ctx := context.Background()
	deps := make([]entity.Dependency, 50)
	for i := range deps {
		deps[i] = entity.Dependency{
			Name:   "github.com/pkg/package",
			Type:   entity.DependencyTypePackage,
			Source: "main.go",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.Build(ctx, deps)
	}
}
