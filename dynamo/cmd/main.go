package main

import (
	"fmt"

	"github.com/a-h/watchman/observer/dynamo"
)

const repoTableName = "watchman-observer-prod-repository"
const commentTableName = "watchman-observer-prod-comment"

func main() {
	repo, err := dynamo.NewRepoStore("eu-west-2", repoTableName)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = repo.Add("github.com", "https://github.com/a-h/generate", "https://github.com/a-h/watchman")
	if err != nil {
		fmt.Println("err:", err)
	}
}
