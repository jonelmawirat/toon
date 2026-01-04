# toon

A Go implementation of TOON (Token-Oriented Object Notation): a line-oriented, indentation-based text format that encodes the JSON data model with explicit structure, minimal quoting, and efficient tabular arrays.

This repository includes the spec as `SPEC.md` and follows the upstream spec as the basis:
https://github.com/toon-format/spec/blob/main/SPEC.md

## Install

### Library

```sh
go get github.com/jonelmawirat/toon@v1.0.0
```

### CLI

```sh
go install github.com/jonelmawirat/toon/cmd/toon@v1.0.0
```

## Format overview

### Object

```toon
id: 1
name: Alice
active: true
```

### Inline array (primitives)

```toon
tags[3]: admin,ops,dev
```

### Tabular array (array of uniform objects with primitive values)

```toon
users[2]{id,name,role}:
  1,Alice,admin
  2,Bob,user
```

### Expanded array (mixed / non-uniform)

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

## Go library usage

### Decode (Unmarshal)

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

    root := v.(toon.Object)

    usersV, _ := root.Get("users")
    users := usersV.(toon.Array)

    u0 := users[0].(toon.Object)
    name, _ := u0.Get("name")

    fmt.Println(name)
}
```

### Encode (Marshal)

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

### Decoder options

Strict mode is enabled by default. You can disable it for more lenient parsing:

```go
v, err := toon.Unmarshal([]byte("tags[2]: a"), toon.WithStrict(false))
```

Path expansion turns dotted keys into nested objects (safe mode only expands identifier segments):

```go
input := []byte("a.b.c: 1\na.b.d: 2\na.e: 3")
v, err := toon.Unmarshal(input, toon.WithExpandPaths(toon.ExpandPathsSafe))
```

### Encoder options

Choose document delimiter and array delimiter:

```go
out, err := toon.Marshal(
    map[string]any{"tags": []any{"a", "b", "c"}},
    toon.WithArrayDelimiter(toon.Tab),
)
```

Key folding can produce dotted keys for eligible object chains:

```go
out, err := toon.Marshal(
    map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
    toon.WithKeyFolding(toon.KeyFoldingSafe),
)
```

Limit folding depth:

```go
out, err := toon.Marshal(
    map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": 1}}}},
    toon.WithKeyFolding(toon.KeyFoldingSafe),
    toon.WithFlattenDepth(2),
)
```

### Value types

Decoding returns `toon.Value` which is one of:

- `toon.Object`
- `toon.Array`
- primitives:
  - `string`
  - `bool`
  - `nil`
  - `toon.Number` (canonical decimal string)

If you want to convert `toon.Number` to other numeric types, parse `string(n)` according to your needs.

## CLI usage

The CLI reads TOON from stdin and either validates it or reformats it.

### Validate

```sh
cat input.toon | toon validate
```

### Format

```sh
cat input.toon | toon fmt
```

### Useful flags

Strict mode on/off:

```sh
cat input.toon | toon validate -strict=true
cat input.toon | toon validate -strict=false
```

Safe path expansion:

```sh
cat input.toon | toon fmt -expand-paths=safe
```

Indentation width:

```sh
cat input.toon | toon fmt -indent=2
```

Encoding delimiters:

```sh
cat input.toon | toon fmt -doc-delim=pipe -array-delim=tab
```

Key folding on output:

```sh
cat input.toon | toon fmt -fold=safe -flatten-depth=4
```

## Development

Run tests:

```sh
go test ./...
```
