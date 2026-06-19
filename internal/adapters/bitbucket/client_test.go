package bitbucket_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/adapters/bitbucket"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestClient(t *testing.T, baseURL string) *bitbucket.Client {
	t.Helper()
	c, err := bitbucket.NewClient(bitbucket.Config{
		BaseURL: baseURL,
		Token:   "test-token",
	}, newTestLogger())
	require.NoError(t, err)
	return c
}

func testRepo(t *testing.T) valueobject.RepositoryRef {
	t.Helper()
	r, err := valueobject.ParseRepositoryRef("myorg/myrepo")
	require.NoError(t, err)
	return r
}

func testPRNumber(t *testing.T) valueobject.PRNumber {
	t.Helper()
	n, err := valueobject.NewPRNumber(1)
	require.NoError(t, err)
	return n
}

func TestNewClient_NoCredentials(t *testing.T) {
	_, err := bitbucket.NewClient(bitbucket.Config{BaseURL: "http://localhost"}, newTestLogger())
	require.Error(t, err)
}

func TestClient_GetPullRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/pullrequests/1")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":          1,
			"title":       "Test PR",
			"description": "Description",
			"state":       "OPEN",
			"author": map[string]any{
				"display_name": "Dev User",
				"nickname":     "dev",
				"uuid":         "{uuid-1}",
			},
			"source": map[string]any{
				"branch": map[string]any{"name": "feature/test"},
			},
			"destination": map[string]any{
				"branch": map[string]any{"name": "main"},
			},
			"created_on": "2024-01-01T00:00:00Z",
			"updated_on": "2024-01-02T00:00:00Z",
			"links": map[string]any{
				"html": map[string]any{"href": "https://bitbucket.org/myorg/myrepo/pull-requests/1"},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	pr, err := c.GetPullRequest(context.Background(), testRepo(t), testPRNumber(t))
	require.NoError(t, err)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "feature/test", pr.SourceBranch)
	assert.Equal(t, "main", pr.TargetBranch)
	assert.Equal(t, "dev", pr.Author.Username)
}

func TestClient_GetPullRequest_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"Not found"}}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetPullRequest(context.Background(), testRepo(t), testPRNumber(t))
	require.Error(t, err)
}

func TestClient_ListPullRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"id":    1,
					"title": "PR 1",
					"state": "OPEN",
					"author": map[string]any{"display_name": "Dev", "nickname": "dev", "uuid": "{1}"},
					"source":      map[string]any{"branch": map[string]any{"name": "feat"}},
					"destination": map[string]any{"branch": map[string]any{"name": "main"}},
					"created_on":  "2024-01-01T00:00:00Z",
					"updated_on":  "2024-01-02T00:00:00Z",
					"links": map[string]any{"html": map[string]any{"href": ""}},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	prs, err := c.ListPullRequests(context.Background(), testRepo(t), outbound.ListPROptions{State: "OPEN", Page: 1, PerPage: 25})
	require.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, "PR 1", prs[0].Title)
}

func TestClient_GetPullRequestFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"status":        "modified",
					"lines_added":   10,
					"lines_removed": 2,
					"new":           map[string]any{"path": "main.go"},
					"old":           map[string]any{"path": "main.go"},
				},
				{
					"status":      "added",
					"lines_added": 50,
					"new":         map[string]any{"path": "new_file.go"},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	files, err := c.GetPullRequestFiles(context.Background(), testRepo(t), testPRNumber(t))
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestClient_GetPullRequestCommits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"hash":    "abc123",
					"message": "feat: add feature",
					"author": map[string]any{
						"user": map[string]any{
							"display_name": "Dev",
							"nickname":     "dev",
						},
					},
					"date":  "2024-01-01T00:00:00Z",
					"links": map[string]any{"html": map[string]any{"href": "https://bb.org"}},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	commits, err := c.GetPullRequestCommits(context.Background(), testRepo(t), testPRNumber(t))
	require.NoError(t, err)
	assert.Len(t, commits, 1)
	assert.Equal(t, "abc123", commits[0].SHA)
}

func TestClient_GetPullRequestComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"id": 1,
					"content": map[string]any{"raw": "Nice work!"},
					"author":  map[string]any{"display_name": "Reviewer", "nickname": "reviewer"},
					"created_on": "2024-01-01T00:00:00Z",
					"inline": map[string]any{"path": "main.go", "to": 42},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	comments, err := c.GetPullRequestComments(context.Background(), testRepo(t), testPRNumber(t))
	require.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Equal(t, "Nice work!", comments[0].Body)
	assert.Equal(t, "main.go", comments[0].FilePath)
	assert.Equal(t, 42, comments[0].Line)
}

func TestClient_ListBranches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"name": "main"},
				{"name": "develop"},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	branches, err := c.ListBranches(context.Background(), testRepo(t))
	require.NoError(t, err)
	assert.Equal(t, []string{"main", "develop"}, branches)
}

func TestClient_GetFileContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("package main\n\nfunc main() {}\n"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	content, err := c.GetFileContent(context.Background(), testRepo(t), "main.go", "main")
	require.NoError(t, err)
	assert.Contains(t, content, "package main")
}

func TestClient_CompareRepositories(t *testing.T) {
	c := newTestClient(t, "http://localhost:9999")
	source, _ := valueobject.ParseRepositoryRef("myorg/source")
	target, _ := valueobject.ParseRepositoryRef("myorg/target")

	result, err := c.CompareRepositories(context.Background(), source, target)
	require.NoError(t, err)
	assert.Equal(t, "source", result.SourceRepo.Name)
	assert.Equal(t, "target", result.TargetRepo.Name)
}

func TestClient_GetPullRequestReviews(t *testing.T) {
	// Reviews are derived from the PR itself; this should call GetPullRequest internally.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    1,
			"title": "PR",
			"state": "OPEN",
			"author": map[string]any{"display_name": "Dev", "nickname": "dev", "uuid": "{1}"},
			"source":      map[string]any{"branch": map[string]any{"name": "feat"}},
			"destination": map[string]any{"branch": map[string]any{"name": "main"}},
			"created_on":  "2024-01-01T00:00:00Z",
			"updated_on":  "2024-01-02T00:00:00Z",
			"links": map[string]any{"html": map[string]any{"href": ""}},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	reviews, err := c.GetPullRequestReviews(context.Background(), testRepo(t), testPRNumber(t))
	require.NoError(t, err)
	// Bitbucket doesn't have a direct reviews endpoint; returns empty slice
	assert.NotNil(t, reviews)
}

func TestClient_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetPullRequest(context.Background(), testRepo(t), testPRNumber(t))
	require.Error(t, err)
}

func TestClient_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetPullRequest(context.Background(), testRepo(t), testPRNumber(t))
	require.Error(t, err)
}

func TestClient_InternalServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetPullRequest(context.Background(), testRepo(t), testPRNumber(t))
	require.Error(t, err)
}

func TestClient_GetPullRequest_MergedState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    2,
			"title": "Merged PR",
			"state": "MERGED",
			"author": map[string]any{"display_name": "Dev", "nickname": "dev", "uuid": "{1}"},
			"source":      map[string]any{"branch": map[string]any{"name": "feat"}},
			"destination": map[string]any{"branch": map[string]any{"name": "main"}},
			"created_on":  "2024-01-01T00:00:00Z",
			"updated_on":  "2024-01-05T00:00:00Z",
			"links":       map[string]any{"html": map[string]any{"href": ""}},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	pr, err := c.GetPullRequest(context.Background(), testRepo(t), testPRNumber(t))
	require.NoError(t, err)
	assert.Equal(t, "Merged PR", pr.Title)
}

func TestNewClient_AppPassword(t *testing.T) {
	c, err := bitbucket.NewClient(bitbucket.Config{
		BaseURL:     "http://localhost",
		Username:    "user",
		AppPassword: "pass",
	}, newTestLogger())
	require.NoError(t, err)
	assert.NotNil(t, c)
}

