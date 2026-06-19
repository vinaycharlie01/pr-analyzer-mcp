package valueobject_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
)

func TestNewRepositoryRef(t *testing.T) {
	tests := []struct {
		name        string
		owner       string
		repoName    string
		expectError bool
	}{
		{name: "valid", owner: "myorg", repoName: "myrepo", expectError: false},
		{name: "empty owner", owner: "", repoName: "myrepo", expectError: true},
		{name: "empty name", owner: "myorg", repoName: "", expectError: true},
		{name: "spaces trimmed", owner: "  myorg  ", repoName: "  myrepo  ", expectError: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := valueobject.NewRepositoryRef(tc.owner, tc.repoName)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "myorg", ref.Owner())
				assert.Equal(t, "myrepo", ref.Name())
				assert.Equal(t, "myorg/myrepo", ref.String())
			}
		})
	}
}

func TestParseRepositoryRef(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		wantOwner   string
		wantName    string
	}{
		{name: "valid", input: "myorg/myrepo", wantOwner: "myorg", wantName: "myrepo"},
		{name: "no slash", input: "myrepo", expectError: true},
		{name: "empty", input: "", expectError: true},
		{name: "nested path", input: "myorg/group/repo", wantOwner: "myorg", wantName: "group/repo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := valueobject.ParseRepositoryRef(tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantOwner, ref.Owner())
				assert.Equal(t, tc.wantName, ref.Name())
			}
		})
	}
}

func TestPRNumber(t *testing.T) {
	tests := []struct {
		name        string
		value       int
		expectError bool
	}{
		{name: "valid", value: 42, expectError: false},
		{name: "zero", value: 0, expectError: true},
		{name: "negative", value: -1, expectError: true},
		{name: "large number", value: 99999, expectError: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prNum, err := valueobject.NewPRNumber(tc.value)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.value, prNum.Value())
				assert.Equal(t, fmt.Sprintf("#%d", tc.value), prNum.String())
			}
		})
	}
}
