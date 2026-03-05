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

func TestEncodeToonObjectPreservesOrderAndArrayHeaders(t *testing.T) {
	v := Object{Members: []Member{
		{Key: "id", Value: Number("123")},
		{Key: "name", Value: "Ada"},
		{Key: "active", Value: true},
		{Key: "tags", Value: Array{"reading", "gaming"}},
	}}
	out, err := Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "id: 123\nname: Ada\nactive: true\ntags[2]: reading,gaming" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestEncodeEmptyStringKeyArrayHeader(t *testing.T) {
	v := Object{Members: []Member{
		{Key: "", Value: Array{Number("1"), Number("2"), Number("3")}},
	}}
	out, err := Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "\"\"[3]: 1,2,3" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestEncodeKeyFoldingSafeSkipsSiblingCollision(t *testing.T) {
	v := Object{Members: []Member{
		{Key: "data", Value: Object{Members: []Member{
			{Key: "meta", Value: Object{Members: []Member{
				{Key: "items", Value: Array{Number("1"), Number("2")}},
			}}},
		}}},
		{Key: "data.meta.items", Value: "literal"},
	}}

	out, err := Marshal(v, WithKeyFolding(KeyFoldingSafe))
	if err != nil {
		t.Fatal(err)
	}
	want := "data:\n  meta:\n    items[2]: 1,2\ndata.meta.items: literal"
	if string(out) != want {
		t.Fatalf("unexpected:\n%s", string(out))
	}
}

func TestEncodeKeyFoldingSafeFlattenDepthTwo(t *testing.T) {
	v := Object{Members: []Member{
		{Key: "a", Value: Object{Members: []Member{
			{Key: "b", Value: Object{Members: []Member{
				{Key: "c", Value: Object{Members: []Member{
					{Key: "d", Value: Number("1")},
				}}},
			}}},
		}}},
	}}

	out, err := Marshal(v, WithKeyFolding(KeyFoldingSafe), WithFlattenDepth(2))
	if err != nil {
		t.Fatal(err)
	}
	want := "a.b:\n  c:\n    d: 1"
	if string(out) != want {
		t.Fatalf("unexpected:\n%s", string(out))
	}
}
