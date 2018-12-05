package main

import (
	"context"
	"fmt"

	"github.com/a-h/watchman/observer/data"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler handles incoming CloudWatch Timer elapsed events.
type Handler struct {
	ListRepositories      data.RepositoryLister
	SendToRepoInputQueue  func(r data.Repository) error
	UpdateLastUpdatedDate data.RepositoryUpdater
}

// Handle incoming CloudWatch Timer elapsed events.
func (h Handler) Handle(ctx context.Context) error {
	repos, err := h.ListRepositories()
	if err != nil {
		return fmt.Errorf("start: error listing repositories: %v", err)
	}
	for _, repo := range repos {
		err = h.SendToRepoInputQueue(repo)
		if err != nil {
			return fmt.Errorf("start: error sending to issue input queue for repo %v: %v", repo.URL, err)
		}
		err = h.UpdateLastUpdatedDate(repo.URL)
		if err != nil {
			return fmt.Errorf("start: error updating last updated date for repo %v: %v", repo.URL, err)
		}
	}
	return nil
}

func main() {
	//TODO: Populate handler.
	h := Handler{}
	lambda.Start(h.Handle)
}
