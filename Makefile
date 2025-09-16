.PHONY: help test test-blank-lines test-feature lint fmt vet bench bench-compare coverage clean install-tools install-gitversion version version-info version-patch version-minor version-major tag-release ci-local docs

# GitVersion configuration file
GITVERSION_CONFIG := GitVersion.yml

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

test: ## Run all tests
	go test -v -race ./...

test-blank-lines: ## Run blank line preservation tests
	go test -v -run TestBlankLinePreservation ./...

test-feature: ## Run feature tests with preservation enabled
	PRESERVE_BLANK_LINES=true go test -v -run "TestBlankLine|TestPerInstance" ./...

lint: ## Run linters
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

bench-compare: ## Run benchmarks with and without blank line preservation
	@echo "Running benchmarks WITHOUT preservation..."
	@go test -bench=BenchmarkBlankLinePreservation/WithoutPreservation -benchmem | tee without.txt
	@echo ""
	@echo "Running benchmarks WITH preservation..."
	@go test -bench=BenchmarkBlankLinePreservation/WithPreservation -benchmem | tee with.txt
	@echo ""
	@echo "Comparison complete. Results saved to without.txt and with.txt"

coverage: ## Generate coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	rm -f coverage.out coverage.html
	rm -f without.txt with.txt
	go clean -testcache

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

install-gitversion: ## Install GitVersion tool
	@which gitversion > /dev/null || (echo "Please install GitVersion from https://gitversion.net/docs/installation" && exit 1)

# GitVersion-based version management
version: install-gitversion ## Calculate version using GitVersion
	@gitversion /config $(GITVERSION_CONFIG) /showvariable SemVer

version-info: install-gitversion ## Show detailed version information from GitVersion
	@gitversion /config $(GITVERSION_CONFIG)

version-patch: install-gitversion ## Create a patch version tag using commit message
	@echo "Creating patch version bump commit..."
	@git commit --allow-empty -m "fix: Patch version bump"
	@NEW_VERSION=$$(gitversion /config $(GITVERSION_CONFIG) /showvariable SemVer); \
	echo "New version: v$$NEW_VERSION"; \
	git tag "v$$NEW_VERSION"

version-minor: install-gitversion ## Create a minor version tag using commit message
	@echo "Creating minor version bump commit..."
	@git commit --allow-empty -m "feat: Minor version bump"
	@NEW_VERSION=$$(gitversion /config $(GITVERSION_CONFIG) /showvariable SemVer); \
	echo "New version: v$$NEW_VERSION"; \
	git tag "v$$NEW_VERSION"

version-major: install-gitversion ## Create a major version tag using commit message
	@echo "Creating major version bump commit..."
	@git commit --allow-empty -m "breaking: Major version bump"
	@NEW_VERSION=$$(gitversion /config $(GITVERSION_CONFIG) /showvariable SemVer); \
	echo "New version: v$$NEW_VERSION"; \
	git tag "v$$NEW_VERSION"

tag-release: install-gitversion ## Create a release tag based on current GitVersion
	@NEW_VERSION=$$(gitversion /config $(GITVERSION_CONFIG) /showvariable SemVer); \
	echo "Creating release tag: v$$NEW_VERSION"; \
	git tag "v$$NEW_VERSION"; \
	echo "Tag created. Push with: git push origin v$$NEW_VERSION"

# CI simulation
ci-local: fmt vet lint test bench ## Run all CI checks locally
	@echo "✅ All CI checks passed!"

# Documentation
docs: ## Generate documentation
	@which godoc > /dev/null || go install golang.org/x/tools/cmd/godoc@latest
	@echo "Starting godoc server on http://localhost:6060"
	@echo "Package docs at: http://localhost:6060/pkg/github.com/elioetibr/yaml/"
	godoc -http=:6060