# Mirrors .github/workflows/ci.yml so `make preflight` matches what CI runs.

GO              ?= go
STATICCHECK     ?= staticcheck
STATICCHECK_VER ?= latest
BIN_DIR         ?= bin
BW_BIN          ?= $(BIN_DIR)/bw
COVER_PROFILE   ?= coverage.out
CLI_COVER       ?= cli-coverage.out

.PHONY: help
help:
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---- CI parity ----------------------------------------------------------

.PHONY: preflight
preflight: fmt-check tidy-check build vet staticcheck test-cover ## Run the full CI check sequence locally (with coverage)

.PHONY: build
build: ## go build ./...  (matches CI "Build" step)
	$(GO) build ./...

.PHONY: staticcheck
staticcheck: ensure-staticcheck ## staticcheck ./...  (matches CI "Staticcheck" step)
	$(STATICCHECK) ./...

.PHONY: test
test: ## go test ./... — fast, no coverage
	$(GO) test ./...

.PHONY: test-cover
test-cover: ## go test with the same coverage flags CI uses
	CLI_COVER_PROFILE=$(CURDIR)/$(CLI_COVER) \
	$(GO) test ./... -coverprofile=$(COVER_PROFILE) -covermode=atomic -coverpkg=./...

.PHONY: cover
cover: test-cover ## Show coverage summary after running tests
	$(GO) tool cover -func=$(COVER_PROFILE) | tail -20

# ---- Local convenience --------------------------------------------------

.PHONY: bw
bw: $(BIN_DIR) ## Build the bw binary into ./bin/bw for local use
	$(GO) build -o $(BW_BIN) ./cmd/bw

.PHONY: install
install: ## go install ./cmd/bw to your $GOBIN
	$(GO) install ./cmd/bw

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

.PHONY: tidy-check
tidy-check: ## go mod tidy, then fail if go.mod/go.sum changed
	$(GO) mod tidy
	git diff --exit-code -- go.mod go.sum

.PHONY: vet
vet: ## go vet ./...  (matches CI "Vet" step)
	$(GO) vet ./...

.PHONY: fmt
fmt: ## gofmt -w . — rewrite files in place
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## gofmt -l . — fail if any files need formatting
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		echo "gofmt: the following files need formatting:"; \
		echo "$$out"; \
		exit 1; \
	fi

.PHONY: clean
clean: ## Remove build + coverage artifacts
	rm -rf $(BIN_DIR) $(COVER_PROFILE) $(CLI_COVER) merged.out

# ---- Internal -----------------------------------------------------------

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: ensure-staticcheck
ensure-staticcheck:
	@command -v $(STATICCHECK) >/dev/null 2>&1 || { \
		echo ">> installing staticcheck@$(STATICCHECK_VER)"; \
		$(GO) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VER); \
	}
