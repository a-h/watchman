package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/welldigital/pusher/logger"

	"github.com/a-h/watchman/observer/data"
	"github.com/a-h/watchman/observer/github"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const pkg = "github.com/a-h/watchman/observer/issue"

// A CommentLister lists all comments for a given repository.
type CommentLister func(ctx context.Context, owner, repo string, issueNumber int) (comments []github.Comment, err error)

// Handler handles incoming Repository messages.
type Handler struct {
	ListComments CommentLister
	Notify       func(riss data.RepositoryIssue, comment github.Comment) error
	MarkNotified data.CommentPutter
}

// Handle incoming Repository messages.
func (h Handler) Handle(ctx context.Context, e events.SNSEvent) error {
	for _, r := range e.Records {
		var riss data.RepositoryIssue
		err := json.Unmarshal([]byte(r.SNS.Message), &riss)
		if err != nil {
			return fmt.Errorf("issue: error unmarshalling SNS message: '%v'", r.SNS.Message)
		}
		err = h.handle(ctx, riss)
		if err != nil {
			return fmt.Errorf("issue: error handling SNS message for issue: '%v': %v", riss.Issue.URL, err)
		}
	}
	return nil
}

func (h Handler) handle(ctx context.Context, riss data.RepositoryIssue) error {
	l := logger.For(pkg, "handle").WithField("issueUrl", riss.Issue.URL)
	comments, err := h.ListComments(ctx, riss.Issue.Owner, riss.Issue.Repo, riss.Issue.Number)
	if err != nil {
		l.WithError(err).Error("error listing comments")
		return fmt.Errorf("issue: error listing comments for repo: %v", err)
	}
	for _, comment := range comments {
		ll := l.WithField("commentUrl", comment.URL)
		if comment.UpdatedAt.Before(riss.Repository.LastUpdated) {
			ll.Info("previously checked")
			continue
		}
		exists, err := h.MarkNotified(data.Comment{
			URL:       comment.URL,
			Hash:      comment.Hash(),
			HandledAt: time.Now().UTC(),
		})
		if err != nil {
			return fmt.Errorf("issue: error marking comment %v as notified: %v", comment.URL, err)
		}
		if exists {
			ll.Info("already processed")
		}
		//TODO: Check the content for security-related keywords.
		ll.Info("notifying")
		err = h.Notify(riss, comment)
		if err != nil {
			ll.WithError(err).Error("error sending notification")
			return fmt.Errorf("issue: error sending notification for comment %v: %v", comment.URL, err)
		}
	}
	return nil
}

func main() {
	//TODO: Populate handler.
	h := Handler{}
	lambda.Start(h.Handle)
}
