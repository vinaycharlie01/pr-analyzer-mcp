package architecture

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
)

// Mapper generates architecture impact analysis.
type Mapper struct {
	logger *slog.Logger
}

func NewMapper(logger *slog.Logger) *Mapper {
	return &Mapper{logger: logger}
}

func (m *Mapper) MapImpact(ctx context.Context, pr *entity.PullRequest) (*entity.ArchitectureImpact, error) {
	impact := &entity.ArchitectureImpact{
		LayersAffected:     []string{},
		PatternsUsed:       []string{},
		NewComponents:      []string{},
		ModifiedComponents: []string{},
	}

	layerSet := make(map[string]bool)
	patternSet := make(map[string]bool)

	for _, f := range pr.Files {
		if layer := detectLayer(f.Path); layer != "" {
			layerSet[layer] = true
		}
		for _, pattern := range detectPatterns(f.Path) {
			patternSet[pattern] = true
		}
		switch f.Status {
		case entity.FileStatusAdded:
			impact.NewComponents = append(impact.NewComponents, f.Path)
		case entity.FileStatusModified:
			impact.ModifiedComponents = append(impact.ModifiedComponents, f.Path)
		}
	}

	for l := range layerSet {
		impact.LayersAffected = append(impact.LayersAffected, l)
	}
	for p := range patternSet {
		impact.PatternsUsed = append(impact.PatternsUsed, p)
	}

	impact.Description = m.buildDescription(impact, pr)
	return impact, nil
}

func (m *Mapper) DetectDatabaseImpact(files []entity.ChangedFile) *entity.DatabaseImpact {
	impact := &entity.DatabaseImpact{}

	for _, f := range files {
		if isMigrationFile(f.Path) {
			impact.HasMigrations = true
			impact.Migrations = append(impact.Migrations, entity.DatabaseMigration{
				Name:       f.Path,
				Type:       "sql",
				Reversible: true,
			})
		}
		if strings.Contains(strings.ToLower(f.Path), "schema") ||
			strings.Contains(strings.ToLower(f.Path), "model") {
			impact.TablesAffected = append(impact.TablesAffected, f.Path)
		}
	}

	if impact.HasMigrations {
		impact.Description = fmt.Sprintf("Database migrations detected: %d migration file(s) changed", len(impact.Migrations))
	} else {
		impact.Description = "No database migrations detected"
	}

	return impact
}

func (m *Mapper) DetectConfigImpact(files []entity.ChangedFile) *entity.ConfigurationImpact {
	impact := &entity.ConfigurationImpact{}

	for _, f := range files {
		lower := strings.ToLower(f.Path)
		if isConfigFile(lower) {
			impact.ConfigFilesChanged = append(impact.ConfigFilesChanged, f.Path)
		}
		if strings.Contains(f.Patch, "os.Getenv") || strings.Contains(f.Patch, "env.") {
			// Detect new env var usages from patch
			impact.EnvVarsAdded = append(impact.EnvVarsAdded, extractEnvVars(f.Patch)...)
		}
	}

	if len(impact.ConfigFilesChanged) > 0 {
		impact.Description = fmt.Sprintf("Configuration changes: %d config file(s) modified", len(impact.ConfigFilesChanged))
	} else {
		impact.Description = "No configuration file changes detected"
	}

	return impact
}

func (m *Mapper) DetectKubernetesImpact(files []entity.ChangedFile) *entity.KubernetesImpact {
	impact := &entity.KubernetesImpact{}

	for _, f := range files {
		lower := strings.ToLower(f.Path)
		if isK8sFile(lower) {
			impact.ManifestsChanged = append(impact.ManifestsChanged, f.Path)
			resource := detectK8sResource(f.Patch)
			if resource != "" {
				impact.ResourcesAffected = append(impact.ResourcesAffected, resource)
			}
		}
	}

	if len(impact.ManifestsChanged) > 0 {
		impact.Description = fmt.Sprintf("Kubernetes changes: %d manifest(s) modified", len(impact.ManifestsChanged))
	} else {
		impact.Description = "No Kubernetes manifest changes detected"
	}

	return impact
}

func (m *Mapper) buildDescription(impact *entity.ArchitectureImpact, pr *entity.PullRequest) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PR #%d affects the following architectural layers: ", pr.Number))
	if len(impact.LayersAffected) > 0 {
		sb.WriteString(strings.Join(impact.LayersAffected, ", "))
	} else {
		sb.WriteString("none detected")
	}
	sb.WriteString(". ")
	if len(impact.PatternsUsed) > 0 {
		sb.WriteString(fmt.Sprintf("Design patterns identified: %s.", strings.Join(impact.PatternsUsed, ", ")))
	}
	return sb.String()
}

func detectLayer(path string) string {
	switch {
	case strings.Contains(path, "/domain/"):
		return "domain"
	case strings.Contains(path, "/application/") || strings.Contains(path, "/usecase/"):
		return "application"
	case strings.Contains(path, "/adapter") || strings.Contains(path, "/adapters/"):
		return "adapter"
	case strings.Contains(path, "/port/"):
		return "port"
	case strings.Contains(path, "/service/"):
		return "service"
	case strings.Contains(path, "/cmd/"):
		return "presentation"
	case strings.Contains(path, "/repository/"):
		return "repository"
	case strings.Contains(path, "/pkg/") || strings.Contains(path, "/shared/"):
		return "shared"
	case strings.Contains(path, "/config"):
		return "configuration"
	default:
		return ""
	}
}

func detectPatterns(path string) []string {
	var patterns []string
	lower := strings.ToLower(path)

	if strings.Contains(lower, "repository") {
		patterns = append(patterns, "Repository Pattern")
	}
	if strings.Contains(lower, "usecase") || strings.Contains(lower, "use_case") {
		patterns = append(patterns, "Use Case / Interactor")
	}
	if strings.Contains(lower, "handler") || strings.Contains(lower, "controller") {
		patterns = append(patterns, "Handler / Controller")
	}
	if strings.Contains(lower, "adapter") {
		patterns = append(patterns, "Adapter Pattern")
	}
	if strings.Contains(lower, "port") {
		patterns = append(patterns, "Port / Interface")
	}
	if strings.Contains(lower, "factory") {
		patterns = append(patterns, "Factory Pattern")
	}
	if strings.Contains(lower, "middleware") {
		patterns = append(patterns, "Middleware / Decorator")
	}
	return patterns
}

func isMigrationFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "migration") ||
		strings.Contains(lower, "/migrate/") ||
		(strings.HasSuffix(lower, ".sql") && !strings.Contains(lower, "test"))
}

func isConfigFile(path string) bool {
	return strings.HasSuffix(path, ".yaml") ||
		strings.HasSuffix(path, ".yml") ||
		strings.HasSuffix(path, ".env") ||
		strings.HasSuffix(path, ".toml") ||
		strings.HasSuffix(path, ".json") ||
		strings.Contains(path, "config")
}

func isK8sFile(path string) bool {
	return strings.Contains(path, "k8s") ||
		strings.Contains(path, "kubernetes") ||
		strings.Contains(path, "helm") ||
		strings.Contains(path, "chart") ||
		(strings.HasSuffix(path, ".yaml") && (strings.Contains(path, "deploy") || strings.Contains(path, "service")))
}

func detectK8sResource(patch string) string {
	if strings.Contains(patch, "kind: Deployment") {
		return "Deployment"
	}
	if strings.Contains(patch, "kind: Service") {
		return "Service"
	}
	if strings.Contains(patch, "kind: ConfigMap") {
		return "ConfigMap"
	}
	if strings.Contains(patch, "kind: Secret") {
		return "Secret"
	}
	return ""
}

func extractEnvVars(patch string) []string {
	var vars []string
	lines := strings.Split(patch, "\n")
	for _, line := range lines {
		if strings.Contains(line, "os.Getenv(") {
			start := strings.Index(line, `"`)
			end := strings.LastIndex(line, `"`)
			if start >= 0 && end > start {
				vars = append(vars, line[start+1:end])
			}
		}
	}
	return vars
}
