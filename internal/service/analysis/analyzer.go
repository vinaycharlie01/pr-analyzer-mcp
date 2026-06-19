package analysis

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/dominikbraun/graph"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
)

// Analyzer implements static code analysis for PR changes.
type Analyzer struct {
	logger *slog.Logger
}

func NewAnalyzer(logger *slog.Logger) *Analyzer {
	return &Analyzer{logger: logger}
}

func (a *Analyzer) AnalyzeDependencies(ctx context.Context, files []entity.ChangedFile) ([]entity.Dependency, error) {
	deps := make(map[string]*entity.Dependency)

	for _, f := range files {
		if !isGoFile(f.Path) {
			continue
		}
		if f.Patch == "" {
			continue
		}
		extracted := a.extractImports(f.Patch)
		for _, imp := range extracted {
			depType := classifyDependency(imp)
			key := imp
			if _, exists := deps[key]; !exists {
				deps[key] = &entity.Dependency{
					Name:        imp,
					Type:        depType,
					Required:    true,
					Description: describeImport(imp),
					Source:      f.Path,
				}
			}
		}
	}

	result := make([]entity.Dependency, 0, len(deps))
	for _, d := range deps {
		result = append(result, *d)
	}
	return result, nil
}

func (a *Analyzer) AnalyzeArchitecture(ctx context.Context, files []entity.ChangedFile) (*entity.ArchitectureImpact, error) {
	impact := &entity.ArchitectureImpact{
		LayersAffected:     []string{},
		PatternsUsed:       []string{},
		NewComponents:      []string{},
		ModifiedComponents: []string{},
	}

	layerSet := make(map[string]bool)
	patternSet := make(map[string]bool)

	for _, f := range files {
		layer := detectLayer(f.Path)
		if layer != "" {
			layerSet[layer] = true
		}

		patterns := detectPatterns(f.Path, f.Patch)
		for _, p := range patterns {
			patternSet[p] = true
		}

		switch f.Status {
		case entity.FileStatusAdded:
			impact.NewComponents = append(impact.NewComponents, f.Path)
		case entity.FileStatusModified, entity.FileStatusRenamed:
			impact.ModifiedComponents = append(impact.ModifiedComponents, f.Path)
		}
	}

	for l := range layerSet {
		impact.LayersAffected = append(impact.LayersAffected, l)
	}
	for p := range patternSet {
		impact.PatternsUsed = append(impact.PatternsUsed, p)
	}

	impact.Description = a.buildArchitectureDescription(impact)
	return impact, nil
}

func (a *Analyzer) FindRelatedFiles(ctx context.Context, files []entity.ChangedFile, repoPath string) ([]string, error) {
	related := make(map[string]bool)

	for _, f := range files {
		if !isGoFile(f.Path) {
			continue
		}
		pkg := packageFromPath(f.Path)
		if pkg != "" {
			testFile := strings.TrimSuffix(f.Path, ".go") + "_test.go"
			related[testFile] = true

			dir := filepath.Dir(f.Path)
			related[dir] = true
		}
	}

	result := make([]string, 0, len(related))
	for r := range related {
		result = append(result, r)
	}
	return result, nil
}

func (a *Analyzer) BuildDependencyGraph(ctx context.Context, packages []string) (*outbound.DependencyGraph, error) {
	g := graph.New(graph.StringHash, graph.Directed())

	for _, pkg := range packages {
		_ = g.AddVertex(pkg)
	}

	depGraph := &outbound.DependencyGraph{
		Nodes: make([]outbound.GraphNode, 0, len(packages)),
		Edges: []outbound.GraphEdge{},
	}

	for _, pkg := range packages {
		depGraph.Nodes = append(depGraph.Nodes, outbound.GraphNode{
			ID:    pkg,
			Label: filepath.Base(pkg),
			Type:  "package",
		})
	}

	return depGraph, nil
}

func (a *Analyzer) extractImports(patch string) []string {
	// Parse imports from a diff patch
	var imports []string
	lines := strings.Split(patch, "\n")
	inImportBlock := false

	for _, line := range lines {
		clean := strings.TrimLeft(line, "+- \t")
		if strings.Contains(clean, "import (") {
			inImportBlock = true
			continue
		}
		if inImportBlock {
			if strings.TrimSpace(clean) == ")" {
				inImportBlock = false
				continue
			}
			imp := strings.Trim(clean, `"' `)
			if imp != "" && strings.Contains(imp, "/") {
				imports = append(imports, imp)
			}
			continue
		}
		if strings.HasPrefix(clean, `import "`) {
			imp := strings.TrimPrefix(clean, `import "`)
			imp = strings.TrimSuffix(imp, `"`)
			if imp != "" {
				imports = append(imports, imp)
			}
		}
	}
	return imports
}

func (a *Analyzer) parseGoFile(src string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parsing Go file: %w", err)
	}

	imports := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			imports = append(imports, path)
		}
	}
	return imports, nil
}

func (a *Analyzer) visitAST(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		return n != nil
	})
}

func (a *Analyzer) buildArchitectureDescription(impact *entity.ArchitectureImpact) string {
	var sb strings.Builder
	sb.WriteString("Architecture impact analysis: ")
	if len(impact.LayersAffected) > 0 {
		sb.WriteString(fmt.Sprintf("affects %s layers. ", strings.Join(impact.LayersAffected, ", ")))
	}
	if len(impact.NewComponents) > 0 {
		sb.WriteString(fmt.Sprintf("%d new components added. ", len(impact.NewComponents)))
	}
	if len(impact.ModifiedComponents) > 0 {
		sb.WriteString(fmt.Sprintf("%d existing components modified.", len(impact.ModifiedComponents)))
	}
	return sb.String()
}

func isGoFile(path string) bool {
	return strings.HasSuffix(path, ".go")
}

func packageFromPath(path string) string {
	return filepath.Dir(path)
}

func detectLayer(path string) string {
	switch {
	case strings.Contains(path, "/domain/"):
		return "domain"
	case strings.Contains(path, "/application/"):
		return "application"
	case strings.Contains(path, "/adapters/"):
		return "adapter"
	case strings.Contains(path, "/port/"):
		return "port"
	case strings.Contains(path, "/service/"):
		return "service"
	case strings.Contains(path, "/cmd/"):
		return "presentation"
	case strings.Contains(path, "/repository/"):
		return "repository"
	case strings.Contains(path, "/pkg/"):
		return "shared"
	default:
		return ""
	}
}

func detectPatterns(path, patch string) []string {
	var patterns []string
	content := strings.ToLower(patch + path)

	if strings.Contains(content, "interface") {
		patterns = append(patterns, "Interface")
	}
	if strings.Contains(content, "repository") {
		patterns = append(patterns, "Repository Pattern")
	}
	if strings.Contains(content, "usecase") || strings.Contains(content, "use_case") {
		patterns = append(patterns, "Use Case Pattern")
	}
	if strings.Contains(content, "factory") {
		patterns = append(patterns, "Factory Pattern")
	}
	if strings.Contains(content, "handler") {
		patterns = append(patterns, "Handler Pattern")
	}
	if strings.Contains(content, "middleware") {
		patterns = append(patterns, "Middleware Pattern")
	}
	return patterns
}

func classifyDependency(imp string) entity.DependencyType {
	switch {
	case strings.HasPrefix(imp, "github.com/") || strings.HasPrefix(imp, "golang.org/") || strings.HasPrefix(imp, "go."):
		return entity.DependencyTypePackage
	case strings.Contains(imp, "database") || strings.Contains(imp, "sql") || strings.Contains(imp, "db"):
		return entity.DependencyTypeDatabase
	default:
		return entity.DependencyTypeInternal
	}
}

func describeImport(imp string) string {
	parts := strings.Split(imp, "/")
	if len(parts) == 0 {
		return imp
	}
	return parts[len(parts)-1] + " package"
}
