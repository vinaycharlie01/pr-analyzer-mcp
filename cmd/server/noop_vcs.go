package main

import (
	"context"
	"fmt"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
	apperrors "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/errors"
)

// noopVCS wraps an optional VCS implementation, returning a useful error
// when the underlying client is nil (i.e., credentials not configured).
type noopVCS struct {
	delegate outbound.VCSPort
}

func (n noopVCS) GetPullRequest(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) (*entity.PullRequest, error) {
	if n.delegate == nil {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, fmt.Sprintf("VCS client not configured for %s", repo))
	}
	return n.delegate.GetPullRequest(ctx, repo, number)
}

func (n noopVCS) ListPullRequests(ctx context.Context, repo valueobject.RepositoryRef, opts outbound.ListPROptions) ([]*entity.PullRequest, error) {
	if n.delegate == nil {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "VCS client not configured")
	}
	return n.delegate.ListPullRequests(ctx, repo, opts)
}

func (n noopVCS) GetPullRequestFiles(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.ChangedFile, error) {
	if n.delegate == nil {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "VCS client not configured")
	}
	return n.delegate.GetPullRequestFiles(ctx, repo, number)
}

func (n noopVCS) GetPullRequestCommits(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Commit, error) {
	if n.delegate == nil {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "VCS client not configured")
	}
	return n.delegate.GetPullRequestCommits(ctx, repo, number)
}

func (n noopVCS) GetPullRequestReviews(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Review, error) {
	if n.delegate == nil {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "VCS client not configured")
	}
	return n.delegate.GetPullRequestReviews(ctx, repo, number)
}

func (n noopVCS) GetPullRequestComments(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Comment, error) {
	if n.delegate == nil {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "VCS client not configured")
	}
	return n.delegate.GetPullRequestComments(ctx, repo, number)
}

func (n noopVCS) GetFileContent(ctx context.Context, repo valueobject.RepositoryRef, path, ref string) (string, error) {
	if n.delegate == nil {
		return "", apperrors.New(apperrors.ErrCodeConfiguration, "VCS client not configured")
	}
	return n.delegate.GetFileContent(ctx, repo, path, ref)
}

func (n noopVCS) ListBranches(ctx context.Context, repo valueobject.RepositoryRef) ([]string, error) {
	if n.delegate == nil {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "VCS client not configured")
	}
	return n.delegate.ListBranches(ctx, repo)
}

func (n noopVCS) CompareRepositories(ctx context.Context, source, target valueobject.RepositoryRef) (*outbound.RepositoryComparison, error) {
	if n.delegate == nil {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "VCS client not configured")
	}
	return n.delegate.CompareRepositories(ctx, source, target)
}
