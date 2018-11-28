package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/a-h/watchman/collector/github"
)

var token = flag.String("token", "", "GitHub auth token")
var repo = flag.String("repo", "", "GitHub repo URL")

func main() {
	flag.Parse()
	c := github.NewCollector(*token)
	issues, err := c.Issues(context.Background(), *repo)
	if err != nil {
		fmt.Printf("Error getting issue: %v\n", err)
		return
	}
	for _, issue := range issues {
		//TODO: Only get comments for issues that have been updated since the last time we checked.
		comments, err := c.Comments(context.Background(), issue.Owner, issue.Repo, issue.Number)
		if err != nil {
			fmt.Printf("Error getting comments for issue: %v\n", err)
			return
		}
		for _, c := range comments {
			fmt.Printf("Comment: %s\n", c.URL)
		}
	}
}
