package toon

func WithEncoderIndent(n int) EncoderOption {
	return func(e *Encoder) error {
		if n <= 0 {
			return &Error{Message: "indent must be > 0"}
		}
		e.indent = n
		return nil
	}
}

func WithDecoderIndent(n int) DecoderOption {
	return func(d *Decoder) error {
		if n <= 0 {
			return &Error{Message: "indent must be > 0"}
		}
		d.indent = n
		return nil
	}
}

func WithDocDelimiter(d Delimiter) EncoderOption {
	return func(e *Encoder) error {
		if !d.Valid() {
			return &Error{Message: "invalid document delimiter"}
		}
		e.docDelimiter = d
		return nil
	}
}

func WithArrayDelimiter(d Delimiter) EncoderOption {
	return func(e *Encoder) error {
		if !d.Valid() {
			return &Error{Message: "invalid array delimiter"}
		}
		e.arrDelimiter = d
		return nil
	}
}

func WithKeyFolding(mode KeyFoldingMode) EncoderOption {
	return func(e *Encoder) error {
		if mode != KeyFoldingOff && mode != KeyFoldingSafe {
			return &Error{Message: "invalid key folding mode"}
		}
		e.keyFolding = mode
		return nil
	}
}

func WithFlattenDepth(n int) EncoderOption {
	return func(e *Encoder) error {
		if n < 0 {
			return &Error{Message: "flattenDepth must be >= 0"}
		}
		e.flattenDepth = n
		return nil
	}
}

func WithStrict(v bool) DecoderOption {
	return func(d *Decoder) error {
		d.strict = v
		return nil
	}
}

func WithExpandPaths(mode ExpandPathsMode) DecoderOption {
	return func(d *Decoder) error {
		if mode != ExpandPathsOff && mode != ExpandPathsSafe {
			return &Error{Message: "invalid expandPaths mode"}
		}
		d.expandPaths = mode
		return nil
	}
}
