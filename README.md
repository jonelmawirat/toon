# toon

`toon` is a Go implementation of TOON (Token-Oriented Object Notation), a line-oriented and indentation-based format for the JSON data model.

Upstream TOON specification:
<https://github.com/toon-format/spec/blob/main/SPEC.md>

## Documentation map

- [docs/README.md](docs/README.md)
- [docs/api/library.md](docs/api/library.md)
- [docs/flows/runtime-flows.md](docs/flows/runtime-flows.md)
- [docs/logic/architecture.md](docs/logic/architecture.md)
- [docs/ops/runbook.md](docs/ops/runbook.md)

## Install

### Library

```sh
go get github.com/jonelmawirat/toon@latest
```

### CLI

```sh
go install github.com/jonelmawirat/toon/cmd/toon@latest
```

## TOON at a glance

### Object

```toon
id: 1
name: Alice
active: true
```

### Inline array (primitive values)

```toon
tags[3]: admin,ops,dev
```

### Tabular array (uniform objects with primitive cells)

```toon
users[2]{id,name,role}:
  1,Alice,admin
  2,Bob,user
```

### Expanded array (mixed / non-uniform values)

```toon
items[3]:
  - 1
  - name: Alice
  - hello
```

### Tabular array as first field of a list-item object

```toon
items[1]:
  - users[2]{id,name}:
      1,Ada
      2,Bob
    status: active
```

## Data model and helpers

Decode functions return `toon.Value`.

`toon.Value` can represent:
- `toon.Object`
- array values (materialized arrays and lazy strict-tabular arrays)
- primitives: `string`, `bool`, `nil`, `toon.Number`

Use helpers instead of direct type assertions when you are unsure of the concrete internal type:
- `toon.AsObject(v)` to read objects safely
- `toon.AsArray(v)` to read arrays safely
- `toon.Unbox(v)` to unwrap internal cell wrappers

### Important: strict tabular decode behavior

In strict mode (`WithStrict(true)`, default), tabular arrays may decode to an internal lazy array representation for performance.

Because of that, avoid `v.(toon.Array)` for decode results. Use `toon.AsArray(v)`.

## Library usage

### Decode with `Unmarshal`

```go
package main

import (
	"fmt"

	"github.com/jonelmawirat/toon"
)

func main() {
	input := []byte("users[2]{id,name,role}:\n  1,Alice,admin\n  2,Bob,user\ncount: 2")

	v, err := toon.Unmarshal(input)
	if err != nil {
		panic(err)
	}

	root, ok := toon.AsObject(v)
	if !ok {
		panic("root is not an object")
	}

	usersV, _ := root.Get("users")
	users, ok := toon.AsArray(usersV)
	if !ok {
		panic("users is not an array")
	}

	row0, ok := toon.AsObject(users[0])
	if !ok {
		panic("row is not an object")
	}
	name, _ := row0.Get("name")

	fmt.Println(name)
}
```

### Encode with `Marshal`

```go
package main

import (
	"fmt"

	"github.com/jonelmawirat/toon"
)

func main() {
	v := map[string]any{
		"users": []any{
			map[string]any{"id": 1, "name": "Alice", "active": true},
			map[string]any{"id": 2, "name": "Bob", "active": false},
		},
		"count": 2,
	}

	out, err := toon.Marshal(v)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))
}
```

### Streaming APIs (`Decoder` / `Encoder`)

```go
package main

import (
	"os"

	"github.com/jonelmawirat/toon"
)

func main() {
	d, err := toon.NewDecoder(os.Stdin, toon.WithStrict(true))
	if err != nil {
		panic(err)
	}
	v, err := d.Decode()
	if err != nil {
		panic(err)
	}

	e, err := toon.NewEncoder(os.Stdout, toon.WithEncoderIndent(2))
	if err != nil {
		panic(err)
	}
	if err := e.Encode(v); err != nil {
		panic(err)
	}
}
```

## Decoder options

- `WithStrict(bool)`
  - default: `true`
  - strict validation of indentation/counts/shape
- `WithDecoderIndent(int)`
  - default: `2`
  - indentation width in spaces
- `WithExpandPaths(ExpandPathsMode)`
  - values: `ExpandPathsOff` (default), `ExpandPathsSafe`
  - safely expands dotted keys (for example `a.b.c`) into nested objects

Example:

```go
v, err := toon.Unmarshal(
	[]byte("a.b.c: 1\na.b.d: 2\na.e: 3"),
	toon.WithExpandPaths(toon.ExpandPathsSafe),
)
```

## Encoder options

- `WithEncoderIndent(int)`
  - default: `2`
- `WithDocDelimiter(Delimiter)`
  - values: `Comma`, `Tab`, `Pipe`
- `WithArrayDelimiter(Delimiter)`
  - values: `Comma`, `Tab`, `Pipe`
- `WithKeyFolding(KeyFoldingMode)`
  - values: `KeyFoldingOff` (default), `KeyFoldingSafe`
- `WithFlattenDepth(int)`
  - default: effectively unlimited
  - only used when key folding is enabled

Example:

```go
out, err := toon.Marshal(
	map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
	toon.WithKeyFolding(toon.KeyFoldingSafe),
	toon.WithFlattenDepth(3),
)
```

## Number behavior

`toon.Number` stores canonical decimal text.

Decode behavior:
- valid numeric tokens decode to `toon.Number`
- exponent forms are canonicalized to decimal (for example `1e-6` -> `0.000001`)
- forbidden-leading-zero numeric-like tokens decode as `string` (for example `05`)
- numeric-like tokens that cannot be canonicalized are preserved as `string`

Encode behavior:
- pass canonical numeric strings as `toon.Number`
- floats normalize to canonical decimal text
- `NaN` and `+/-Inf` normalize to `null`

## CLI usage

`toon` reads TOON from `stdin`.

Commands:
- `toon validate`
- `toon fmt`

Examples:

```sh
cat input.toon | toon validate
cat input.toon | toon fmt
```

Useful flags:
- `-strict=true|false`
- `-expand-paths=off|safe`
- `-indent=2`
- `-doc-delim=,|tab|pipe`
- `-array-delim=,|tab|pipe`
- `-fold=off|safe`
- `-flatten-depth=<n>`

Example:

```sh
cat input.toon | toon fmt -strict=true -expand-paths=safe -array-delim=tab -fold=safe -flatten-depth=4
```

## Migration notes

If you previously used direct assertions like `v.(toon.Array)` after decode, update to:

```go
arr, ok := toon.AsArray(v)
```

This is required for compatibility with lazy strict-tabular decode paths.

See:
- [MIGRATION.md](MIGRATION.md)
- [example_test.go](example_test.go)

## Development

Run tests:

```sh
GOCACHE=/tmp/go-cache go test ./...
```

Run key benchmarks:

```sh
GOCACHE=/tmp/go-cache go test -run '^$' -bench 'BenchmarkDecodeTabular1000|BenchmarkDecodeSmall|BenchmarkEncodeSmallMap' -benchmem
```

Run upstream fixture conformance (download done separately):

```sh
GOCACHE=/tmp/go-cache go run ./cmd/toonconformance -fixtures-dir /tmp/toon-spec-main/tests/fixtures -download=false
```
