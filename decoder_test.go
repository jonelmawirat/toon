package toon

import (
	"reflect"
	"testing"
)

func TestDecodeTabular(t *testing.T) {
	input := "users[2]{id,name,role}:\n  1,Alice,admin\n  2,Bob,user"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	usersV, _ := obj.Get("users")
	users := mustArray(t, usersV)
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	u0 := mustObject(t, users[0])
	if got, _ := u0.Get("name"); got != "Alice" {
		t.Fatalf("unexpected name: %#v", got)
	}
}

func TestDecodeRootPrimitive(t *testing.T) {
	input := "hello"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if v != "hello" {
		t.Fatalf("expected hello, got %#v", v)
	}
}

func TestDecodeForbiddenLeadingZerosNumberAsString(t *testing.T) {
	input := "x: 05"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	x, _ := obj.Get("x")
	if x != "05" {
		t.Fatalf("expected string 05, got %#v", x)
	}
}

func TestDecodeExponentNumberCanonicalized(t *testing.T) {
	input := "x: 1e-6"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	x, _ := obj.Get("x")
	if x != Number("0.000001") {
		t.Fatalf("expected 0.000001, got %#v", x)
	}
}

func TestDecodeStrictCountMismatch(t *testing.T) {
	input := "tags[5]: a,b,c"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeStrictInlineHeaderRequiresSingleSpaceAfterColon(t *testing.T) {
	cases := []string{
		"tags[2]:a,b",
		"tags[2]:  a,b",
		"tags[2]: \ta,b",
	}

	for _, input := range cases {
		if _, err := Unmarshal([]byte(input)); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestDecodeNonStrictInlineHeaderAllowsLenientSpacing(t *testing.T) {
	input := "tags[2]:a,b"
	v, err := Unmarshal([]byte(input), WithStrict(false))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	tagsV, ok := obj.Get("tags")
	if !ok {
		t.Fatal("missing tags")
	}
	tags := mustArray(t, tagsV)
	if len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

func TestDecodeInvalidEscape(t *testing.T) {
	input := "name: \"bad\\xesc\""
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeIndentationError(t *testing.T) {
	input := "items[1]:\n   - value"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeBlankLineInsideArrayStrict(t *testing.T) {
	input := "items[2]:\n  - 1\n\n  - 2"
	_, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeListItemWithTabularFirstField(t *testing.T) {
	input := "items[1]:\n  - users[2]{id,name}:\n      1,Ada\n      2,Bob\n    status: active"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	itemsV, _ := obj.Get("items")
	items := mustArray(t, itemsV)
	item0 := mustObject(t, items[0])
	usersV, _ := item0.Get("users")
	users := mustArray(t, usersV)
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	status, _ := item0.Get("status")
	if status != "active" {
		t.Fatalf("expected active, got %#v", status)
	}
}

func TestExpandPathsSafe(t *testing.T) {
	input := "a.b.c: 1\na.b.d: 2\na.e: 3"
	v, err := Unmarshal([]byte(input), WithExpandPaths(ExpandPathsSafe))
	if err != nil {
		t.Fatal(err)
	}
	want := Object{Members: []Member{
		{Key: "a", Value: Object{Members: []Member{
			{Key: "b", Value: Object{Members: []Member{
				{Key: "c", Value: Number("1")},
				{Key: "d", Value: Number("2")},
			}}},
			{Key: "e", Value: Number("3")},
		}}},
	}}
	if !reflect.DeepEqual(v, want) {
		t.Fatalf("mismatch\ngot:  %#v\nwant: %#v", v, want)
	}
}

func TestExpandPathsConflictStrict(t *testing.T) {
	input := "a.b: 1\na: 2"
	_, err := Unmarshal([]byte(input), WithExpandPaths(ExpandPathsSafe))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExpandPathsConflictNonStrictLWW(t *testing.T) {
	input := "a.b: 1\na: 2"
	v, err := Unmarshal([]byte(input), WithExpandPaths(ExpandPathsSafe), WithStrict(false))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	a, _ := obj.Get("a")
	if a != Number("2") {
		t.Fatalf("expected 2, got %#v", a)
	}
}

func TestDecodeHeaderAllowsWhitespaceBetweenBracketAndFields(t *testing.T) {
	input := "users[2] {id,name}:\n  1,Alice\n  2,Bob"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	usersV, ok := obj.Get("users")
	if !ok {
		t.Fatal("missing users key")
	}
	users := mustArray(t, usersV)
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	u0 := mustObject(t, users[0])
	id, _ := u0.Get("id")
	name, _ := u0.Get("name")
	if id != Number("1") || name != "Alice" {
		t.Fatalf("unexpected first row: %#v", u0)
	}
}

func TestDecodeRootHeaderAllowsWhitespaceBetweenBracketAndFields(t *testing.T) {
	input := "[2] {id,name}:\n  1,Alice\n  2,Bob"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	arr := mustArray(t, v)
	if len(arr) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(arr))
	}
	r0 := mustObject(t, arr[0])
	id, _ := r0.Get("id")
	name, _ := r0.Get("name")
	if id != Number("1") || name != "Alice" {
		t.Fatalf("unexpected first row: %#v", r0)
	}
}

func TestExpandPathsSafePreservesEncounterOrderOnDeepMerge(t *testing.T) {
	input := "a.b.c: 1\na.d: 2\na.b.e: 3"
	v, err := Unmarshal([]byte(input), WithExpandPaths(ExpandPathsSafe))
	if err != nil {
		t.Fatal(err)
	}

	want := Object{Members: []Member{
		{Key: "a", Value: Object{Members: []Member{
			{Key: "b", Value: Object{Members: []Member{
				{Key: "c", Value: Number("1")},
				{Key: "e", Value: Number("3")},
			}}},
			{Key: "d", Value: Number("2")},
		}}},
	}}

	if !reflect.DeepEqual(v, want) {
		t.Fatalf("mismatch\ngot:  %#v\nwant: %#v", v, want)
	}
}

func TestDecodeQuotedKeyInlineArrayHeader(t *testing.T) {
	input := "\"my-key\"[2]: a,b"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	got, ok := obj.Get("my-key")
	if !ok {
		t.Fatal("missing key")
	}
	arr := mustArray(t, got)
	if len(arr) != 2 || arr[0] != "a" || arr[1] != "b" {
		t.Fatalf("unexpected array: %#v", arr)
	}
}

func TestDecodeQuotedEmptyKeyInlineArrayHeader(t *testing.T) {
	input := "\"\"[2]: a,b"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	got, ok := obj.Get("")
	if !ok {
		t.Fatal("missing empty key")
	}
	arr := mustArray(t, got)
	if len(arr) != 2 || arr[0] != "a" || arr[1] != "b" {
		t.Fatalf("unexpected array: %#v", arr)
	}
}

func TestDecodeNestedArrayAsListItemValue(t *testing.T) {
	input := "items[3]:\n  - summary\n  - id: 1\n    name: Ada\n  - [2]:\n    - id: 2\n    - status: draft"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	itemsV, ok := obj.Get("items")
	if !ok {
		t.Fatal("missing items")
	}
	items := mustArray(t, itemsV)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	inner := mustArray(t, items[2])
	if len(inner) != 2 {
		t.Fatalf("expected inner len 2, got %d", len(inner))
	}
}

func TestDecodeStrictAllowsWhitespaceOnlyBlankLine(t *testing.T) {
	input := "a: 1\n   \nb: 2"
	v, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	obj := mustObject(t, v)
	if got, _ := obj.Get("a"); got != Number("1") {
		t.Fatalf("unexpected a: %#v", got)
	}
	if got, _ := obj.Get("b"); got != Number("2") {
		t.Fatalf("unexpected b: %#v", got)
	}
}
