BINARY    := slack-router
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X main.version=$(VERSION) \
           -X main.commit=$(COMMIT) \
           -X main.buildDate=$(BUILD_DATE) \
           -s -w

GO_BUILD := go build -trimpath -ldflags "$(LDFLAGS)"

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64  \
	linux/arm64  \
	windows/amd64

.DEFAULT_GOAL := help

# ─── primary targets ────────────────────────────────────────────────────────

.PHONY: build
build: ## Build for the current platform
	$(GO_BUILD) -o $(BINARY) .

.PHONY: release
release: ## Cross-compile for all platforms → dist/
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out="dist/$(BINARY)-$(VERSION)-$$os-$$arch$$ext"; \
		printf "  building %-50s" "$$out ..."; \
		GOOS=$$os GOARCH=$$arch $(GO_BUILD) -o "$$out" . \
			&& echo "ok" || { echo "FAILED"; exit 1; }; \
	done
	@echo ""
	@echo "Artifacts in dist/:"
	@ls -lh dist/

.PHONY: clean
clean: ## Remove build artifacts (binary + dist/)
	rm -f $(BINARY)
	rm -rf dist/

# ─── development ────────────────────────────────────────────────────────────

.PHONY: run
run: build ## Build and run with config.yaml
	./$(BINARY) -config config.yaml

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: lint
lint: ## Run go vet
	go vet ./...

.PHONY: tidy
tidy: ## Tidy go modules
	go mod tidy

# ─── release helpers ────────────────────────────────────────────────────────

.PHONY: version
version: ## Print the current version (from git describe)
	@echo $(VERSION)

.PHONY: tag
tag: ## Create and push a release tag  (usage: make tag VERSION=v0.2.0)
ifndef VERSION
	$(error VERSION is not set. Usage: make tag VERSION=v0.x.y)
endif
	git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Created tag $(VERSION). Push with: git push origin $(VERSION)"

# ─── help ───────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show available targets
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Current version: $(VERSION)"
