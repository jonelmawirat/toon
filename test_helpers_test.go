package toon

import "testing"

func mustObject(t testing.TB, v Value) Object {
	t.Helper()
	obj, ok := objectFromValue(v)
	if !ok {
		t.Fatalf("expected object, got %#v", v)
	}
	return obj
}

func mustArray(t testing.TB, v Value) Array {
	t.Helper()
	arr, ok := arrayFromValue(v)
	if !ok {
		t.Fatalf("expected array, got %#v", v)
	}
	return arr
}
