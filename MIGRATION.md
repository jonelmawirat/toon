# Migration Guide

This guide covers decode-side usage updates for newer `toon` versions.

## Why this change

Strict tabular decode paths now use lazy internal array representations for performance.

This improves decode speed and memory use, but it means direct type assertions like `v.(toon.Array)` are no longer always safe for decoded values.

## Required code change

### Before

```go
v, _ := toon.Unmarshal(input)
arr := v.(toon.Array)
```

### After

```go
v, _ := toon.Unmarshal(input)
arr, ok := toon.AsArray(v)
if !ok {
	// handle non-array
}
```

## Recommended decode pattern

Use conversion helpers for decoded values:

- `toon.AsObject(v)` for objects
- `toon.AsArray(v)` for arrays
- `toon.Unbox(v)` if you need to normalize internal wrappers to plain values

Example:

```go
v, err := toon.Unmarshal(input)
if err != nil {
	return err
}

root, ok := toon.AsObject(v)
if !ok {
	return fmt.Errorf("expected object root")
}

usersV, _ := root.Get("users")
users, ok := toon.AsArray(usersV)
if !ok {
	return fmt.Errorf("expected users array")
}
```

## Compatibility notes

- `Marshal` / `Encoder` still accept ordinary Go values and existing `toon` values.
- Conformance behavior remains aligned with the upstream spec.
- If your code already uses `AsObject` and `AsArray`, no migration is needed.
