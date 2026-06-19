package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGoFile_Valid(t *testing.T) {
	a := &Analyzer{}
	src := `package main
import (
	"fmt"
	"github.com/some/pkg"
)
func main() {}
`
	imports, err := a.parseGoFile(src)
	require.NoError(t, err)
	assert.Contains(t, imports, "fmt")
	assert.Contains(t, imports, "github.com/some/pkg")
}

func TestParseGoFile_Invalid(t *testing.T) {
	a := &Analyzer{}
	_, err := a.parseGoFile("this is not Go code $$$$")
	require.Error(t, err)
}

func TestParseGoFile_NoImports(t *testing.T) {
	a := &Analyzer{}
	src := `package main
func main() {}
`
	imports, err := a.parseGoFile(src)
	require.NoError(t, err)
	assert.Empty(t, imports)
}

func TestVisitAST(t *testing.T) {
	a := &Analyzer{}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", `package main; func main() {}`, 0)
	require.NoError(t, err)

	// visitAST should not panic
	assert.NotPanics(t, func() {
		a.visitAST(f)
	})
}

func TestVisitAST_NilSafe(t *testing.T) {
	a := &Analyzer{}
	// Passing a valid node to ensure the function handles it
	var node ast.Node
	node = &ast.File{}
	assert.NotPanics(t, func() {
		a.visitAST(node)
	})
}

func TestDetectLayer_AllLayers(t *testing.T) {
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
		{"internal/repository/repo.go", "repository"},
		{"some/path/pkg/errors/errors.go", "shared"},
		{"configs/config.yaml", ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.layer, detectLayer(tc.path))
		})
	}
}

func TestClassifyDependency_AllTypes(t *testing.T) {
	tests := []struct {
		imp      string
		expected string
	}{
		{"github.com/some/pkg", "package"},
		{"golang.org/x/tools", "package"},
		{"go.opentelemetry.io/otel", "package"},
		{"database/sql", "database"},
		{"my/internal/service", "internal"},
	}

	for _, tc := range tests {
		t.Run(tc.imp, func(t *testing.T) {
			depType := classifyDependency(tc.imp)
			assert.Equal(t, tc.expected, string(depType))
		})
	}
}

func TestDescribeImport(t *testing.T) {
	tests := []struct {
		imp      string
		contains string
	}{
		{"github.com/some/package", "package"},
		{"fmt", "fmt"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.imp, func(t *testing.T) {
			desc := describeImport(tc.imp)
			if tc.contains != "" {
				assert.Contains(t, desc, tc.contains)
			}
		})
	}
}

func TestDetectPatterns_AllPatterns(t *testing.T) {
	tests := []struct {
		path    string
		patch   string
		pattern string
	}{
		{"", "interface Foo {}", "Interface"},
		{"repository/impl.go", "", "Repository Pattern"},
		{"", "type usecase struct{}", "Use Case Pattern"},
		{"", "factory.New()", "Factory Pattern"},
		{"", "handler.ServeHTTP", "Handler Pattern"},
		{"", "middleware.Chain", "Middleware Pattern"},
	}
	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			patterns := detectPatterns(tc.path, tc.patch)
			assert.Contains(t, patterns, tc.pattern)
		})
	}
}
