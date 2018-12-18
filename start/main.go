package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/a-h/watchman/observer/sender"

	"github.com/a-h/watchman/observer/data"
	"github.com/a-h/watchman/observer/dynamo"
	"github.com/a-h/watchman/observer/logger"
	"github.com/aws/aws-lambda-go/lambda"
)

const pkg = "github.com/a-h/watchman/observer/start"

// Handler handles incoming CloudWatch Timer elapsed events.
type Handler struct {
	ListRepositories      data.RepositoryLister
	SendToRepoInputQueue  func(r data.Repository) error
	UpdateLastUpdatedDate data.RepositoryUpdater
}

// Handle incoming CloudWatch Timer elapsed events.
func (h Handler) Handle(ctx context.Context) error {
	l := logger.For(pkg, "handle")
	repos, err := h.ListRepositories()
	if err != nil {
		l.WithError(err).Error("failed to list repositories")
		return fmt.Errorf("start: error listing repositories: %v", err)
	}
	for _, repo := range repos {
		ll := l.WithField("repoUrl", repo.URL)
		err = h.SendToRepoInputQueue(repo)
		if err != nil {
			ll.WithError(err).Error("failed to send to repo input queue")
			return fmt.Errorf("start: error sending to issue input queue for repo %v: %v", repo.URL, err)
		}
		err = h.UpdateLastUpdatedDate(repo.URL)
		if err != nil {
			ll.WithError(err).Error("failed to set last updated date")
			return fmt.Errorf("start: error updating last updated date for repo %v: %v", repo.URL, err)
		}
		ll.Info("processed repo")
	}
	return nil
}

func main() {
	awsRegion := os.Getenv("AWS_REGION")
	repositoryTableName := os.Getenv("REPOSITORY_TABLE_NAME")
	store, err := dynamo.NewRepoStore(awsRegion, repositoryTableName)
	if err != nil {
		logger.For(pkg, "main").WithError(err).Fatal("failed to create repository store")
	}
	processingTopic := os.Getenv("REPOSITORY_PROCESSING_TOPIC")
	s := sender.NewSNS(awsRegion, processingTopic)
	h := Handler{
		ListRepositories: func() (repos []data.Repository, err error) {
			repos, _, err = store.Query("github")
			return
		},
		SendToRepoInputQueue: func(r data.Repository) (err error) {
			return s.Send("repo", r)
		},
		UpdateLastUpdatedDate: func(repoURL string) error {
			return store.Update("github", repoURL, time.Now().UTC())
		},
	}
	lambda.Start(h.Handle)
}
