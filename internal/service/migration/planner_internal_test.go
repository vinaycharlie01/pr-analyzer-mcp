package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
)

func makeChangedFiles() []entity.ChangedFile {
	return []entity.ChangedFile{
		{Path: "internal/service/new.go", Status: entity.FileStatusAdded},
		{Path: "internal/domain/model.go", Status: entity.FileStatusAdded},
		{Path: "main.go", Status: entity.FileStatusModified},
	}
}

func makeRootChangedFiles() []entity.ChangedFile {
	return []entity.ChangedFile{
		{Path: "main.go", Status: entity.FileStatusAdded},
	}
}

func TestDirectoryOf(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"internal/service/foo.go", "internal/service"},
		{"main.go", ""},
		{"a/b/c/file.go", "a/b/c"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.expected, directoryOf(tc.path))
		})
	}
}

func TestBuildFileMigrationCommands_WithAddedFiles(t *testing.T) {
	p := &Planner{}

	files := makeChangedFiles()
	cmds := p.buildFileMigrationCommands(files)

	// Files with directories should generate mkdir -p commands
	assert.NotEmpty(t, cmds)
	hasMkdir := false
	hasCopy := false
	for _, cmd := range cmds {
		if len(cmd) > 8 && cmd[:8] == "mkdir -p" {
			hasMkdir = true
		}
		if len(cmd) > 7 && cmd[:7] == "# Copy:" {
			hasCopy = true
		}
	}
	assert.True(t, hasMkdir, "should generate mkdir commands for files in subdirs")
	assert.True(t, hasCopy, "should generate copy comments")
}

func TestBuildFileMigrationCommands_RootFile(t *testing.T) {
	p := &Planner{}
	files := makeRootChangedFiles()
	cmds := p.buildFileMigrationCommands(files)

	// Root-level added file should not generate mkdir
	hasMkdir := false
	for _, cmd := range cmds {
		if len(cmd) > 8 && cmd[:8] == "mkdir -p" {
			hasMkdir = true
		}
	}
	assert.False(t, hasMkdir, "root-level files should not generate mkdir")
}
