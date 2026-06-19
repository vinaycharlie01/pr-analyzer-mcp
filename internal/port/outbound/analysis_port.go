package outbound

import (
	"context"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
)

// AnalysisPort defines the outbound port for code analysis operations.
type AnalysisPort interface {
	AnalyzeDependencies(ctx context.Context, files []entity.ChangedFile) ([]entity.Dependency, error)
	AnalyzeArchitecture(ctx context.Context, files []entity.ChangedFile) (*entity.ArchitectureImpact, error)
	FindRelatedFiles(ctx context.Context, files []entity.ChangedFile, repoPath string) ([]string, error)
	BuildDependencyGraph(ctx context.Context, packages []string) (*DependencyGraph, error)
}

type DependencyGraph struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

type GraphNode struct {
	ID    string
	Label string
	Type  string
}

type GraphEdge struct {
	Source string
	Target string
	Label  string
}
