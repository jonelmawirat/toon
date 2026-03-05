package toon

import (
	"math"
	"strings"
	"testing"
)

func TestConformance_EmptyDocumentIsEmptyObject(t *testing.T) {
	v, err := Unmarshal([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	if obj.Len() != 0 {
		t.Fatalf("expected empty object, got %#v", obj)
	}
}

func TestConformance_MissingColonInNestedObjectStrictErrors(t *testing.T) {
	input := "a:\n  b 1"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConformance_QuotedStringEscapes(t *testing.T) {
	input := "x: \"a\\\\b\\\"c\\n\\r\\t\""
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	x, _ := obj.Get("x")
	if x != "a\\b\"c\n\r\t" {
		t.Fatalf("unexpected: %#v", x)
	}
}

func TestConformance_InvalidEscapeErrors(t *testing.T) {
	input := "x: \"\\u1234\""
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConformance_NumberForbiddenLeadingZerosDecodesAsString(t *testing.T) {
	input := "x: 0001"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	x, _ := obj.Get("x")
	if x != "0001" {
		t.Fatalf("unexpected: %#v", x)
	}
}

func TestConformance_NumberExponentAcceptedAndCanonicalized(t *testing.T) {
	input := "x: -1E+03"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	x, _ := obj.Get("x")
	if x != Number("-1000") {
		t.Fatalf("unexpected: %#v", x)
	}
}

func TestConformance_DelimiterScoping_SplitOnlyOnActiveDelimiter(t *testing.T) {
	input := "x[3|]: a|b,c|d"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	xv, _ := obj.Get("x")
	a := mustArray(t, xv)
	if len(a) != 3 {
		t.Fatalf("expected 3, got %d", len(a))
	}
	if a[1] != "b,c" {
		t.Fatalf("expected token with comma preserved, got %#v", a[1])
	}
}

func TestConformance_InlineArray_EmptyTokenIsEmptyString(t *testing.T) {
	input := "x[3]: a,,c"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	xv, _ := obj.Get("x")
	a := mustArray(t, xv)
	if a[1] != "" {
		t.Fatalf("expected empty string, got %#v", a[1])
	}
}

func TestConformance_StrictInlineArrayCountMismatchErrors(t *testing.T) {
	input := "x[2]: a"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConformance_StrictTabularRowCountMismatchErrors(t *testing.T) {
	input := "x[2]{a,b}:\n  1,2"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConformance_StrictTabularRowWidthMismatchErrors(t *testing.T) {
	input := "x[1]{a,b}:\n  1"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConformance_EncodeQuotingRules_EmptyString(t *testing.T) {
	out, err := Marshal(map[string]any{"x": ""})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestConformance_EncodeQuotingRules_LeadingTrailingWhitespace(t *testing.T) {
	out, err := Marshal(map[string]any{"x": " a"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \" a\"" {
		t.Fatalf("unexpected: %q", string(out))
	}

	out, err = Marshal(map[string]any{"x": "a "})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"a \"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestConformance_EncodeQuotingRules_ReservedLiterals(t *testing.T) {
	out, err := Marshal(map[string]any{"t": "true", "f": "false", "n": "null"})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("unexpected: %q", string(out))
	}
	for _, line := range lines {
		if !strings.Contains(line, ": \"") {
			t.Fatalf("expected quoted reserved literal, got %q", line)
		}
	}
}

func TestConformance_EncodeQuotingRules_NumericLikeStrings(t *testing.T) {
	out, err := Marshal(map[string]any{"x": "1e-6"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"1e-6\"" {
		t.Fatalf("unexpected: %q", string(out))
	}

	out, err = Marshal(map[string]any{"x": "05"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"05\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestConformance_EncodeQuotingRules_ContainsColon(t *testing.T) {
	out, err := Marshal(map[string]any{"x": "a:b"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"a:b\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestConformance_EncodeQuotingRules_LeadingHyphen(t *testing.T) {
	out, err := Marshal(map[string]any{"x": "-abc"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"-abc\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestConformance_EncodeDocDelimiterAffectsQuotingInObjectFields(t *testing.T) {
	out, err := Marshal(map[string]any{"x": "a|b"}, WithDocDelimiter(Pipe))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: \"a|b\"" {
		t.Fatalf("unexpected: %q", string(out))
	}

	out, err = Marshal(map[string]any{"x": "a|b"}, WithDocDelimiter(Comma))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x: a|b" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestConformance_EncodeActiveDelimiterAffectsQuotingInInlineArray(t *testing.T) {
	out, err := Marshal(map[string]any{"x": []any{"a|b"}}, WithArrayDelimiter(Pipe))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x[1|]: \"a|b\"" {
		t.Fatalf("unexpected: %q", string(out))
	}
}

func TestConformance_EncodeNoTrailingNewlineOrTrailingSpaces(t *testing.T) {
	out, err := Marshal(map[string]any{"a": 1, "b": "x"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.HasSuffix(s, "\n") {
		t.Fatal("must not end with newline")
	}
	for _, line := range strings.Split(s, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Fatalf("trailing space in %q", line)
		}
	}
}

func TestConformance_EncodeFloatNaNAndInfBecomeNull(t *testing.T) {
	out, err := Marshal(map[string]any{"a": math.NaN(), "b": math.Inf(1), "c": math.Inf(-1)})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "a: null") || !strings.Contains(s, "b: null") || !strings.Contains(s, "c: null") {
		t.Fatalf("unexpected: %q", s)
	}
}
