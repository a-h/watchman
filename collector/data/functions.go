package data

import "time"

// A RepositoryPutter puts a Repository into a database.
type RepositoryPutter func(r Repository) error

// A Repository within Github.
type Repository struct {
	URL string `json:"url"`

	// UsedBy is the URL of a repo that uses the repository.
	UsedByURL string `json:"usedByUrl"`
	// LastChecked was the time that the repo was last checked.
	LastChecked time.Time `json:"lastChecked"`
}

// IssuePutter upserts an issue.
type IssuePutter func(issue Issue) (updated bool, err error)

// A Issue is an issue within a repo.
type Issue struct {
	URL string `json:"url"`

	RepositoryURL string `json:"repositoryUrl"`
	// LastUpdated is the time that the issue was last updated.
	// Used to determine whether we need to read the issue's comments, or we've already done it.
	LastUpdated time.Time `json:"lastUpdated"`
	// LastChecked was the time that the issue was last checked for updates.
	LastChecked time.Time `json:"lastChecked"`
}

// A CommentPutter upserts a Comment.
type CommentPutter func(comment Comment) (updated bool, err error)

// A Comment on a specific issue. If it's present, then it's been processed
// and a notification doesn't need to be sent.
type Comment struct {
	URL      string `json:"url"`
	TextHash string `json:"textHash"`

	IssueURL      string `json:"issueUrl"`
	RepositoryURL string `json:"repositoryUrl"`
	// CheckedAt is the time that the issue was checked.
	CheckedAt time.Time `json:"checkedAt"`
}
