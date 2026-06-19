package dependency

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dominikbraun/graph"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
)

// GraphService builds and analyzes dependency graphs.
type GraphService struct {
	logger *slog.Logger
}

func NewGraphService(logger *slog.Logger) *GraphService {
	return &GraphService{logger: logger}
}

func (s *GraphService) Build(ctx context.Context, deps []entity.Dependency) (*outbound.DependencyGraph, error) {
	g := graph.New(graph.StringHash, graph.Directed())

	nodes := make(map[string]outbound.GraphNode)
	var edges []outbound.GraphEdge

	for _, dep := range deps {
		if _, ok := nodes[dep.Name]; !ok {
			nodes[dep.Name] = outbound.GraphNode{
				ID:    dep.Name,
				Label: shortName(dep.Name),
				Type:  string(dep.Type),
			}
			_ = g.AddVertex(dep.Name)
		}

		if dep.Source != "" {
			if _, ok := nodes[dep.Source]; !ok {
				nodes[dep.Source] = outbound.GraphNode{
					ID:    dep.Source,
					Label: shortName(dep.Source),
					Type:  "file",
				}
				_ = g.AddVertex(dep.Source)
			}
			if err := g.AddEdge(dep.Source, dep.Name); err == nil {
				edges = append(edges, outbound.GraphEdge{
					Source: dep.Source,
					Target: dep.Name,
					Label:  "imports",
				})
			}
		}
	}

	nodeList := make([]outbound.GraphNode, 0, len(nodes))
	for _, n := range nodes {
		nodeList = append(nodeList, n)
	}

	return &outbound.DependencyGraph{
		Nodes: nodeList,
		Edges: edges,
	}, nil
}

func (s *GraphService) DetectCycles(ctx context.Context, depGraph *outbound.DependencyGraph) ([][]string, error) {
	g := graph.New(graph.StringHash, graph.Directed())

	for _, node := range depGraph.Nodes {
		_ = g.AddVertex(node.ID)
	}
	for _, edge := range depGraph.Edges {
		_ = g.AddEdge(edge.Source, edge.Target)
	}

	cycles, err := graph.StronglyConnectedComponents(g)
	if err != nil {
		return nil, fmt.Errorf("detecting cycles: %w", err)
	}

	var result [][]string
	for _, cycle := range cycles {
		if len(cycle) > 1 {
			result = append(result, cycle)
		}
	}
	return result, nil
}

func shortName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}
