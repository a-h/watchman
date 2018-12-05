build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/start start/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/repo repo/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/issue issue/main.go

.PHONY: clean
clean:
	rm -rf ./bin

.PHONY: deploy
deploy: clean build
	sls deploy --verbose
