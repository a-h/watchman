package data

import (
	"time"

	"github.com/a-h/watchman/observer/github"
)

// A RepositoryLister lists all of the repos.
type RepositoryLister func() (repos []Repository, err error)

// A RepositoryAdder adds a repository to the list of repos to check.
type RepositoryAdder func(repoURL string, usedByURL string) error

// A RepositoryUpdater updates the lastUpdated field of a repo.
type RepositoryUpdater func(repoURL string) error

// A Repository within Github.
type Repository struct {
	URL string `json:"url"`
	// UsedByUrls is list of URLs that uses the repository.
	UsedByURLs []string `json:"usedByUrls"`
	// LastUpdated is the time that this repo's issues were last checked.
	// We don't need to re-check any issues which haven't been updated since this this time.
	LastUpdated time.Time `json:"lastUpdated"`
}

// RepositoryIssue is a struct containing a repo and an issue.
type RepositoryIssue struct {
	Repository Repository   `json:"repository"`
	Issue      github.Issue `json:"issue"`
}

// A CommentPutter upserts a Comment.
type CommentPutter func(comment Comment) (exists bool, err error)

// A Comment on a specific issue. If it's present, then it's been processed
// and a notification doesn't need to be sent.
type Comment struct {
	URL       string    `json:"url"`
	Hash      string    `json:"hash"`
	HandledAt time.Time `json:"handledAt"`
}
