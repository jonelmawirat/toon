package toon

import "testing"

func TestEncodeTabular(t *testing.T) {
	v := map[string]any{
		"users": []any{
			map[string]any{"id": 1, "name": "Alice", "active": true},
			map[string]any{"id": 2, "name": "Bob", "active": false},
		},
		"count": 2,
	}
	out, err := Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if s == "" {
		t.Fatal("expected output")
	}
}

func TestEncodeNoTrailingNewline(t *testing.T) {
	out, err := Marshal(map[string]any{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > 0 && out[len(out)-1] == '\n' {
		t.Fatal("must not end with newline")
	}
}

func TestEncodeKeyFoldingSafe(t *testing.T) {
	v := map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}}
	out, err := Marshal(v, WithKeyFolding(KeyFoldingSafe))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "a.b.c: 1" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestEncodeDelimiterTab(t *testing.T) {
	v := map[string]any{
		"tags": []any{"a", "b", "c"},
	}
	out, err := Marshal(v, WithArrayDelimiter(Tab))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "tags[3\t]: a\tb\tc" {
		t.Fatalf("unexpected: %q", string(out))
	}
}
