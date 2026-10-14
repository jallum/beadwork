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

To distinguish a custom build from the upstream release, supply build metadata:

```sh
make bw BUILD_METADATA=local
./bin/bw --version             # bw 0.13.3+local
```

`make install BUILD_METADATA=local` applies the same label when installing.
Use a label such as `local.prompts.1` to distinguish successive custom builds.
The label only changes `--version` output; upgrade checks compare the base release
number. Omitting `BUILD_METADATA` produces the usual release version output.

Plain `go install ./cmd/bw` omits the label. To apply it directly with Go:

```sh
go install -ldflags '-X main.buildMetadata=local' ./cmd/bw
```

Check `command -v bw` against `go env GOBIN GOPATH`: Go installs into `GOBIN`,
or `GOPATH/bin` if `GOBIN` is unset. An older binary earlier on your PATH can still
be selected. For example, if your shell finds `~/.local/bin/bw`, install there:

```sh
GOBIN="$HOME/.local/bin" make install BUILD_METADATA=local
bw --version
```

## Working on prompts

`bw config list` shows `prompts.prime` and `prompts.start`; their empty defaults
select the embedded templates. To try project-specific guidance, edit custom
Markdown files and select them with `bw config set prompts.prime <path>` or
`bw config set prompts.start <path>`. Changes to those files need no rebuild.
Set either option to `""` to restore its built-in template.

To change the embedded defaults for all users, edit `prompts/prime.md` or
`prompts/start.md` and rebuild with `make bw`. Use `./bin/bw` to verify your build;
`bw` on your PATH may point to a different installation.

See the [template reference](docs/design-templates.md) for variables, file loading,
and errors, and the [prompt guide](docs/prompts/prompts.md) for behavior testing.

## Pre-flight before pushing

```
make preflight
```

This runs the same checks CI runs on every PR (format → tidy-check → build → vet → staticcheck → test-cover). If it passes locally, CI will too.

## Filing changes

Open a PR against `main`. Keep changes focused; a separate PR per concern is easier to review than a bundle.
