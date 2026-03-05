# Operations Runbook

## Baseline validation

Run full tests:

```sh
GOCACHE=/tmp/go-cache go test ./...
```

Run conformance harness against upstream fixtures:

```sh
GOCACHE=/tmp/go-cache go run ./cmd/toonconformance -fixtures-dir /tmp/toon-spec-main/tests/fixtures -download=false
```

## Benchmarking

Run all benchmarks:

```sh
GOCACHE=/tmp/go-cache go test ./... -run '^$' -bench . -benchmem
```

Run focused benchmarks:

```sh
GOCACHE=/tmp/go-cache go test -run '^$' -bench 'BenchmarkDecodeTabular1000|BenchmarkDecodeSmall|BenchmarkEncodeSmallMap' -benchmem
```

## Profiling

Makefile targets:

```sh
make profile-cpu
make profile-mem
make profile-cpu-text
make profile-mem-text
make gc-trace
```

Output location: `profiles/`
- CPU profile: `profiles/cpu.out`, summary `profiles/cpu.txt`
- Memory profile: `profiles/mem.out`, summaries `profiles/mem_inuse.txt`, `profiles/mem_alloc.txt`
- GC trace: `profiles/gctrace.txt`

## Build and release artifacts

Build cross-platform CLI binaries:

```sh
make build
```

Artifacts are generated in `dist/`.

## Troubleshooting guide

Common failures and checks:
- Option parse errors: verify delimiters and mode flags (`off|safe`, `,|tab|pipe`).
- Decode strict errors: retry with `-strict=false` only for diagnostics.
- Conformance harness failures: inspect failing spec section and fixture file from harness output.
- Unexpected decode shape: use `AsObject` and `AsArray` instead of direct assertions.

## Documentation update trigger

Update docs whenever any of these change:
- CLI flags or defaults
- encode/decode option semantics
- value shape behavior or helper usage recommendations
- test/conformance/profiling commands
