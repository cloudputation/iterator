# GNUmakefile

.DEFAULT_GOAL := all
# Configure shell path
SHELL := /bin/bash

# Name of the binary to be built
BINARY_NAME := iterator

# Source directory
SRC_DIR := .

# Build directory
BUILD_DIR := ./build
ARTIFACTS_DIR := ./artifacts

# Exclude specific directories and/or file patterns
EXCLUDE_DIR := ./tests
EXCLUDE_PATTERN := *.back.go

# Find command adjusted to exclude the specified directories and patterns
SOURCES := $(shell find $(SRC_DIR) -name '*.go' ! -path "$(EXCLUDE_DIR)/*" ! -name "$(EXCLUDE_PATTERN)")

# Docker-related variables
DOCKER_IMAGE := iterator
IMAGE_DISTRIBUTOR := cloudputation
DOCKER_TAG := $(shell cat ./VERSION)
VERSION_MAJOR := $(shell echo $(DOCKER_TAG) | cut -d '.' -f1)
VERSION_MINOR := $(shell echo $(DOCKER_TAG) | cut -d '.' -f2)
VERSION_PATCH := $(shell echo $(DOCKER_TAG) | cut -d '.' -f3)

# Phony targets for make commands
.PHONY: all
all: mod inst gen spell lint test build docker-build docker-push ## run all targets

.PHONY: release
release: all ## run all targets for release

# CI build pipeline
.PHONY: ci
ci: mod inst gen spell test build docker-build ## run CI pipeline (skip docker-push)


# Extract release changelog
.PHONY: changelog
changelog: ## extract release changelog
	@echo "Extracting release changelog"
	bash tools/changelog.sh

.PHONY: mod
mod: ## go mod tidy
	$(call print-target)
	go mod tidy
	cd tools && go mod tidy

.PHONY: inst
inst: ## go install tools
	$(call print-target)
	cd tools && go install $(shell cd tools && go list -e -f '{{ join .Imports " " }}' -tags=tools)

.PHONY: gen
gen: ## go generate
	$(call print-target)
	go generate ./...

# .PHONY: build
# build: ## goreleaser build
# 	$(call print-target)
# 	goreleaser build --rm-dist --single-target --snapshot

.PHONY: spell
spell: ## misspell
	$(call print-target)
	misspell -locale=US -w **.md

.PHONY: lint
lint: ## golangci-lint
	$(call print-target)
	-golangci-lint run --fix

.PHONY: test
test: ## go test
	$(call print-target)
	-go test -race -covermode=atomic -coverprofile=coverage.out -coverpkg=./... ./...
	-go tool cover -html=coverage.out -o coverage.html

.PHONY: diff
diff: ## git diff
	@echo "Checking for uncommitted changes..."
	@git diff
	@untracked=$$(git status --porcelain) ; \
	if [ -n "$$untracked" ]; then \
		echo "Untracked or modified files:" ; \
		echo "$$untracked" ; \
	fi

# Build binary
.PHONY: build
build: $(SOURCES) ## build binary
	@echo "Downloading dependencies..."
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod download
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)

# Build the Docker image
.PHONY: docker-build
docker-build: build ## build Docker container image
	@echo "Building the Docker image..."
	docker build --build-arg PRODUCT_VERSION=$(DOCKER_TAG) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):$(VERSION_MAJOR)
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):$(VERSION_MAJOR).$(VERSION_MINOR)
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_PATCH)

# Push the Docker image to registry
.PHONY: docker-push
docker-push: ## push Docker image
	@echo "Pushing the Docker images..."
	docker push $(IMAGE_DISTRIBUTOR)/$(DOCKER_IMAGE):latest
	docker push $(IMAGE_DISTRIBUTOR)/$(DOCKER_IMAGE):$(VERSION_MAJOR)
	docker push $(IMAGE_DISTRIBUTOR)/$(DOCKER_IMAGE):$(VERSION_MAJOR).$(VERSION_MINOR)
	docker push $(IMAGE_DISTRIBUTOR)/$(DOCKER_IMAGE):$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_PATCH)


# Clean up
.PHONY: clean
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(ARTIFACTS_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.*
	@rm -f '"$(shell go env GOCACHE)/../golangci-lint"'
	go clean -i -cache -testcache -modcache -fuzzcache -x

# help
help:
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'



define print-target
    @printf "Executing target: \033[36m$@\033[0m\n"
endef
