# Contributing

Thanks for your interest in beadwork. This file covers the local dev loop.

## Prerequisites

- Go 1.24.4+ (matches CI)
- `make`

`staticcheck` is installed automatically the first time you need it.

## Make targets

The `Makefile` mirrors the CI pipeline (`.github/workflows/ci.yml`) so you can run the same checks locally before pushing.

```
make preflight     # fmt-check + tidy-check + build + vet + staticcheck + test-cover
make build         # go build ./...
make vet           # go vet ./...
make staticcheck   # honnef.co/go/tools/cmd/staticcheck ./...
make test          # go test ./... without coverage
make test-cover    # go test with the same coverage flags CI uses
make cover         # run tests, then print coverage summary
make fmt           # gofmt -w .
make fmt-check     # gofmt -l .  (fails if anything needs formatting)
make tidy          # go mod tidy
make tidy-check    # fail if go.mod/go.sum are not tidy
make bw            # build a local ./bin/bw binary you can run directly
make install       # go install ./cmd/bw to your $GOBIN
make clean         # remove build + coverage artifacts
make help          # list targets
```

`make build` is a compile check across all packages — it does not produce a runnable binary. Use `make bw` (writes `./bin/bw`) or `make install` for that.

## Pre-flight before pushing

```
make preflight
```

This runs the same checks CI runs on every PR (format → tidy-check → build → vet → staticcheck → test-cover). If it passes locally, CI will too.

## Filing changes

Open a PR against `main`. Keep changes focused; a separate PR per concern is easier to review than a bundle.
