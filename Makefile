GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

.PHONY: init
# init env
init:
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest

.PHONY: generate
# generate
generate:
	go mod tidy
	go generate ./...

.PHONY: all
# generate all
all:
	make init;
	make generate;

doc: .
	gomarkdoc --output '{{.Dir}}/README.md' ./...

go_fmt:
	gofumpt -w .

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
