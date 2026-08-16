SHELL := /bin/bash

FRONTEND_DIR := frontend
WAILSJS_DIR := $(FRONTEND_DIR)/wailsjs

.DEFAULT_GOAL := help

.PHONY: help build test lint frontend-build frontend-test verify codegen codegen-check format-check go-lint frontend-lint

help:
	@printf '%s\n' \
		'Nil local workflows' \
		'' \
		'  make build           Build the desktop app with Wails' \
		'  make test            Run the Go test suite and frontend tests' \
		'  make lint            Run Go format checks, hook-equivalent Go lint, and frontend lint' \
		'  make frontend-build  Run the standalone frontend production build' \
		'  make frontend-test   Run the frontend test suite (Vitest)' \
		'  make codegen         Regenerate Wails TypeScript bindings' \
		'  make codegen-check   Regenerate bindings and fail if tracked files changed' \
		'  make verify          Run the full local verification path'

build:
	wails build

test: frontend-test
	go test ./...

lint: format-check go-lint frontend-lint

frontend-build:
	npm --prefix $(FRONTEND_DIR) run build

frontend-test:
	npm --prefix $(FRONTEND_DIR) run test

verify: lint test frontend-build codegen-check build

codegen:
	wails generate module

codegen-check:
	wails generate module
	@changed="$$(git diff --name-only -- $(WAILSJS_DIR))"; \
	if [ -n "$$changed" ]; then \
		printf '%s\n' "$$changed" | while IFS= read -r file; do \
			[ -n "$$file" ] || continue; \
			mode="$$(git ls-files --stage -- "$$file" | awk '{print $$1}')"; \
			case "$$mode" in \
				100644) chmod 644 "$$file" ;; \
				100755) chmod 755 "$$file" ;; \
			esac; \
		done; \
	fi; \
	git diff --exit-code -- $(WAILSJS_DIR) || \
		(printf '%s\n' "Generated Wails bindings are out of date. Review changes under $(WAILSJS_DIR)." && exit 1)

format-check:
	@unformatted="$$(gofmt -l . 2>/dev/null)"; \
	if command -v goimports >/dev/null 2>&1; then \
		goimports_out="$$(goimports -l . 2>/dev/null)"; \
		if [ -n "$$goimports_out" ]; then \
			unformatted="$${unformatted}"$${unformatted:+$$'\n'}"$$goimports_out"; \
		fi; \
	fi; \
	unformatted="$$(printf '%s\n' "$$unformatted" | sed '/^$$/d' | sort -u)"; \
	if [ -n "$$unformatted" ]; then \
		printf '%s\n' 'Unformatted Go files:' "$$unformatted"; \
		exit 1; \
	fi

go-lint:
	golangci-lint run --new --timeout 2m

frontend-lint:
	@files="$$(git diff --name-only --diff-filter=ACMR HEAD -- '*.ts' '*.tsx' '*.js' '*.jsx' | sed "s#^$(FRONTEND_DIR)/##")"; \
	if [ -z "$$files" ]; then \
		printf '%s\n' 'No changed frontend files to lint.'; \
		exit 0; \
	fi; \
	(cd $(FRONTEND_DIR) && npx biome check --no-errors-on-unmatched $$files)
