package architecture

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
)

func TestDetectK8sResource_AllKinds(t *testing.T) {
	tests := []struct {
		patch    string
		expected string
	}{
		{"kind: Deployment\nspec:", "Deployment"},
		{"kind: Service\nspec:", "Service"},
		{"kind: ConfigMap\ndata:", "ConfigMap"},
		{"kind: Secret\ndata:", "Secret"},
		{"kind: Ingress\n", ""},
	}
	for _, tc := range tests {
		t.Run(tc.expected+tc.patch, func(t *testing.T) {
			assert.Equal(t, tc.expected, detectK8sResource(tc.patch))
		})
	}
}

func TestExtractEnvVars(t *testing.T) {
	patch := `
+func main() {
+	db := os.Getenv("DATABASE_URL")
+	port := os.Getenv("PORT")
+	_ = db + port
+}
`
	vars := extractEnvVars(patch)
	assert.Contains(t, vars, "DATABASE_URL")
	assert.Contains(t, vars, "PORT")
}

func TestExtractEnvVars_None(t *testing.T) {
	vars := extractEnvVars("no env vars here")
	assert.Empty(t, vars)
}

func TestExtractEnvVars_MalformedQuote(t *testing.T) {
	// Line with os.Getenv but no matching quotes
	patch := `os.Getenv(noQuotes)`
	vars := extractEnvVars(patch)
	assert.Empty(t, vars)
}

func TestDetectLayer_AllCases(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"internal/domain/entity/foo.go", "domain"},
		{"internal/application/usecase/foo.go", "application"},
		{"internal/usecase/foo.go", "application"},
		{"internal/adapters/github/client.go", "adapter"},
		{"internal/adapter/http.go", "adapter"},
		{"internal/port/inbound/port.go", "port"},
		{"internal/service/analysis/analyzer.go", "service"},
		{"internal/cmd/server/main.go", "presentation"},
		{"internal/repository/repo.go", "repository"},
		{"internal/pkg/errors/errors.go", "shared"},
		{"internal/shared/utils/helper.go", "shared"},
		{"internal/config/settings.go", "configuration"},
		{"random/path.go", ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.expected, detectLayer(tc.path))
		})
	}
}

func TestDetectPatterns_AllPatterns(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
	}{
		{"user_repository.go", "Repository Pattern"},
		{"create_usecase.go", "Use Case / Interactor"},
		{"create_use_case.go", "Use Case / Interactor"},
		{"api_handler.go", "Handler / Controller"},
		{"web_controller.go", "Handler / Controller"},
		{"grpc_adapter.go", "Adapter Pattern"},
		{"inbound_port.go", "Port / Interface"},
		{"user_factory.go", "Factory Pattern"},
		{"auth_middleware.go", "Middleware / Decorator"},
	}
	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			patterns := detectPatterns(tc.path)
			assert.Contains(t, patterns, tc.pattern)
		})
	}
}

func TestDetectConfigImpact_WithEnvVars(t *testing.T) {
	m := &Mapper{}
	files := []entity.ChangedFile{
		{
			Path:  "internal/config/app.go",
			Patch: `+func load() { db := os.Getenv("DATABASE_URL") }`,
		},
	}
	impact := m.DetectConfigImpact(files)
	assert.NotNil(t, impact)
	assert.NotEmpty(t, impact.EnvVarsAdded)
	assert.Contains(t, impact.EnvVarsAdded, "DATABASE_URL")
}

func TestDetectKubernetesImpact_WithResources(t *testing.T) {
	m := &Mapper{}
	files := []entity.ChangedFile{
		{
			Path:  "k8s/deployment.yaml",
			Patch: "kind: Deployment\nspec:\n  replicas: 3",
		},
		{
			Path:  "k8s/service.yaml",
			Patch: "kind: Service\nspec:",
		},
	}
	impact := m.DetectKubernetesImpact(files)
	assert.NotNil(t, impact)
	assert.Contains(t, impact.ResourcesAffected, "Deployment")
	assert.Contains(t, impact.ResourcesAffected, "Service")
	assert.Len(t, impact.ManifestsChanged, 2)
}

func TestDetectKubernetesImpact_EmptyPatch(t *testing.T) {
	m := &Mapper{}
	files := []entity.ChangedFile{
		{
			Path:  "helm/templates/deployment.yaml",
			Patch: "# just a comment",
		},
	}
	impact := m.DetectKubernetesImpact(files)
	assert.NotNil(t, impact)
	assert.Len(t, impact.ManifestsChanged, 1)
	assert.Empty(t, impact.ResourcesAffected)
}
