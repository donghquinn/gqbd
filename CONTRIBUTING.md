# Contributing to GQBD

**English** | [한국어](./CONTRIBUTING.ko.md)

Thanks for helping. GQBD is a single Go package with no dependencies, so getting started takes a minute.

## Prerequisites

- Go 1.26.10 or newer (the minimum in `go.mod`)
- Optional: [golangci-lint v2](https://golangci-lint.run/), [govulncheck](https://go.dev/doc/security/vuln/), [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)

## Before opening a pull request

Run the same checks as CI:

```bash
gofmt -l .                       # must print nothing
go vet ./...
go test -race -shuffle=on ./...
golangci-lint run
govulncheck ./...
```

No test needs a database: every builder returns query text, arguments or request bodies, and the tests compare those.

## Changing hot paths

`gqbd.go`, `build.go`, `dialect.go` and the `Build` methods of the other builders are allocation-sensitive. If you touch them, compare benchmarks before and after:

```bash
git stash && go test -run='^$' -bench=. -benchmem -count=6 . > /tmp/before.txt
git stash pop && go test -run='^$' -bench=. -benchmem -count=6 . > /tmp/after.txt
benchstat /tmp/before.txt /tmp/after.txt
```

Paste the `benchstat` output in the pull request. See [docs/performance.md](docs/performance.md) for the techniques the builders rely on.

## Adding a SQL database

Database differences live in one table in `dialect.go`. A new SQL database is usually one `dialect` entry, a `DBType` constant, a case in `dialectFor` and `NewQueryBuilder`, a connection-string builder in `connection.go`, tests, and a page under `docs/` (with its Korean translation under `docs/ko/`).

## Documentation

English is the source of truth. When you change `README.md`, `CONTRIBUTING.md` or a page in `docs/`, update the Korean file (`README.ko.md`, `CONTRIBUTING.ko.md`, `docs/ko/...`) in the same pull request. If you can't, say so in the pull request so a maintainer can.

## Continuous integration

| Workflow | Runs on | What it does |
|----------|---------|--------------|
| `test.yml` | push to `main`/`dev`, pull requests | `go mod tidy` check, gofmt, vet, race tests on the minimum and latest Go, benchmark smoke run, fuzzing, coverage, golangci-lint |
| `security.yml` | push, pull requests, weekly | govulncheck, gosec, CodeQL (security-extended), dependency review on pull requests |
| `release.yml` | `v*.*.*` tags | re-runs tests and govulncheck, creates the GitHub Release with generated notes, and publishes the version to the Go module proxy |

## Releasing (maintainers)

GQBD follows [semantic versioning](https://semver.org/). Tag a commit on `main` and push the tag:

```bash
git tag v1.6.0
git push origin v1.6.0
```

The release workflow does the rest. Tags with a suffix (`v1.6.0-rc.1`) are marked as pre-releases.
