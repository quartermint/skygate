.PHONY: build test test-go test-bats lint lint-ansible deploy clean cross-build help

# Go settings
GOOS ?= linux
GOARCH ?= arm64
BINARY_NAME = skygate-bypass

# Ansible settings
ANSIBLE_DIR = pi/ansible
PI_HOST ?= skygate

## Build

build: ## Build bypass daemon for current platform
	go build -o bin/$(BINARY_NAME) ./cmd/bypass-daemon/

cross-build: ## Cross-compile bypass daemon for Pi (linux/arm64)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o bin/$(BINARY_NAME)-$(GOOS)-$(GOARCH) ./cmd/bypass-daemon/

## Test

test: test-go test-bats lint ## Run all tests

test-go: ## Run Go unit tests
	go test ./... -v -short

test-bats: ## Run BATS tests
	bats pi/scripts/tests/

## Lint

lint: lint-ansible ## Run all linters

lint-ansible: ## Lint Ansible playbooks
	cd $(ANSIBLE_DIR) && ansible-lint playbook.yml

## Deploy

deploy: cross-build ## Deploy to Pi via Ansible
	cd $(ANSIBLE_DIR) && ansible-playbook playbook.yml

deploy-check: ## Dry-run Ansible deploy
	cd $(ANSIBLE_DIR) && ansible-playbook playbook.yml --check --diff

## Clean

clean: ## Remove build artifacts
	rm -rf bin/

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
