package architecture_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/service/architecture"
)

func newTestMapper() *architecture.Mapper {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return architecture.NewMapper(log)
}

func TestMapper_MapImpact(t *testing.T) {
	m := newTestMapper()
	ctx := context.Background()

	pr := &entity.PullRequest{
		Number: 1,
		Title:  "Add feature",
		Files: []entity.ChangedFile{
			{Path: "internal/domain/entity/user.go", Status: entity.FileStatusAdded},
			{Path: "internal/adapters/github/client.go", Status: entity.FileStatusModified},
			{Path: "internal/application/usecase/create_user.go", Status: entity.FileStatusAdded},
			{Path: "internal/port/inbound/user_port.go", Status: entity.FileStatusAdded},
		},
	}

	impact, err := m.MapImpact(ctx, pr)
	require.NoError(t, err)
	assert.NotNil(t, impact)
	assert.Contains(t, impact.LayersAffected, "domain")
	assert.Contains(t, impact.LayersAffected, "adapter")
	assert.Contains(t, impact.LayersAffected, "application")
	assert.Len(t, impact.NewComponents, 3)
	assert.Len(t, impact.ModifiedComponents, 1)
	assert.NotEmpty(t, impact.Description)
}

func TestMapper_DetectDatabaseImpact(t *testing.T) {
	m := newTestMapper()

	tests := []struct {
		name          string
		files         []entity.ChangedFile
		wantMigration bool
	}{
		{
			name: "has migration file",
			files: []entity.ChangedFile{
				{Path: "db/migrations/001_create_users.sql", Status: entity.FileStatusAdded},
			},
			wantMigration: true,
		},
		{
			name: "no migration files",
			files: []entity.ChangedFile{
				{Path: "internal/service/user.go", Status: entity.FileStatusModified},
			},
			wantMigration: false,
		},
		{
			name:          "empty files",
			files:         []entity.ChangedFile{},
			wantMigration: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			impact := m.DetectDatabaseImpact(tc.files)
			assert.NotNil(t, impact)
			assert.Equal(t, tc.wantMigration, impact.HasMigrations)
			assert.NotEmpty(t, impact.Description)
		})
	}
}

func TestMapper_DetectConfigImpact(t *testing.T) {
	m := newTestMapper()

	// .env.example ends in ".example", not ".env", so only config.yaml is matched.
	files := []entity.ChangedFile{
		{Path: "configs/config.yaml", Status: entity.FileStatusModified},
		{Path: "configs/settings.json", Status: entity.FileStatusModified},
	}

	impact := m.DetectConfigImpact(files)
	assert.NotNil(t, impact)
	assert.Len(t, impact.ConfigFilesChanged, 2)
	assert.NotEmpty(t, impact.Description)
}

func TestMapper_DetectKubernetesImpact(t *testing.T) {
	m := newTestMapper()

	tests := []struct {
		name        string
		files       []entity.ChangedFile
		wantChanged bool
	}{
		{
			name: "has k8s manifests",
			files: []entity.ChangedFile{
				{Path: "k8s/deployment.yaml", Status: entity.FileStatusModified, Patch: "kind: Deployment"},
			},
			wantChanged: true,
		},
		{
			name: "no k8s files",
			files: []entity.ChangedFile{
				{Path: "internal/service/user.go", Status: entity.FileStatusModified},
			},
			wantChanged: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			impact := m.DetectKubernetesImpact(tc.files)
			assert.NotNil(t, impact)
			assert.Equal(t, tc.wantChanged, len(impact.ManifestsChanged) > 0)
		})
	}
}

func BenchmarkMapper_MapImpact(b *testing.B) {
	m := newTestMapper()
	ctx := context.Background()
	pr := &entity.PullRequest{
		Number: 1,
		Title:  "Large PR",
	}
	for i := 0; i < 30; i++ {
		pr.Files = append(pr.Files, entity.ChangedFile{
			Path:   "internal/service/some/file.go",
			Status: entity.FileStatusModified,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.MapImpact(ctx, pr)
	}
}
