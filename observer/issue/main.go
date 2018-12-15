package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/a-h/watchman/observer/data"
	"github.com/a-h/watchman/observer/dynamo"
	"github.com/a-h/watchman/observer/github"
	"github.com/a-h/watchman/observer/logger"
	"github.com/a-h/watchman/observer/notify"
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
	l.WithField("commentCount", len(comments)).Info("found comments")
	// Check the issue text.
	if riss.Issue.UpdatedAt.After(riss.Repository.LastUpdated) {
		if containsSecurityKeywords(riss.Issue.BodyText) {
			l.Info("notifying based on issue body text")
			err = h.Notify(riss, github.Comment{})
			if err != nil {
				return fmt.Errorf("issue: error notifying based on issue body text for issue %v: %v", riss.Issue.URL, err)
			}
		}
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
			continue
		}
		if !containsSecurityKeywords(comment.BodyText) {
			ll.Info("no security content found")
		}
		ll.Info("notifying")
		err = h.Notify(riss, comment)
		if err != nil {
			ll.WithError(err).Error("error sending notification")
			return fmt.Errorf("issue: error sending notification for comment %v: %v", comment.URL, err)
		}
	}
	return nil
}

var securityWords = []string{"security", "exploit", "vulnerable", "exploited", "insecure", "xss"}

func containsSecurityKeywords(text string) bool {
	scanner := bufio.NewScanner(bytes.NewBufferString(text))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		for _, securityWord := range securityWords {
			if strings.EqualFold(securityWord, scanner.Text()) {
				return true
			}
		}
	}
	return false
}

func main() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	collector := github.NewCollector(githubToken)

	commentTableName := os.Getenv("COMMENT_TABLE_NAME")
	awsRegion := os.Getenv("AWS_REGION")
	store, err := dynamo.NewCommentStore(awsRegion, commentTableName)
	if err != nil {
		logger.For(pkg, "main").WithError(err).Fatal("failed to create comment store")
	}
	alertTopic := os.Getenv("ALERT_SNS_TOPIC_ARN")
	n := notify.NewSNS(alertTopic)
	h := Handler{
		ListComments: collector.Comments,
		Notify:       n.Notify,
		MarkNotified: func(comment data.Comment) (exists bool, err error) {
			return store.Put(data.Comment{})
		},
	}
	lambda.Start(h.Handle)
}
