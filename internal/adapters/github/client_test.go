package github_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ghclient "github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/github"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
)

func newGHLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newGHTestClient creates a client pointing at the given test server URL.
// go-github's WithEnterpriseURLs routes all calls to that base URL.
func newGHTestClient(t *testing.T, baseURL string) *ghclient.Client {
	t.Helper()
	c, err := ghclient.NewClient(ghclient.Config{
		Token:   "test-token",
		BaseURL: baseURL + "/",
	}, newGHLogger())
	require.NoError(t, err)
	return c
}

func ghTestRepo(t *testing.T) valueobject.RepositoryRef {
	t.Helper()
	ref, err := valueobject.ParseRepositoryRef("myorg/myrepo")
	require.NoError(t, err)
	return ref
}

func ghTestPRNumber(t *testing.T) valueobject.PRNumber {
	t.Helper()
	n, err := valueobject.NewPRNumber(1)
	require.NoError(t, err)
	return n
}

func TestGHNewClient_NoToken(t *testing.T) {
	_, err := ghclient.NewClient(ghclient.Config{}, newGHLogger())
	require.Error(t, err)
}

func TestGHClient_GetPullRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     100,
			"number": 1,
			"title":  "Test PR",
			"body":   "Description",
			"state":  "open",
			"user": map[string]any{
				"id":    42,
				"login": "developer",
			},
			"head": map[string]any{"ref": "feature/test"},
			"base": map[string]any{"ref": "main"},
			"html_url":   "https://github.com/myorg/myrepo/pull/1",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-02T00:00:00Z",
			"labels":     []any{},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	pr, err := c.GetPullRequest(context.Background(), ghTestRepo(t), ghTestPRNumber(t))
	require.NoError(t, err)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "feature/test", pr.SourceBranch)
	assert.Equal(t, "main", pr.TargetBranch)
	assert.Equal(t, "developer", pr.Author.Username)
}

func TestGHClient_GetPullRequest_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, `{"message":"Not Found"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	_, err := c.GetPullRequest(context.Background(), ghTestRepo(t), ghTestPRNumber(t))
	require.Error(t, err)
}

func TestGHClient_ListPullRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":     101,
				"number": 2,
				"title":  "PR 2",
				"state":  "open",
				"user":   map[string]any{"id": 1, "login": "dev"},
				"head":   map[string]any{"ref": "feat"},
				"base":   map[string]any{"ref": "main"},
				"html_url": "",
				"labels": []any{},
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	prs, err := c.ListPullRequests(context.Background(), ghTestRepo(t), outbound.ListPROptions{State: "open", Page: 1, PerPage: 25})
	require.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, "PR 2", prs[0].Title)
}

func TestGHClient_GetPullRequestFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/pulls/1/files", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"filename":  "main.go",
				"status":    "modified",
				"additions": 10,
				"deletions": 2,
				"patch":     "@@ -1,3 +1,3 @@ func main() {}",
			},
			{
				"filename":  "new_file.go",
				"status":    "added",
				"additions": 50,
				"deletions": 0,
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	files, err := c.GetPullRequestFiles(context.Background(), ghTestRepo(t), ghTestPRNumber(t))
	require.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "main.go", files[0].Path)
}

func TestGHClient_GetPullRequestCommits(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/pulls/1/commits", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"sha": "abc123",
				"url": "https://github.com/commit/abc123",
				"commit": map[string]any{
					"message": "feat: add feature",
					"author": map[string]any{
						"name":  "Dev User",
						"email": "dev@example.com",
						"date":  "2024-01-01T00:00:00Z",
					},
				},
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	commits, err := c.GetPullRequestCommits(context.Background(), ghTestRepo(t), ghTestPRNumber(t))
	require.NoError(t, err)
	assert.Len(t, commits, 1)
	assert.Equal(t, "abc123", commits[0].SHA)
	assert.Equal(t, "feat: add feature", commits[0].Message)
}

func TestGHClient_GetPullRequestReviews(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":   1,
				"body": "Looks good!",
				"state": "APPROVED",
				"user": map[string]any{"id": 2, "login": "reviewer"},
				"submitted_at": "2024-01-02T00:00:00Z",
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	reviews, err := c.GetPullRequestReviews(context.Background(), ghTestRepo(t), ghTestPRNumber(t))
	require.NoError(t, err)
	assert.Len(t, reviews, 1)
	assert.Equal(t, "Looks good!", reviews[0].Body)
}

func TestGHClient_GetPullRequestComments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":   99,
				"body": "Nice work!",
				"user": map[string]any{"id": 3, "login": "commenter"},
				"created_at": "2024-01-01T00:00:00Z",
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	comments, err := c.GetPullRequestComments(context.Background(), ghTestRepo(t), ghTestPRNumber(t))
	require.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Equal(t, "Nice work!", comments[0].Body)
	assert.Equal(t, "commenter", comments[0].Author.Username)
}

func TestGHClient_ListBranches(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/branches", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "main"},
			{"name": "develop"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	branches, err := c.ListBranches(context.Background(), ghTestRepo(t))
	require.NoError(t, err)
	assert.Equal(t, []string{"main", "develop"}, branches)
}

func TestGHClient_GetFileContent(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/contents/main.go", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// go-github decodes base64 content
		json.NewEncoder(w).Encode(map[string]any{
			"name":     "main.go",
			"path":     "main.go",
			"type":     "file",
			"encoding": "base64",
			// base64 of "package main\n"
			"content": "cGFja2FnZSBtYWluCg==\n",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	content, err := c.GetFileContent(context.Background(), ghTestRepo(t), "main.go", "main")
	require.NoError(t, err)
	assert.Contains(t, content, "package main")
}

func TestGHClient_CompareRepositories(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/source", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":        1,
			"name":      "source",
			"full_name": "myorg/source",
			"html_url":  "https://github.com/myorg/source",
			"language":  "Go",
		})
	})
	mux.HandleFunc("/api/v3/repos/myorg/target", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":        2,
			"name":      "target",
			"full_name": "myorg/target",
			"html_url":  "https://github.com/myorg/target",
			"language":  "Go",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	source, _ := valueobject.ParseRepositoryRef("myorg/source")
	target, _ := valueobject.ParseRepositoryRef("myorg/target")
	result, err := c.CompareRepositories(context.Background(), source, target)
	require.NoError(t, err)
	assert.Equal(t, "source", result.SourceRepo.Name)
	assert.Equal(t, "target", result.TargetRepo.Name)
}

func TestGHClient_GetPullRequest_Merged(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":        200,
			"number":    1,
			"title":     "Merged PR",
			"state":     "closed",
			"merged_at": "2024-01-05T10:00:00Z",
			"user":      map[string]any{"id": 1, "login": "dev"},
			"head":      map[string]any{"ref": "feat"},
			"base":      map[string]any{"ref": "main"},
			"html_url":  "",
			"labels":    []any{},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	pr, err := c.GetPullRequest(context.Background(), ghTestRepo(t), ghTestPRNumber(t))
	require.NoError(t, err)
	assert.Equal(t, "Merged PR", pr.Title)
	assert.NotNil(t, pr.MergedAt)
}

func TestGHClient_GetPullRequest_WithLabels(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/myorg/myrepo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     300,
			"number": 1,
			"title":  "Labelled PR",
			"state":  "open",
			"user":   map[string]any{"id": 1, "login": "dev"},
			"head":   map[string]any{"ref": "feat"},
			"base":   map[string]any{"ref": "main"},
			"html_url": "",
			"labels": []any{
				map[string]any{"name": "bug"},
				map[string]any{"name": "priority:high"},
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newGHTestClient(t, srv.URL)
	pr, err := c.GetPullRequest(context.Background(), ghTestRepo(t), ghTestPRNumber(t))
	require.NoError(t, err)
	assert.Contains(t, pr.Labels, "bug")
	assert.Contains(t, pr.Labels, "priority:high")
}
