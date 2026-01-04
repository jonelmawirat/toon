package toon

import (
	"bytes"
	"io"
	"strings"
)

func readAllOptimized(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, &Error{Message: "nil reader"}
	}

	switch t := r.(type) {
	case *bytes.Reader:
		n := t.Len()
		if n == 0 {
			return []byte{}, nil
		}
		b := make([]byte, n)
		_, err := io.ReadFull(t, b)
		if err != nil {
			return nil, err
		}
		return b, nil
	case *strings.Reader:
		n := t.Len()
		if n == 0 {
			return []byte{}, nil
		}
		b := make([]byte, n)
		_, err := io.ReadFull(t, b)
		if err != nil {
			return nil, err
		}
		return b, nil
	case *bytes.Buffer:
		n := t.Len()
		if n == 0 {
			return []byte{}, nil
		}
		return t.Next(n), nil
	default:
		return io.ReadAll(r)
	}
}
