package toon

import (
	"bytes"
	"testing"
)

func BenchmarkDecodeSmall(b *testing.B) {
	input := []byte("users[2]{id,name,active}:\n  1,Alice,true\n  2,Bob,false\ncount: 2")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeSmallMap(b *testing.B) {
	v := map[string]any{
		"users": []any{
			map[string]any{"id": 1, "name": "Alice", "active": true},
			map[string]any{"id": 2, "name": "Bob", "active": false},
		},
		"count": 2,
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(v)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeTabular1000(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteString("items[1000]{id,name}:\n")
	for i := 1; i <= 1000; i++ {
		buf.WriteString("  ")
		buf.WriteString(itoa(i))
		buf.WriteString(",Item")
		buf.WriteString(itoa(i))
		if i != 1000 {
			buf.WriteByte('\n')
		}
	}
	input := buf.Bytes()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
