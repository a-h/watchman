package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/a-h/watchman/observer/data"
	"github.com/a-h/watchman/observer/github"
	"github.com/a-h/watchman/observer/logger"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const pkg = "github.com/a-h/watchman/observer/repo"

type IssueLister func(ctx context.Context, repository string) (issues []github.Issue, err error)

// Handler handles incoming Repository messages.
type Handler struct {
	ListIssues            IssueLister
	SendToIssueInputQueue func(iss github.Issue) error
}

// Handle incoming Repository messages.
func (h Handler) Handle(ctx context.Context, e events.SNSEvent) error {
	for _, r := range e.Records {
		var repo data.Repository
		err := json.Unmarshal([]byte(r.SNS.Message), &repo)
		if err != nil {
			return fmt.Errorf("repo: error unmarshalling SNS message: '%v'", r.SNS.Message)
		}
		err = h.handle(ctx, repo)
		if err != nil {
			return fmt.Errorf("repo: error handling SNS message for repo: '%v': %v", repo.URL, err)
		}
	}
	return nil
}

func (h Handler) handle(ctx context.Context, repo data.Repository) error {
	l := logger.For(pkg, "handle").WithField("repoUrl", repo.URL)
	issues, err := h.ListIssues(ctx, repo.URL)
	if err != nil {
		l.WithError(err).Error("failed to list issues")
		return fmt.Errorf("start: error listing issues for repo: %v", err)
	}
	for _, issue := range issues {
		ll := l.WithField("issueUrl", issue.URL)
		if issue.UpdatedAt.Before(repo.LastUpdated) {
			ll.Info("skipping issue, not updated recently")
			continue
		}
		err = h.SendToIssueInputQueue(issue)
		if err != nil {
			ll.WithError(err).Error("failed to send to Issue Input queue")
			return fmt.Errorf("start: error sending to issue input queue for repo %v: %v", repo.URL, err)
		}
	}
	return nil
}

func main() {
	//TODO: Populate handler.
	h := Handler{}
	lambda.Start(h.Handle)
}
