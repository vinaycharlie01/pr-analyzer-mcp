package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	gogithub "github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
	apperrors "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/errors"
)

type Config struct {
	Token   string
	BaseURL string
}

type Client struct {
	client *gogithub.Client
	logger *slog.Logger
}

func NewClient(cfg Config, logger *slog.Logger) (*Client, error) {
	if cfg.Token == "" {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "GitHub token is required")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Token})
	tc := oauth2.NewClient(context.Background(), ts)
	ghClient := gogithub.NewClient(tc)

	if cfg.BaseURL != "" && cfg.BaseURL != "https://api.github.com" {
		var err error
		ghClient, err = ghClient.WithAuthToken(cfg.Token).WithEnterpriseURLs(cfg.BaseURL, cfg.BaseURL)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeConfiguration, "setting GitHub enterprise URL", err)
		}
	}

	return &Client{client: ghClient, logger: logger}, nil
}

func (c *Client) GetPullRequest(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) (*entity.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, repo.Owner(), repo.Name(), number.Value())
	if err != nil {
		return nil, wrapGitHubError(err, "getting pull request")
	}
	return mapGitHubPR(pr, repo), nil
}

func (c *Client) ListPullRequests(ctx context.Context, repo valueobject.RepositoryRef, opts outbound.ListPROptions) ([]*entity.PullRequest, error) {
	ghOpts := &gogithub.PullRequestListOptions{
		State: opts.State,
		ListOptions: gogithub.ListOptions{
			Page:    opts.Page,
			PerPage: opts.PerPage,
		},
	}
	if ghOpts.State == "" {
		ghOpts.State = "open"
	}

	prs, _, err := c.client.PullRequests.List(ctx, repo.Owner(), repo.Name(), ghOpts)
	if err != nil {
		return nil, wrapGitHubError(err, "listing pull requests")
	}

	result := make([]*entity.PullRequest, 0, len(prs))
	for _, pr := range prs {
		result = append(result, mapGitHubPR(pr, repo))
	}
	return result, nil
}

func (c *Client) GetPullRequestFiles(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.ChangedFile, error) {
	files, _, err := c.client.PullRequests.ListFiles(ctx, repo.Owner(), repo.Name(), number.Value(), nil)
	if err != nil {
		return nil, wrapGitHubError(err, "listing PR files")
	}

	result := make([]entity.ChangedFile, 0, len(files))
	for _, f := range files {
		result = append(result, entity.ChangedFile{
			Path:      derefStr(f.Filename),
			OldPath:   derefStr(f.PreviousFilename),
			Status:    mapGitHubFileStatus(derefStr(f.Status)),
			Additions: derefInt(f.Additions),
			Deletions: derefInt(f.Deletions),
			Patch:     derefStr(f.Patch),
		})
	}
	return result, nil
}

func (c *Client) GetPullRequestCommits(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Commit, error) {
	commits, _, err := c.client.PullRequests.ListCommits(ctx, repo.Owner(), repo.Name(), number.Value(), nil)
	if err != nil {
		return nil, wrapGitHubError(err, "listing PR commits")
	}

	result := make([]entity.Commit, 0, len(commits))
	for _, c := range commits {
		commit := entity.Commit{
			SHA:     derefStr(c.SHA),
			URL:     derefStr(c.URL),
		}
		if c.Commit != nil {
			commit.Message = derefStr(c.Commit.Message)
			if c.Commit.Author != nil {
				commit.Timestamp = derefTime(c.Commit.Author.Date)
				commit.Author = entity.Author{
					Name:  derefStr(c.Commit.Author.Name),
					Email: derefStr(c.Commit.Author.Email),
				}
			}
		}
		result = append(result, commit)
	}
	return result, nil
}

func (c *Client) GetPullRequestReviews(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Review, error) {
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, repo.Owner(), repo.Name(), number.Value(), nil)
	if err != nil {
		return nil, wrapGitHubError(err, "listing PR reviews")
	}

	result := make([]entity.Review, 0, len(reviews))
	for _, r := range reviews {
		review := entity.Review{
			ID:   fmt.Sprintf("%d", derefInt64(r.ID)),
			Body: derefStr(r.Body),
		}
		if r.User != nil {
			review.Author = entity.Author{
				ID:       fmt.Sprintf("%d", derefInt64(r.User.ID)),
				Username: derefStr(r.User.Login),
			}
		}
		if r.SubmittedAt != nil {
			review.CreatedAt = r.SubmittedAt.Time
		}
		switch derefStr(r.State) {
		case "APPROVED":
			review.State = entity.ReviewStateApproved
		case "CHANGES_REQUESTED":
			review.State = entity.ReviewStateRequestedChanges
		default:
			review.State = entity.ReviewStateCommented
		}
		result = append(result, review)
	}
	return result, nil
}

func (c *Client) GetPullRequestComments(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Comment, error) {
	comments, _, err := c.client.Issues.ListComments(ctx, repo.Owner(), repo.Name(), number.Value(), nil)
	if err != nil {
		return nil, wrapGitHubError(err, "listing PR comments")
	}

	result := make([]entity.Comment, 0, len(comments))
	for _, cm := range comments {
		comment := entity.Comment{
			ID:   fmt.Sprintf("%d", derefInt64(cm.ID)),
			Body: derefStr(cm.Body),
		}
		if cm.User != nil {
			comment.Author = entity.Author{
				ID:       fmt.Sprintf("%d", derefInt64(cm.User.ID)),
				Username: derefStr(cm.User.Login),
			}
		}
		if cm.CreatedAt != nil {
			comment.CreatedAt = cm.CreatedAt.Time
		}
		result = append(result, comment)
	}
	return result, nil
}

func (c *Client) GetFileContent(ctx context.Context, repo valueobject.RepositoryRef, path, ref string) (string, error) {
	opts := &gogithub.RepositoryContentGetOptions{Ref: ref}
	content, _, _, err := c.client.Repositories.GetContents(ctx, repo.Owner(), repo.Name(), path, opts)
	if err != nil {
		return "", wrapGitHubError(err, "getting file content")
	}
	if content == nil {
		return "", apperrors.New(apperrors.ErrCodeNotFound, "file content not found")
	}
	decoded, err := content.GetContent()
	if err != nil {
		return "", apperrors.Wrap(apperrors.ErrCodeInternal, "decoding file content", err)
	}
	return decoded, nil
}

func (c *Client) ListBranches(ctx context.Context, repo valueobject.RepositoryRef) ([]string, error) {
	branches, _, err := c.client.Repositories.ListBranches(ctx, repo.Owner(), repo.Name(), nil)
	if err != nil {
		return nil, wrapGitHubError(err, "listing branches")
	}

	result := make([]string, 0, len(branches))
	for _, b := range branches {
		result = append(result, derefStr(b.Name))
	}
	return result, nil
}

func (c *Client) CompareRepositories(ctx context.Context, source, target valueobject.RepositoryRef) (*outbound.RepositoryComparison, error) {
	sourceRepo, _, err := c.client.Repositories.Get(ctx, source.Owner(), source.Name())
	if err != nil {
		return nil, wrapGitHubError(err, "getting source repository")
	}
	targetRepo, _, err := c.client.Repositories.Get(ctx, target.Owner(), target.Name())
	if err != nil {
		return nil, wrapGitHubError(err, "getting target repository")
	}

	return &outbound.RepositoryComparison{
		SourceRepo: entity.Repository{
			Name:     derefStr(sourceRepo.Name),
			FullName: derefStr(sourceRepo.FullName),
			Owner:    source.Owner(),
			Platform: entity.PlatformGitHub,
			URL:      derefStr(sourceRepo.HTMLURL),
			Language: derefStr(sourceRepo.Language),
		},
		TargetRepo: entity.Repository{
			Name:     derefStr(targetRepo.Name),
			FullName: derefStr(targetRepo.FullName),
			Owner:    target.Owner(),
			Platform: entity.PlatformGitHub,
			URL:      derefStr(targetRepo.HTMLURL),
			Language: derefStr(targetRepo.Language),
		},
	}, nil
}

func mapGitHubPR(pr *gogithub.PullRequest, repo valueobject.RepositoryRef) *entity.PullRequest {
	result := &entity.PullRequest{
		ID:          fmt.Sprintf("%d", derefInt64(pr.ID)),
		Number:      derefInt(pr.Number),
		Title:       derefStr(pr.Title),
		Description: derefStr(pr.Body),
		Platform:    entity.PlatformGitHub,
		URL:         derefStr(pr.HTMLURL),
		Repository: entity.Repository{
			Name:     repo.Name(),
			FullName: repo.String(),
			Owner:    repo.Owner(),
			Platform: entity.PlatformGitHub,
		},
	}

	if pr.User != nil {
		result.Author = entity.Author{
			ID:       fmt.Sprintf("%d", derefInt64(pr.User.ID)),
			Username: derefStr(pr.User.Login),
			Name:     derefStr(pr.User.Name),
		}
	}

	if pr.Head != nil {
		result.SourceBranch = derefStr(pr.Head.Ref)
	}
	if pr.Base != nil {
		result.TargetBranch = derefStr(pr.Base.Ref)
	}

	switch derefStr(pr.State) {
	case "open":
		result.Status = entity.PRStatusOpen
	case "closed":
		if pr.MergedAt != nil {
			result.Status = entity.PRStatusMerged
			t := pr.MergedAt.Time
			result.MergedAt = &t
		} else {
			result.Status = entity.PRStatusClosed
		}
	}

	if pr.CreatedAt != nil {
		result.CreatedAt = pr.CreatedAt.Time
	}
	if pr.UpdatedAt != nil {
		result.UpdatedAt = pr.UpdatedAt.Time
	}

	for _, label := range pr.Labels {
		result.Labels = append(result.Labels, derefStr(label.Name))
	}

	return result
}

func mapGitHubFileStatus(status string) entity.FileStatus {
	switch status {
	case "added":
		return entity.FileStatusAdded
	case "removed":
		return entity.FileStatusDeleted
	case "renamed":
		return entity.FileStatusRenamed
	default:
		return entity.FileStatusModified
	}
}

func wrapGitHubError(err error, msg string) error {
	var errResp *gogithub.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil && errResp.Response.StatusCode == 404 {
		return apperrors.Wrap(apperrors.ErrCodeNotFound, msg, err)
	}
	return apperrors.Wrap(apperrors.ErrCodeInternal, msg, err)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func derefTime(t *gogithub.Timestamp) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.Time
}
