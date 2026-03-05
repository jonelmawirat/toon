# Architecture and Module Responsibilities

## High-level modules

- `api.go`: public constructors and top-level encode/decode entry points.
- `types.go`: public value types, options, errors, and helper conversion APIs.
- `options.go`: validation and application of encoder/decoder options.
- `decoder.go`: parser orchestration and root-shape handling.
- `scan.go`, `lex.go`, `header.go`: line scanning, token splitting, and array header parsing.
- `primitive.go`, `number.go`, `strings.go`: primitive token parsing and canonicalization.
- `encoder.go`: object/array/primitive encoding paths and shape selection.
- `normalize.go`, `fold.go`, `expand.go`: pre-encode normalization, key folding, dotted-path expansion.
- `readall.go`: optimized read path for stream decode.

## Internal representation strategy

- Public decode contract returns `Value`.
- Objects are represented by ordered `Object{Members}`.
- Strict tabular decode can use packed internal cells and lazy row/array wrappers.
- Public helpers (`AsObject`, `AsArray`, `Unbox`) materialize and normalize these shapes safely.

## Design invariants

- Encode output never ends with a trailing newline.
- Strict mode enforces indentation/count/shape constraints from spec-compatible behavior.
- Numeric canonicalization preserves valid numbers as `Number` and keeps invalid numeric-like tokens as `string`.
- Key order in `Object.Members` is preserved and reflected in encode output.

## Change impact map

When changing parser logic:
- re-run `go test ./...`
- re-run conformance harness (`cmd/toonconformance`)
- validate decode helper compatibility (`AsObject`, `AsArray`)

When changing encoding decisions:
- verify quoting/delimiter tests
- verify key-folding and flatten-depth tests

When changing performance internals:
- keep public decode helper behavior stable
- update migration/docs if caller-facing access patterns change
