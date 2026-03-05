# API Reference

Package: `github.com/jonelmawirat/toon`

## Core library APIs

One-shot APIs:
- `Marshal(v any, opts ...EncoderOption) ([]byte, error)`
- `Unmarshal(data []byte, opts ...DecoderOption) (Value, error)`

Streaming APIs:
- `NewEncoder(w io.Writer, opts ...EncoderOption) (*Encoder, error)`
- `(*Encoder).Encode(v any) error`
- `NewDecoder(r io.Reader, opts ...DecoderOption) (*Decoder, error)`
- `NewDecoderBytes(data []byte, opts ...DecoderOption) (*Decoder, error)`
- `(*Decoder).Decode() (Value, error)`
- `(*Decoder).Reset(r io.Reader) error`
- `(*Decoder).ResetBytes(data []byte)`

Decode helpers:
- `AsObject(v Value) (Object, bool)`
- `AsArray(v Value) (Array, bool)`
- `Unbox(v Value) Value`

## Value model

Decode returns `Value` with these shapes:
- `Object`
- array values (including strict-tabular lazy array representations)
- primitives: `string`, `bool`, `nil`, `Number`

For decode results, prefer helper accessors over direct type assertions.

Safe pattern:
```go
v, err := toon.Unmarshal(data)
if err != nil {
	return err
}
obj, ok := toon.AsObject(v)
if !ok {
	return fmt.Errorf("expected object")
}
```

## Decoder options

- `WithStrict(bool)` (default `true`)
- `WithDecoderIndent(int)` (default `2`)
- `WithExpandPaths(ExpandPathsMode)` (`ExpandPathsOff` or `ExpandPathsSafe`)

## Encoder options

- `WithEncoderIndent(int)` (default `2`)
- `WithDocDelimiter(Delimiter)` (`Comma`, `Tab`, `Pipe`)
- `WithArrayDelimiter(Delimiter)` (`Comma`, `Tab`, `Pipe`)
- `WithKeyFolding(KeyFoldingMode)` (`KeyFoldingOff`, `KeyFoldingSafe`)
- `WithFlattenDepth(int)` (active when key folding is enabled)

## CLI contract

Command: `toon`

Modes:
- `toon fmt` (default)
- `toon validate`

Flags:
- `-strict=true|false`
- `-expand-paths=off|safe`
- `-indent=<n>`
- `-doc-delim=,|tab|pipe`
- `-array-delim=,|tab|pipe`
- `-fold=off|safe`
- `-flatten-depth=<n>`

Input: reads TOON from `stdin`.
Output: `fmt` writes normalized TOON to `stdout`; `validate` only returns status.
