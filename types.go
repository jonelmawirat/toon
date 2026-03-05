package toon

import (
	"fmt"
	"io"
)

// Value is any TOON value.
type Value any

// Number is a canonical TOON numeric literal stored as decimal text.
type Number string

// Array is a TOON array value.
type Array []Value

// Member is a key/value pair in an Object.
type Member struct {
	// Key is the object member key.
	Key string
	// Value is the object member value.
	Value  Value
	quoted bool
}

// Object is an ordered TOON object.
type Object struct {
	Members []Member
}

type tabularSchema struct {
	keys       []string
	fieldCount int
	kinds      []uint8
	texts      []string
}

type tabularSchema2 struct {
	key0  string
	key1  string
	kinds []uint8
	texts []string
}

type tabularRow struct {
	schema *tabularSchema
	base   int
}

type tabularRow2 struct {
	schema *tabularSchema2
	base   int
}

type tabularArray struct {
	schema *tabularSchema
	len    int
}

type tabularArray2 struct {
	schema *tabularSchema2
	len    int
}

func materializeTabularMembers(schema *tabularSchema, base int) []Member {
	if schema == nil || schema.fieldCount == 0 {
		return nil
	}
	n := schema.fieldCount
	members := make([]Member, n)
	for i := 0; i < n; i++ {
		idx := base + i
		members[i] = Member{Key: schema.keys[i], Value: packedCellToValue(schema.kinds[idx], schema.texts[idx])}
	}
	return members
}

func materializeTabularMembers2(schema *tabularSchema2, base int) []Member {
	if schema == nil {
		return nil
	}
	return []Member{
		{Key: schema.key0, Value: packedCellToValue(schema.kinds[base], schema.texts[base])},
		{Key: schema.key1, Value: packedCellToValue(schema.kinds[base+1], schema.texts[base+1])},
	}
}

func materializeTabularArrayRows(schema *tabularSchema, length int) Array {
	if length <= 0 {
		return Array{}
	}
	rows := make([]tabularRow, length)
	out := make(Array, length)
	stride := schema.fieldCount
	for i := 0; i < length; i++ {
		rows[i] = tabularRow{schema: schema, base: i * stride}
		out[i] = &rows[i]
	}
	return out
}

func materializeTabularArrayRows2(schema *tabularSchema2, length int) Array {
	if length <= 0 {
		return Array{}
	}
	rows := make([]tabularRow2, length)
	out := make(Array, length)
	for i := 0; i < length; i++ {
		rows[i] = tabularRow2{schema: schema, base: i * 2}
		out[i] = &rows[i]
	}
	return out
}

func objectFromValue(v Value) (Object, bool) {
	switch t := v.(type) {
	case Object:
		return t, true
	case *Object:
		if t == nil {
			return Object{}, false
		}
		return *t, true
	case tabularRow:
		return Object{Members: materializeTabularMembers(t.schema, t.base)}, true
	case *tabularRow:
		if t == nil {
			return Object{}, false
		}
		return Object{Members: materializeTabularMembers(t.schema, t.base)}, true
	case tabularRow2:
		return Object{Members: materializeTabularMembers2(t.schema, t.base)}, true
	case *tabularRow2:
		if t == nil {
			return Object{}, false
		}
		return Object{Members: materializeTabularMembers2(t.schema, t.base)}, true
	default:
		return Object{}, false
	}
}

// AsObject converts v to Object when possible.
func AsObject(v Value) (Object, bool) {
	return objectFromValue(v)
}

func arrayFromValue(v Value) (Array, bool) {
	switch t := v.(type) {
	case Array:
		return t, true
	case *tabularArray:
		if t == nil || t.schema == nil {
			return Array{}, false
		}
		return materializeTabularArrayRows(t.schema, t.len), true
	case tabularArray:
		if t.schema == nil {
			return Array{}, false
		}
		return materializeTabularArrayRows(t.schema, t.len), true
	case *tabularArray2:
		if t == nil || t.schema == nil {
			return Array{}, false
		}
		return materializeTabularArrayRows2(t.schema, t.len), true
	case tabularArray2:
		if t.schema == nil {
			return Array{}, false
		}
		return materializeTabularArrayRows2(t.schema, t.len), true
	default:
		return nil, false
	}
}

// AsArray converts v to Array when possible.
func AsArray(v Value) (Array, bool) {
	return arrayFromValue(v)
}

// Len returns the number of members in the object.
func (o Object) Len() int {
	return len(o.Members)
}

// Get returns a value by key.
func (o Object) Get(key string) (Value, bool) {
	for _, m := range o.Members {
		if m.Key == key {
			return unboxValue(m.Value), true
		}
	}
	return nil, false
}

// Set inserts or replaces a key/value pair.
func (o *Object) Set(key string, v Value) {
	o.setWithQuoted(key, v, false)
}

func (o *Object) setWithQuoted(key string, v Value, quoted bool) {
	if cap(o.Members) == 0 {
		o.Members = make([]Member, 0, 2)
	}
	for i := range o.Members {
		if o.Members[i].Key == key {
			o.Members = append(o.Members[:i], o.Members[i+1:]...)
			break
		}
	}
	o.Members = append(o.Members, Member{Key: key, Value: v, quoted: quoted})
}

// Delete removes a key and reports whether it was present.
func (o *Object) Delete(key string) bool {
	for i := range o.Members {
		if o.Members[i].Key == key {
			o.Members = append(o.Members[:i], o.Members[i+1:]...)
			return true
		}
	}
	return false
}

// Delimiter configures document/array delimiter behavior.
type Delimiter byte

const (
	// Comma configures comma-separated inline and tabular segments.
	Comma Delimiter = ','
	// Tab configures tab-separated inline and tabular segments.
	Tab Delimiter = '\t'
	// Pipe configures pipe-separated inline and tabular segments.
	Pipe Delimiter = '|'
)

// Valid reports whether d is a supported delimiter.
func (d Delimiter) Valid() bool {
	return d == Comma || d == Tab || d == Pipe
}

// KeyFoldingMode controls safe dotted-key folding during encoding.
type KeyFoldingMode string

const (
	// KeyFoldingOff leaves object keys unchanged during encoding.
	KeyFoldingOff KeyFoldingMode = "off"
	// KeyFoldingSafe folds dotted keys into nested objects when safe.
	KeyFoldingSafe KeyFoldingMode = "safe"
)

// ExpandPathsMode controls safe dotted-path expansion during decoding.
type ExpandPathsMode string

const (
	// ExpandPathsOff leaves dotted keys unchanged during decoding.
	ExpandPathsOff ExpandPathsMode = "off"
	// ExpandPathsSafe expands dotted keys into nested objects when safe.
	ExpandPathsSafe ExpandPathsMode = "safe"
)

// Error is a TOON parsing/encoding error with optional line and column.
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

// EncoderOption configures an Encoder.
type EncoderOption func(*Encoder) error

// DecoderOption configures a Decoder.
type DecoderOption func(*Decoder) error

// Encoder writes TOON documents.
type Encoder struct {
	w            io.Writer
	indent       int
	docDelimiter Delimiter
	arrDelimiter Delimiter
	keyFolding   KeyFoldingMode
	flattenDepth int
}

// Decoder reads TOON documents.
type Decoder struct {
	r           io.Reader
	data        []byte
	fromBytes   bool
	indent      int
	strict      bool
	expandPaths ExpandPathsMode
}
