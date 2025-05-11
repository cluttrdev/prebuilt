
REPO_URL := "https://github.com/cluttrdev/prebuilt"

PKG ?= .
APP ?= .

.DEFAULT_GOAL := help

.PHONY: fmt
fmt: ## Format source code
	go fmt ${PKG}/...

.PHONY: lint
lint: ## Run set of static code analysis tools
	golangci-lint run ${PKG}/...

.PHONY: vet
vet: ## Examine code for suspicious constructs
	go vet ${PKG}/...

.PHONY: build
build:  ## Create application binary
	if [ -z "${os}" ]; then goos=$$(go env GOOS); else goos="${os}"; fi; \
	if [ -z "${arch}" ]; then goarch=$$(go env GOARCH); else goarch="${arch}"; fi; \
	if [ -z "${output}" ]; then output="bin/"; else output="${output}"; fi; \
	GOOS="$${goos}" GOARCH="$${goarch}" \
	CGO_ENABLED=0 \
	go build \
		-ldflags "-s -w" \
		-o "$${output}" \
		${APP}

.PHONY: test
test: ## Run tests
	go test ${PKG}/...

.PHONY: changes
changes: ## Get commits since last release
	to=HEAD; \
	if [ -n "${to}" ]; then to="${to}"; fi; \
	from=$$(git describe --tags --abbrev=0 "$${to}^" 2>/dev/null); \
	if [ -n "${from}" ]; then from="${from}"; fi; \
	if [ -n "$${from}" ]; then \
		git log --oneline --no-decorate $${from}..$${to}; \
	else \
		git log --oneline --no-decorate $${to}; \
	fi

.PHONY: changelog
changelog:
	printf "# Changelog\n\n"; \
	latest=$$(git describe --tags --abbrev=0 2>/dev/null); \
	changes=$$(make --no-print-directory changes from="$${latest}" | awk '{ print "- " $$0 }'); \
	if [ -n "$${changes}" ]; then \
		url="${REPO_URL}/-/compare/$${latest}..HEAD"; \
		printf "## [Unreleased](%s)\n\n%s\n\n" "$${url}" "$${changes}"; \
	fi; \
	for tag in $$(git tag --list | sort --version-sort --reverse); do \
		previous=$$(git describe --tags --abbrev=0 "$${tag}^" 2>/dev/null); \
		changes=$$(make --no-print-directory changes to=$${tag} | awk '{ print "- " $$0 }'); \
		if [ -n "$${previous}" ]; then \
			url="${REPO_URL}/-/compare/$${previous}..$${tag}"; \
		else \
			url="${REPO_URL}/-/commits/$${tag}"; \
		fi; \
		printf "## [%s](%s)\n\n%s\n\n" "$${tag#v}" "$${url}" "$${changes}"; \
	done

.PHONY: help
help: ## Display this help page
	grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[33m%-30s\033[0m %s\n", $$1, $$2}'

ifneq "${VERBOSE}" "1"
.SILENT:
endif
