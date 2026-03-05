package toon

import (
	"bytes"
	"io"
)

// NewEncoder creates a TOON encoder that writes to w.
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

// Marshal encodes a Go value into TOON bytes.
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

// Encode writes one TOON document to the encoder writer.
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

// NewDecoder creates a decoder that reads TOON from r.
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

// NewDecoderBytes creates a decoder backed by in-memory bytes.
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

// Reset switches the decoder input to r.
func (d *Decoder) Reset(r io.Reader) error {
	if r == nil {
		return &Error{Message: "nil reader"}
	}
	d.r = r
	d.data = nil
	d.fromBytes = false
	return nil
}

// ResetBytes switches the decoder input to in-memory bytes.
func (d *Decoder) ResetBytes(data []byte) {
	d.r = nil
	d.data = data
	d.fromBytes = true
}

// Unmarshal decodes one TOON document from bytes.
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

// Decode decodes one TOON document from the decoder input.
func (d *Decoder) Decode() (Value, error) {
	if d.fromBytes {
		return decodeDocumentBytes(d.data, d.indent, d.strict, d.expandPaths)
	}
	return decodeDocumentReader(d.r, d.indent, d.strict, d.expandPaths)
}
