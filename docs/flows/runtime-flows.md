# Runtime Flows

## Decode flow (`Unmarshal` / `Decoder.Decode`)

1. Input bytes are scanned into logical lines (`scanString`) with indentation metadata.
2. Parser selects root form:
- object
- root array header
- root primitive
3. Object/array nodes are parsed recursively with strict/non-strict checks.
4. Primitive tokens are resolved to `string`, `bool`, `nil`, or `Number`.
5. In strict tabular arrays, packed/lazy internal representations may be produced for efficiency.
6. If enabled, dotted paths are expanded (`ExpandPathsSafe`).
7. Result is returned as `Value`.

Failure paths:
- indentation errors
- malformed headers or quoted strings
- strict count/width mismatches in arrays
- strict delimiter mismatches
- expand-path conflicts (strict mode)

## Encode flow (`Marshal` / `Encoder.Encode`)

1. Input is normalized to TOON value model.
2. Optional safe key folding is applied (`WithKeyFolding`, `WithFlattenDepth`).
3. Root is encoded by shape:
- object
- array (inline/tabular/expanded/array-of-arrays)
- primitive
4. Active delimiters (`doc` and `array`) drive quoting decisions.
5. Lines are joined with `\n` (no trailing newline).

Failure paths:
- unsupported value types during normalization
- invalid option values (indent/delimiter/mode)
- invalid key or primitive encoding constraints

## Strict-tabular decode note

Strict decode can return arrays that are not directly `toon.Array` assertions.
Always access arrays through `AsArray` for decoded values:

```go
arr, ok := toon.AsArray(v)
```

This keeps callers compatible with optimized internal representations.
