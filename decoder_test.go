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
	obj := v.(Object)
	usersV, _ := obj.Get("users")
	users := usersV.(Array)
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	u0 := users[0].(Object)
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
	obj := v.(Object)
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
	obj := v.(Object)
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
	obj := v.(Object)
	itemsV, _ := obj.Get("items")
	items := itemsV.(Array)
	item0 := items[0].(Object)
	usersV, _ := item0.Get("users")
	users := usersV.(Array)
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
	obj := v.(Object)
	a, _ := obj.Get("a")
	if a != Number("2") {
		t.Fatalf("expected 2, got %#v", a)
	}
}
