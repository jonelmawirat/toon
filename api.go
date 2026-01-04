package toon

import (
	"bytes"
	"io"
)

func NewEncoder(w io.Writer, opts ...EncoderOption) (*Encoder, error) {
	e := &Encoder{
		w:            w,
		indent:       2,
		docDelimiter: Comma,
		arrDelimiter: Comma,
		keyFolding:   KeyFoldingOff,
		flattenDepth: int(^uint(0) >> 1),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(e); err != nil {
			return nil, err
		}
	}
	return e, nil
}

func Marshal(v any, opts ...EncoderOption) ([]byte, error) {
	var buf bytes.Buffer
	e, err := NewEncoder(&buf, opts...)
	if err != nil {
		return nil, err
	}
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *Encoder) Encode(v any) error {
	val, err := normalize(v)
	if err != nil {
		return err
	}
	if e.keyFolding == KeyFoldingSafe {
		fd := e.flattenDepth
		if fd < 0 {
			fd = 0
		}
		val = foldValueSafe(val, fd)
	}
	lines, err := encodeDocument(val, e.indent, e.docDelimiter, e.arrDelimiter)
	if err != nil {
		return err
	}
	_, err = io.WriteString(e.w, lines)
	return err
}

func NewDecoder(r io.Reader, opts ...DecoderOption) (*Decoder, error) {
	if r == nil {
		return nil, &Error{Message: "nil reader"}
	}
	d := &Decoder{
		r:           r,
		indent:      2,
		strict:      true,
		expandPaths: ExpandPathsOff,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func NewDecoderBytes(data []byte, opts ...DecoderOption) (*Decoder, error) {
	d := &Decoder{
		data:        data,
		fromBytes:   true,
		indent:      2,
		strict:      true,
		expandPaths: ExpandPathsOff,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (d *Decoder) Reset(r io.Reader) error {
	if r == nil {
		return &Error{Message: "nil reader"}
	}
	d.r = r
	d.data = nil
	d.fromBytes = false
	return nil
}

func (d *Decoder) ResetBytes(data []byte) {
	d.r = nil
	d.data = data
	d.fromBytes = true
}

func Unmarshal(data []byte, opts ...DecoderOption) (Value, error) {
	d := Decoder{
		data:        data,
		fromBytes:   true,
		indent:      2,
		strict:      true,
		expandPaths: ExpandPathsOff,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&d); err != nil {
			return nil, err
		}
	}
	return decodeDocumentBytes(d.data, d.indent, d.strict, d.expandPaths)
}

func (d *Decoder) Decode() (Value, error) {
	if d.fromBytes {
		return decodeDocumentBytes(d.data, d.indent, d.strict, d.expandPaths)
	}
	return decodeDocumentReader(d.r, d.indent, d.strict, d.expandPaths)
}
