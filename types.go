package toon

import (
	"fmt"
	"io"
)

type Value any

type Number string

type Array []Value

type Member struct {
	Key   string
	Value Value
}

type Object struct {
	Members []Member
}

func (o Object) Len() int {
	return len(o.Members)
}

func (o Object) Get(key string) (Value, bool) {
	for _, m := range o.Members {
		if m.Key == key {
			return m.Value, true
		}
	}
	return nil, false
}

func (o *Object) Set(key string, v Value) {
	if cap(o.Members) == 0 {
		o.Members = make([]Member, 0, 2)
	}
	for i := range o.Members {
		if o.Members[i].Key == key {
			o.Members = append(o.Members[:i], o.Members[i+1:]...)
			break
		}
	}
	o.Members = append(o.Members, Member{Key: key, Value: v})
}

func (o *Object) Delete(key string) bool {
	for i := range o.Members {
		if o.Members[i].Key == key {
			o.Members = append(o.Members[:i], o.Members[i+1:]...)
			return true
		}
	}
	return false
}

type Delimiter byte

const (
	Comma Delimiter = ','
	Tab   Delimiter = '\t'
	Pipe  Delimiter = '|'
)

func (d Delimiter) Valid() bool {
	return d == Comma || d == Tab || d == Pipe
}

type KeyFoldingMode string

const (
	KeyFoldingOff  KeyFoldingMode = "off"
	KeyFoldingSafe KeyFoldingMode = "safe"
)

type ExpandPathsMode string

const (
	ExpandPathsOff  ExpandPathsMode = "off"
	ExpandPathsSafe ExpandPathsMode = "safe"
)

type Error struct {
	Line    int
	Column  int
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return "toon: error"
	}
	if e.Line > 0 {
		if e.Column > 0 {
			return fmt.Sprintf("toon: line %d:%d: %s", e.Line, e.Column, e.Message)
		}
		return fmt.Sprintf("toon: line %d: %s", e.Line, e.Message)
	}
	return "toon: " + e.Message
}

type EncoderOption func(*Encoder) error

type DecoderOption func(*Decoder) error

type Encoder struct {
	w            io.Writer
	indent       int
	docDelimiter Delimiter
	arrDelimiter Delimiter
	keyFolding   KeyFoldingMode
	flattenDepth int
}

type Decoder struct {
	r           io.Reader
	data        []byte
	fromBytes   bool
	indent      int
	strict      bool
	expandPaths ExpandPathsMode
}
