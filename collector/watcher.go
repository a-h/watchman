package collector

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/machinebox/graphql"

	"golang.org/x/oauth2"
)

// Collector uses the Github GraphQL API to collect data.
type Collector struct {
	githubToken string
	Log         func(s string)
}

// maximumPageSize of Github GraphQL requests.
const maximumPageSize = 100

// NewCollector creates a Github data collector with the githubToken used to authenticate API calls.
// See https://developer.github.com/v4/guides/forming-calls/#authenticating-with-graphql
func NewCollector(githubToken string) *Collector {
	return &Collector{
		githubToken: githubToken,
	}
}

func (c *Collector) client(ctx context.Context) *graphql.Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.githubToken},
	)
	httpClient := oauth2.NewClient(ctx, src)
	client := graphql.NewClient("https://api.github.com/graphql",
		graphql.WithHTTPClient(httpClient))
	if c.Log != nil {
		client.Log = c.Log
	}
	return client
}

// Issue within Github.
type Issue struct {
	Owner     string    `json:"owner"`
	Repo      string    `json:"repo"`
	URL       string    `json:"url"`
	Number    int       `json:"number"`
	BodyText  string    `json:"bodyText"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewIssue creates an issue with all fields populated.
func NewIssue(owner, repo, url string, number int, bodyText string, updatedAt time.Time) Issue {
	return Issue{
		Owner:     owner,
		Repo:      repo,
		URL:       url,
		Number:    number,
		BodyText:  bodyText,
		UpdatedAt: updatedAt,
	}
}

// Issues returns all issues for a given repo, by calling the same GraphQL query in a loop for each of the
// issues.
func (c Collector) Issues(ctx context.Context, repository string) (issues []Issue, err error) {
	owner, repo, err := unpackURL(repository)
	if err != nil {
		return
	}

	var cursor *string
	for {
		res, issErr := c.issuesPage(ctx, owner, repo, maximumPageSize, cursor)
		if issErr != nil {
			err = fmt.Errorf("collector: failed to get issues for repo '%s': %v", repository, issErr)
			return
		}
		for _, n := range res.Repository.Issues.Nodes {
			issues = append(issues, NewIssue(owner, repo, n.URL, n.Number, n.BodyText, n.UpdatedAt))
		}
		cursor = &res.Repository.Issues.PageInfo.EndCursor
		if !res.Repository.Issues.PageInfo.HasNextPage {
			return
		}
	}
}

func (c Collector) issuesPage(ctx context.Context, owner, repo string, first int, cursor *string) (result issuesQueryResult, err error) {
	req := graphql.NewRequest(issuesQuery)

	req.Var("owner", owner)
	req.Var("repo", repo)
	req.Var("first", first)
	req.Var("cursor", cursor)

	req.Header.Set("Cache-Control", "no-cache")

	err = c.client(ctx).Run(ctx, req, &result)
	return
}

const issuesQuery = `query($owner:String!, $repo:String!, $first:Int!, $cursor:String) {
  repository(owner: $owner, name: $repo) {
    issues(first: $first, after: $cursor) {
      pageInfo {
        endCursor
        hasNextPage
      }
      nodes {
				url
				number
        bodyText
        updatedAt
      }
    }
  }
}`

type issuesQueryResult struct {
	Repository struct {
		Issues struct {
			PageInfo struct {
				EndCursor   string `json:"endCursor"`
				HasNextPage bool   `json:"hasNextPage"`
			} `json:"pageInfo"`
			Nodes []struct {
				URL       string    `json:"url"`
				Number    int       `json:"number"`
				BodyText  string    `json:"bodyText"`
				UpdatedAt time.Time `json:"updatedAt"`
			} `json:"nodes"`
		} `json:"issues"`
	} `json:"repository"`
}

// A Comment on a Github issue.
type Comment struct {
	Owner       string    `json:"owner"`
	Repo        string    `json:"repo"`
	IssueNumber int       `json:"issueNumber"`
	URL         string    `json:"url"`
	BodyText    string    `json:"bodyText"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// NewComment creates a comment with all required fields populated.
func NewComment(owner, repo string, issueNumber int, url string, bodyText string, updatedAt time.Time) Comment {
	return Comment{
		Owner:       owner,
		Repo:        repo,
		IssueNumber: issueNumber,
		URL:         url,
		BodyText:    bodyText,
		UpdatedAt:   updatedAt,
	}
}

// Comments retrieves all of the comments for a particular issue.
func (c Collector) Comments(ctx context.Context, owner, repo string, issueNumber int) (comments []Comment, err error) {
	var cursor *string
	for {
		res, comErr := c.commentsPage(ctx, owner, repo, issueNumber, maximumPageSize, cursor)
		if comErr != nil {
			err = fmt.Errorf("collector: failed to get comments for repo '%s/%s/issues/%d': %v", owner, repo, issueNumber, comErr)
			return
		}
		for _, n := range res.Repository.Issue.Comments.Nodes {
			comments = append(comments, NewComment(owner, repo, issueNumber, n.URL, n.BodyText, n.UpdatedAt))
		}
		cursor = &res.Repository.Issue.Comments.PageInfo.EndCursor
		if !res.Repository.Issue.Comments.PageInfo.HasNextPage {
			return
		}
	}
}

func (c Collector) commentsPage(ctx context.Context, owner, repo string, issueNumber int, first int, cursor *string) (result commentsQueryResult, err error) {
	req := graphql.NewRequest(commentsQuery)

	req.Var("owner", owner)
	req.Var("repo", repo)
	req.Var("issueNumber", issueNumber)
	req.Var("first", first)
	req.Var("cursor", cursor)

	req.Header.Set("Cache-Control", "no-cache")

	err = c.client(ctx).Run(ctx, req, &result)
	return
}

const commentsQuery = `query ($owner: String!, $repo: String!, $issueNumber: Int!, $first: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    issue(number: $issueNumber) {
      comments(first: $first, after: $cursor) {
        pageInfo {
          endCursor
          hasNextPage
        }
        nodes {
          url
          updatedAt
          bodyText
        }
      }
    }
  }
}`

type commentsQueryResult struct {
	Repository struct {
		Issue struct {
			Comments struct {
				PageInfo struct {
					EndCursor   string `json:"endCursor"`
					HasNextPage bool   `json:"hasNextPage"`
				} `json:"pageInfo"`
				Nodes []struct {
					URL       string    `json:"url"`
					UpdatedAt time.Time `json:"updatedAt"`
					BodyText  string    `json:"bodyText"`
				} `json:"nodes"`
			} `json:"comments"`
		} `json:"issue"`
	} `json:"repository"`
}

// ErrInvalidGithubRepoURL is the error returned when a given URL is not of the form:
// github.com/%s/%s where %s/%s is the owner and repo.
var ErrInvalidGithubRepoURL = errors.New("Invalid Github repo URL")

// ErrInvalidGithubRepoHostname is the error returned if the URL passed for the repo isn't
// github.com
var ErrInvalidGithubRepoHostname = errors.New("Invalid Github repo hostname")

func unpackURL(u string) (owner, repo string, err error) {
	repoURL, err := url.Parse(u)
	if err != nil {
		return
	}
	if !strings.EqualFold(repoURL.Hostname(), "github.com") {
		err = ErrInvalidGithubRepoHostname
		return
	}
	p := strings.Split(strings.Trim(repoURL.Path, "/"), "/")
	if len(p) != 2 {
		fmt.Println(p, len(p))
		err = ErrInvalidGithubRepoURL
		return
	}
	owner = p[0]
	repo = p[1]
	return
}
