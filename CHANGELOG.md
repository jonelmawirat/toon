# Changelog

## Unreleased

- Documentation overhaul:
  - full README usage guide for library + CLI
  - explicit migration guidance for strict tabular decode array handling
  - refreshed examples using `AsObject` / `AsArray`
- Added `MIGRATION.md` and runnable `Example*` tests for decode helper patterns
- Added package-level GoDoc in `doc.go`
- Added structured docs set under `docs/`:
  - `docs/api/library.md`
  - `docs/flows/runtime-flows.md`
  - `docs/logic/architecture.md`
  - `docs/ops/runbook.md`
- Expanded exported symbol comments for GoDoc clarity (`Unbox`, delimiter/mode constants)

## v1.0.0

- Initial public release
- Go library: Marshal/Unmarshal, Encoder/Decoder, strict mode, delimiter control
- Tabular arrays, inline arrays, expanded arrays, arrays-of-arrays
- Key folding (safe) and path expansion (safe)
- CLI: `toon fmt` and `toon validate`
