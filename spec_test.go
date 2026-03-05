package toon

import "testing"

func TestDecodeInlineArrayEmptyTokens(t *testing.T) {
	input := "tags[3]: a,,c"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	tagsV, _ := obj.Get("tags")
	tags := mustArray(t, tagsV)
	if len(tags) != 3 {
		t.Fatalf("expected 3, got %d", len(tags))
	}
	if tags[1] != "" {
		t.Fatalf("expected empty string, got %#v", tags[1])
	}
}

func TestDecodeDelimiterMismatchStrict(t *testing.T) {
	input := "items[1|]{a,b}:\n  1,2"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeTabularDisambiguationEndsRows(t *testing.T) {
	input := "items[2]{a,b}:\n  1,2\n  next: x"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error due to row count mismatch")
	}
}

func TestDecodeStrictTwoRootPrimitivesInvalid(t *testing.T) {
	input := "hello\nworld"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeQuotedKey(t *testing.T) {
	input := "\"my-key\": 1"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	got, ok := obj.Get("my-key")
	if !ok {
		t.Fatal("missing key")
	}
	if got != Number("1") {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestEncodeReservedLiteralQuoted(t *testing.T) {
	out, err := Marshal(map[string]any{"x": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"true\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestEncodeNumericLikeQuoted(t *testing.T) {
	out, err := Marshal(map[string]any{"x": "1e-6"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"1e-6\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestEncodeDocDelimiterAffectsQuoting(t *testing.T) {
	out, err := Marshal(map[string]any{"x": "a|b"}, WithDocDelimiter(Pipe))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"a|b\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestEncodeActiveDelimiterAffectsQuotingInlineArray(t *testing.T) {
	out, err := Marshal(map[string]any{"tags": []any{"a|b"}}, WithArrayDelimiter(Pipe))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "tags[1|]: \"a|b\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}
