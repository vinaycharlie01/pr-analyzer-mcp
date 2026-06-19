package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/entity"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/domain/valueobject"
	"github.com/vinaycharlie01/pr-analyzer-mcp/internal/port/outbound"
	apperrors "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/errors"
)

type Config struct {
	BaseURL       string
	DatacenterURL string
	Token         string
	Username      string
	AppPassword   string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(cfg Config, logger *slog.Logger) (*Client, error) {
	if cfg.Token == "" && cfg.AppPassword == "" {
		return nil, apperrors.New(apperrors.ErrCodeConfiguration, "Bitbucket token or app password is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.bitbucket.org/2.0"
	}
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}, nil
}

func (c *Client) doRequest(ctx context.Context, method, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "creating request", err)
	}

	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	} else if c.cfg.Username != "" && c.cfg.AppPassword != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.AppPassword)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "executing request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "reading response body", err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return body, nil
	case http.StatusNotFound:
		return nil, apperrors.New(apperrors.ErrCodeNotFound, fmt.Sprintf("resource not found: %s", url))
	case http.StatusUnauthorized:
		return nil, apperrors.New(apperrors.ErrCodeUnauthorized, "unauthorized: check Bitbucket credentials")
	case http.StatusTooManyRequests:
		return nil, apperrors.New(apperrors.ErrCodeRateLimit, "rate limit exceeded")
	default:
		return nil, apperrors.New(apperrors.ErrCodeInternal, fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)))
	}
}

func (c *Client) GetPullRequest(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) (*entity.PullRequest, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d",
		c.cfg.BaseURL, repo.Owner(), repo.Name(), number.Value())

	body, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	var bbPR bitbucketPR
	if err := json.Unmarshal(body, &bbPR); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "parsing pull request", err)
	}

	return mapBitbucketPR(&bbPR, repo), nil
}

func (c *Client) ListPullRequests(ctx context.Context, repo valueobject.RepositoryRef, opts outbound.ListPROptions) ([]*entity.PullRequest, error) {
	state := "OPEN"
	if opts.State != "" {
		state = opts.State
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests?state=%s&pagelen=%d&page=%d",
		c.cfg.BaseURL, repo.Owner(), repo.Name(), state, opts.PerPage, opts.Page)

	body, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	var response struct {
		Values []*bitbucketPR `json:"values"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "parsing pull requests", err)
	}

	result := make([]*entity.PullRequest, 0, len(response.Values))
	for _, pr := range response.Values {
		result = append(result, mapBitbucketPR(pr, repo))
	}
	return result, nil
}

func (c *Client) GetPullRequestFiles(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.ChangedFile, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/diffstat",
		c.cfg.BaseURL, repo.Owner(), repo.Name(), number.Value())

	body, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	var response struct {
		Values []struct {
			Status string `json:"status"`
			New    *struct {
				Path string `json:"path"`
			} `json:"new"`
			Old *struct {
				Path string `json:"path"`
			} `json:"old"`
			LinesAdded   int `json:"lines_added"`
			LinesRemoved int `json:"lines_removed"`
		} `json:"values"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "parsing diffstat", err)
	}

	result := make([]entity.ChangedFile, 0, len(response.Values))
	for _, f := range response.Values {
		cf := entity.ChangedFile{
			Additions: f.LinesAdded,
			Deletions: f.LinesRemoved,
		}
		if f.New != nil {
			cf.Path = f.New.Path
		}
		if f.Old != nil {
			cf.OldPath = f.Old.Path
		}
		switch f.Status {
		case "added":
			cf.Status = entity.FileStatusAdded
		case "removed":
			cf.Status = entity.FileStatusDeleted
		case "renamed":
			cf.Status = entity.FileStatusRenamed
		default:
			cf.Status = entity.FileStatusModified
		}
		result = append(result, cf)
	}
	return result, nil
}

func (c *Client) GetPullRequestCommits(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Commit, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/commits",
		c.cfg.BaseURL, repo.Owner(), repo.Name(), number.Value())

	body, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	var response struct {
		Values []struct {
			Hash    string `json:"hash"`
			Message string `json:"message"`
			Author  struct {
				User struct {
					DisplayName string `json:"display_name"`
					Nickname    string `json:"nickname"`
				} `json:"user"`
				Raw string `json:"raw"`
			} `json:"author"`
			Date  string `json:"date"`
			Links struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"values"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "parsing commits", err)
	}

	result := make([]entity.Commit, 0, len(response.Values))
	for _, c := range response.Values {
		t, _ := time.Parse(time.RFC3339, c.Date)
		result = append(result, entity.Commit{
			SHA:     c.Hash,
			Message: c.Message,
			Author: entity.Author{
				Name:     c.Author.User.DisplayName,
				Username: c.Author.User.Nickname,
			},
			Timestamp: t,
			URL:       c.Links.HTML.Href,
		})
	}
	return result, nil
}

func (c *Client) GetPullRequestReviews(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Review, error) {
	pr, err := c.GetPullRequest(ctx, repo, number)
	if err != nil {
		return nil, err
	}
	_ = pr
	return []entity.Review{}, nil
}

func (c *Client) GetPullRequestComments(ctx context.Context, repo valueobject.RepositoryRef, number valueobject.PRNumber) ([]entity.Comment, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments",
		c.cfg.BaseURL, repo.Owner(), repo.Name(), number.Value())

	body, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	var response struct {
		Values []struct {
			ID      int    `json:"id"`
			Content struct {
				Raw string `json:"raw"`
			} `json:"content"`
			Author struct {
				DisplayName string `json:"display_name"`
				Nickname    string `json:"nickname"`
			} `json:"author"`
			CreatedOn string `json:"created_on"`
			Inline    *struct {
				Path string `json:"path"`
				To   int    `json:"to"`
			} `json:"inline"`
		} `json:"values"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "parsing comments", err)
	}

	result := make([]entity.Comment, 0, len(response.Values))
	for _, cm := range response.Values {
		t, _ := time.Parse(time.RFC3339, cm.CreatedOn)
		comment := entity.Comment{
			ID:        fmt.Sprintf("%d", cm.ID),
			Body:      cm.Content.Raw,
			CreatedAt: t,
			Author: entity.Author{
				Name:     cm.Author.DisplayName,
				Username: cm.Author.Nickname,
			},
		}
		if cm.Inline != nil {
			comment.FilePath = cm.Inline.Path
			comment.Line = cm.Inline.To
		}
		result = append(result, comment)
	}
	return result, nil
}

func (c *Client) GetFileContent(ctx context.Context, repo valueobject.RepositoryRef, path, ref string) (string, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/src/%s/%s",
		c.cfg.BaseURL, repo.Owner(), repo.Name(), ref, path)

	body, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) ListBranches(ctx context.Context, repo valueobject.RepositoryRef) ([]string, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/refs/branches",
		c.cfg.BaseURL, repo.Owner(), repo.Name())

	body, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	var response struct {
		Values []struct {
			Name string `json:"name"`
		} `json:"values"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "parsing branches", err)
	}

	result := make([]string, 0, len(response.Values))
	for _, b := range response.Values {
		result = append(result, b.Name)
	}
	return result, nil
}

func (c *Client) CompareRepositories(ctx context.Context, source, target valueobject.RepositoryRef) (*outbound.RepositoryComparison, error) {
	return &outbound.RepositoryComparison{
		SourceRepo: entity.Repository{
			Name:     source.Name(),
			FullName: source.String(),
			Owner:    source.Owner(),
			Platform: entity.PlatformBitbucket,
		},
		TargetRepo: entity.Repository{
			Name:     target.Name(),
			FullName: target.String(),
			Owner:    target.Owner(),
			Platform: entity.PlatformBitbucket,
		},
	}, nil
}

// bitbucketPR is the raw Bitbucket API PR response structure.
type bitbucketPR struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Author      struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
		UUID        string `json:"uuid"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

func mapBitbucketPR(pr *bitbucketPR, repo valueobject.RepositoryRef) *entity.PullRequest {
	result := &entity.PullRequest{
		ID:           fmt.Sprintf("%d", pr.ID),
		Number:       pr.ID,
		Title:        pr.Title,
		Description:  pr.Description,
		Platform:     entity.PlatformBitbucket,
		SourceBranch: pr.Source.Branch.Name,
		TargetBranch: pr.Destination.Branch.Name,
		URL:          pr.Links.HTML.Href,
		Author: entity.Author{
			ID:       pr.Author.UUID,
			Username: pr.Author.Nickname,
			Name:     pr.Author.DisplayName,
		},
		Repository: entity.Repository{
			Name:     repo.Name(),
			FullName: repo.String(),
			Owner:    repo.Owner(),
			Platform: entity.PlatformBitbucket,
		},
	}

	switch pr.State {
	case "OPEN":
		result.Status = entity.PRStatusOpen
	case "MERGED":
		result.Status = entity.PRStatusMerged
	default:
		result.Status = entity.PRStatusClosed
	}

	result.CreatedAt, _ = time.Parse(time.RFC3339, pr.CreatedOn)
	result.UpdatedAt, _ = time.Parse(time.RFC3339, pr.UpdatedOn)

	return result
}
