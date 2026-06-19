package outbound

import (
	"context"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
)

// VCSPort defines the outbound port for version control system operations.
type VCSPort interface {
	GetPullRequest(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) (*entity.PullRequest, error)
	ListPullRequests(ctx context.Context, repo valueobject.RepositoryRef, opts ListPROptions) ([]*entity.PullRequest, error)
	GetPullRequestFiles(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.ChangedFile, error)
	GetPullRequestCommits(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Commit, error)
	GetPullRequestReviews(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Review, error)
	GetPullRequestComments(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Comment, error)
	GetFileContent(ctx context.Context, repo valueobject.RepositoryRef, path, ref string) (string, error)
	ListBranches(ctx context.Context, repo valueobject.RepositoryRef) ([]string, error)
	CompareRepositories(ctx context.Context, source, target valueobject.RepositoryRef) (*RepositoryComparison, error)
}

type ListPROptions struct {
	State  string
	Page   int
	PerPage int
}

type RepositoryComparison struct {
	SourceRepo   entity.Repository
	TargetRepo   entity.Repository
	CommonFiles  []string
	SourceOnly   []string
	TargetOnly   []string
	Differences  []FileDiff
}

type FileDiff struct {
	Path    string
	Source  string
	Target  string
	Similar bool
}
