package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
	apperrors "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/errors"
)

func testRef(t *testing.T) valueobject.RepositoryRef {
	t.Helper()
	ref, err := valueobject.ParseRepositoryRef("myorg/myrepo")
	require.NoError(t, err)
	return ref
}

func testNum(t *testing.T) valueobject.PRNumber {
	t.Helper()
	n, err := valueobject.NewPRNumber(1)
	require.NoError(t, err)
	return n
}

func TestNoopVCS_NilDelegate_GetPullRequest(t *testing.T) {
	v := noopVCS{delegate: nil}
	_, err := v.GetPullRequest(context.Background(), testRef(t), testNum(t))
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

func TestNoopVCS_NilDelegate_ListPullRequests(t *testing.T) {
	v := noopVCS{delegate: nil}
	_, err := v.ListPullRequests(context.Background(), testRef(t), outbound.ListPROptions{})
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

func TestNoopVCS_NilDelegate_GetPullRequestFiles(t *testing.T) {
	v := noopVCS{delegate: nil}
	_, err := v.GetPullRequestFiles(context.Background(), testRef(t), testNum(t))
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

func TestNoopVCS_NilDelegate_GetPullRequestCommits(t *testing.T) {
	v := noopVCS{delegate: nil}
	_, err := v.GetPullRequestCommits(context.Background(), testRef(t), testNum(t))
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

func TestNoopVCS_NilDelegate_GetPullRequestReviews(t *testing.T) {
	v := noopVCS{delegate: nil}
	_, err := v.GetPullRequestReviews(context.Background(), testRef(t), testNum(t))
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

func TestNoopVCS_NilDelegate_GetPullRequestComments(t *testing.T) {
	v := noopVCS{delegate: nil}
	_, err := v.GetPullRequestComments(context.Background(), testRef(t), testNum(t))
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

func TestNoopVCS_NilDelegate_GetFileContent(t *testing.T) {
	v := noopVCS{delegate: nil}
	_, err := v.GetFileContent(context.Background(), testRef(t), "main.go", "main")
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

func TestNoopVCS_NilDelegate_ListBranches(t *testing.T) {
	v := noopVCS{delegate: nil}
	_, err := v.ListBranches(context.Background(), testRef(t))
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

func TestNoopVCS_NilDelegate_CompareRepositories(t *testing.T) {
	v := noopVCS{delegate: nil}
	src, _ := valueobject.ParseRepositoryRef("org/src")
	dst, _ := valueobject.ParseRepositoryRef("org/dst")
	_, err := v.CompareRepositories(context.Background(), src, dst)
	require.Error(t, err)
	assert.True(t, apperrors.IsConfiguration(err))
}

// stubVCS is a minimal delegate for testing the pass-through path.
type stubVCS struct{}

func (s *stubVCS) GetPullRequest(_ context.Context, _ valueobject.RepositoryRef, _ valueobject.PRNumber) (*entity.PullRequest, error) {
	return &entity.PullRequest{Title: "stub"}, nil
}
func (s *stubVCS) ListPullRequests(_ context.Context, _ valueobject.RepositoryRef, _ outbound.ListPROptions) ([]*entity.PullRequest, error) {
	return []*entity.PullRequest{{Title: "stub"}}, nil
}
func (s *stubVCS) GetPullRequestFiles(_ context.Context, _ valueobject.RepositoryRef, _ valueobject.PRNumber) ([]entity.ChangedFile, error) {
	return []entity.ChangedFile{{Path: "file.go"}}, nil
}
func (s *stubVCS) GetPullRequestCommits(_ context.Context, _ valueobject.RepositoryRef, _ valueobject.PRNumber) ([]entity.Commit, error) {
	return []entity.Commit{{SHA: "abc"}}, nil
}
func (s *stubVCS) GetPullRequestReviews(_ context.Context, _ valueobject.RepositoryRef, _ valueobject.PRNumber) ([]entity.Review, error) {
	return []entity.Review{}, nil
}
func (s *stubVCS) GetPullRequestComments(_ context.Context, _ valueobject.RepositoryRef, _ valueobject.PRNumber) ([]entity.Comment, error) {
	return []entity.Comment{}, nil
}
func (s *stubVCS) GetFileContent(_ context.Context, _ valueobject.RepositoryRef, _, _ string) (string, error) {
	return "content", nil
}
func (s *stubVCS) ListBranches(_ context.Context, _ valueobject.RepositoryRef) ([]string, error) {
	return []string{"main"}, nil
}
func (s *stubVCS) CompareRepositories(_ context.Context, _, _ valueobject.RepositoryRef) (*outbound.RepositoryComparison, error) {
	return &outbound.RepositoryComparison{}, nil
}

func TestNoopVCS_WithDelegate_GetPullRequest(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	pr, err := v.GetPullRequest(context.Background(), testRef(t), testNum(t))
	require.NoError(t, err)
	assert.Equal(t, "stub", pr.Title)
}

func TestNoopVCS_WithDelegate_ListPullRequests(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	prs, err := v.ListPullRequests(context.Background(), testRef(t), outbound.ListPROptions{})
	require.NoError(t, err)
	assert.Len(t, prs, 1)
}

func TestNoopVCS_WithDelegate_GetPullRequestFiles(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	files, err := v.GetPullRequestFiles(context.Background(), testRef(t), testNum(t))
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestNoopVCS_WithDelegate_GetPullRequestCommits(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	commits, err := v.GetPullRequestCommits(context.Background(), testRef(t), testNum(t))
	require.NoError(t, err)
	assert.Len(t, commits, 1)
}

func TestNoopVCS_WithDelegate_GetPullRequestReviews(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	reviews, err := v.GetPullRequestReviews(context.Background(), testRef(t), testNum(t))
	require.NoError(t, err)
	assert.NotNil(t, reviews)
}

func TestNoopVCS_WithDelegate_GetPullRequestComments(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	comments, err := v.GetPullRequestComments(context.Background(), testRef(t), testNum(t))
	require.NoError(t, err)
	assert.NotNil(t, comments)
}

func TestNoopVCS_WithDelegate_GetFileContent(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	content, err := v.GetFileContent(context.Background(), testRef(t), "main.go", "main")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestNoopVCS_WithDelegate_ListBranches(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	branches, err := v.ListBranches(context.Background(), testRef(t))
	require.NoError(t, err)
	assert.Equal(t, []string{"main"}, branches)
}

func TestNoopVCS_WithDelegate_CompareRepositories(t *testing.T) {
	v := noopVCS{delegate: &stubVCS{}}
	src, _ := valueobject.ParseRepositoryRef("org/src")
	dst, _ := valueobject.ParseRepositoryRef("org/dst")
	result, err := v.CompareRepositories(context.Background(), src, dst)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
