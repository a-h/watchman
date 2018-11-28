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
	return graphql.NewClient("https://api.github.com/graphql",
		graphql.WithHTTPClient(httpClient))
}

// Issue within Github.
type Issue struct {
	URL       string    `json:"url"`
	BodyText  string    `json:"bodyText"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewIssue creates an issue with all fields populated.
func NewIssue(url, bodyText string, updatedAt time.Time) Issue {
	return Issue{
		URL:       url,
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
			issues = append(issues, NewIssue(n.URL, n.BodyText, n.UpdatedAt))
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
	req.Var("issueCursor", cursor)

	req.Header.Set("Cache-Control", "no-cache")

	err = c.client(ctx).Run(ctx, req, &result)
	return
}

const issuesQuery = `query($owner:String!, $repo:String!, $first:Int!, $issueCursor:String) {
  repository(owner: $owner, name: $repo) {
    issues(first: $first, after: $issueCursor) {
      pageInfo {
        endCursor
        hasNextPage
      }
      nodes {
        url
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
				BodyText  string    `json:"bodyText"`
				UpdatedAt time.Time `json:"updatedAt"`
			} `json:"nodes"`
		} `json:"issues"`
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
