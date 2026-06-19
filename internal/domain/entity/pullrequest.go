package entity

import (
	"time"
)

type PlatformType string

const (
	PlatformGitHub    PlatformType = "github"
	PlatformBitbucket PlatformType = "bitbucket"
)

type PRStatus string

const (
	PRStatusOpen   PRStatus = "open"
	PRStatusClosed PRStatus = "closed"
	PRStatusMerged PRStatus = "merged"
)

type PullRequest struct {
	ID          string
	Number      int
	Title       string
	Description string
	Author      Author
	Status      PRStatus
	Platform    PlatformType
	Repository  Repository
	SourceBranch string
	TargetBranch string
	Files        []ChangedFile
	Commits      []Commit
	Reviews      []Review
	Comments     []Comment
	Labels       []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MergedAt     *time.Time
	URL          string
}

type Author struct {
	ID       string
	Username string
	Email    string
	Name     string
}

type Repository struct {
	ID       string
	Name     string
	FullName string
	Owner    string
	Platform PlatformType
	URL      string
	Language string
}

type ChangedFile struct {
	Path      string
	OldPath   string
	Status    FileStatus
	Additions int
	Deletions int
	Patch     string
}

type FileStatus string

const (
	FileStatusAdded    FileStatus = "added"
	FileStatusModified FileStatus = "modified"
	FileStatusDeleted  FileStatus = "deleted"
	FileStatusRenamed  FileStatus = "renamed"
)

type Commit struct {
	SHA       string
	Message   string
	Author    Author
	Timestamp time.Time
	URL       string
}

type Review struct {
	ID        string
	Author    Author
	State     ReviewState
	Body      string
	CreatedAt time.Time
}

type ReviewState string

const (
	ReviewStateApproved         ReviewState = "approved"
	ReviewStateRequestedChanges ReviewState = "changes_requested"
	ReviewStateCommented        ReviewState = "commented"
)

type Comment struct {
	ID        string
	Author    Author
	Body      string
	FilePath  string
	Line      int
	CreatedAt time.Time
}
