# Tests and Benchmarks

Command reference for this repository.

## Prerequisites

- Go version from `go.mod` (currently Go 1.25)
- `CGO_ENABLED=1` (required for sqlite3 builds)

## Go tests

Run full test suite:

```bash
go test ./...
```

Run action/integration tests (live in `server`):

```bash
go test ./server
```

Run a single test by name:

```bash
go test ./server -run TestName
```

Disable test cache:

```bash
go test ./server -count=1
```

## E2E tests

```bash
npm run test:e2e
npm run test:e2e:ui
```

See `e2e/README.md` for required environment variables.

## Go benchmarks

Run file listing benchmark matrix:

```bash
go test ./server -run '^$' -bench '^BenchmarkFileListing$' -benchmem
```

Run cross-space listing benchmark:

```bash
go test ./server -run '^$' -bench '^BenchmarkFileListingAcrossTenSpaces$' -benchmem
```

Run browse FTS benchmark matrix:

```bash
go test ./server -run '^$' -bench '^BenchmarkBrowseFTSQuery$' -benchmem
```

Run one browse FTS sub-benchmark quickly (smoke check):

```bash
go test ./server -run '^$' -bench 'BenchmarkBrowseFTSQuery/single_space_1000$' -benchmem -benchtime=1x
```

Optional profiling:

```bash
go test ./server -run '^$' -bench '^BenchmarkFileListing$' -benchmem -cpuprofile cpu.out -memprofile mem.out
go test ./server -run '^$' -bench '^BenchmarkBrowseFTSQuery$' -benchmem -cpuprofile cpu.out -memprofile mem.out
```
