.PHONY: build build-bypass build-dashboard build-tunnel build-proxy build-all test test-go test-bats test-proxy test-all lint lint-ansible deploy deploy-check clean cross-build cross-build-bypass cross-build-dashboard cross-build-tunnel docker-build docker-up docker-down help

# Go settings
GOOS ?= linux
GOARCH ?= arm64
BINARY_NAME = skygate-bypass
DASHBOARD_BINARY = skygate-dashboard
TUNNEL_BINARY = skygate-tunnel-monitor
PROXY_BINARY = skygate-proxy

# Ansible settings
ANSIBLE_DIR = pi/ansible
PI_HOST ?= skygate

## Build

build: build-bypass build-dashboard build-tunnel ## Build all Pi daemons for current platform

build-all: build build-proxy ## Build all daemons including proxy (requires libwebp-dev)

build-bypass: ## Build bypass daemon for current platform
	go build -o bin/$(BINARY_NAME) ./cmd/bypass-daemon/

build-dashboard: ## Build dashboard daemon for current platform
	go build -o bin/$(DASHBOARD_BINARY) ./cmd/dashboard-daemon/

build-tunnel: ## Build tunnel monitor for current platform
	go build -o bin/$(TUNNEL_BINARY) ./cmd/tunnel-monitor/

build-proxy: ## Build proxy server for current platform (requires libwebp-dev)
	CGO_ENABLED=1 go build -o bin/$(PROXY_BINARY) ./cmd/proxy-server/

cross-build: cross-build-bypass cross-build-dashboard cross-build-tunnel ## Cross-compile all daemons for Pi (linux/arm64)

cross-build-bypass: ## Cross-compile bypass daemon for Pi
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o bin/$(BINARY_NAME)-$(GOOS)-$(GOARCH) ./cmd/bypass-daemon/

cross-build-dashboard: ## Cross-compile dashboard daemon for Pi
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o bin/$(DASHBOARD_BINARY)-$(GOOS)-$(GOARCH) ./cmd/dashboard-daemon/

cross-build-tunnel: ## Cross-compile tunnel monitor for Pi
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o bin/$(TUNNEL_BINARY)-$(GOOS)-$(GOARCH) ./cmd/tunnel-monitor/

## Test

test: test-go test-bats lint ## Run all tests

test-all: test test-proxy ## Run all tests including proxy (requires libwebp-dev)

test-go: ## Run Go unit tests
	go test ./... -v -short

test-bats: ## Run BATS tests
	bats pi/scripts/tests/

test-proxy: ## Run proxy server tests (requires libwebp-dev)
	CGO_ENABLED=1 go test ./cmd/proxy-server/ -v -short

## Lint

lint: lint-ansible ## Run all linters

lint-ansible: ## Lint Ansible playbooks
	cd $(ANSIBLE_DIR) && ansible-lint playbook.yml

## Docker

docker-build: ## Build proxy Docker image
	docker build -f server/Dockerfile.proxy -t skygate-proxy .

docker-up: ## Start remote server stack (WireGuard + proxy)
	cd server && docker compose up -d

docker-down: ## Stop remote server stack
	cd server && docker compose down

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
