# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

export PATH := $(abspath bin/):${PATH}

# Dependency versions
GOLANGCI_VERSION = 1.53.3
LICENSEI_VERSION = 0.8.0

.PHONY: build
build: ## Build binary
	@mkdir -p build
	go build -race -o build/

.PHONY: lint
lint: lint-go lint-yaml #lint-docker
lint: ## Run linters

.PHONY: lint-go
lint-go:
	golangci-lint run $(if ${CI},--out-format github-actions,)

#.PHONY: lint-docker
#lint-docker:
#	hadolint Dockerfile

.PHONY: lint-yaml
lint-yaml:
	yamllint $(if ${CI},-f github,) --no-warnings .

.PHONY: fmt
fmt: ## Format code
	golangci-lint run --fix

.PHONY: test
test: ## Run tests
	go test -race -v ./...

.PHONY: license-check
license-check: ## Run license check
	licensei check
	licensei header

deps: bin/golangci-lint bin/licensei
deps: ## Install dependencies

bin/golangci-lint:
	@mkdir -p bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- v${GOLANGCI_VERSION}

bin/licensei:
	@mkdir -p bin
	curl -sfL https://raw.githubusercontent.com/goph/licensei/master/install.sh | bash -s -- v${LICENSEI_VERSION}
