package toon

import "strings"

type cellKind uint8

const (
	cellKindNull cellKind = iota
	cellKindBool
	cellKindString
	cellKindNumber
)

const (
	packedKindNull uint8 = iota
	packedKindFalse
	packedKindTrue
	packedKindString
	packedKindNumber
)

type cellValue struct {
	kind cellKind
	b    bool
	s    string
}

func (c *cellValue) value() Value {
	if c == nil {
		return nil
	}
	switch c.kind {
	case cellKindNull:
		return nil
	case cellKindBool:
		return c.b
	case cellKindNumber:
		return Number(c.s)
	default:
		return c.s
	}
}

func unboxValue(v Value) Value {
	switch t := v.(type) {
	case *cellValue:
		return t.value()
	default:
		return v
	}
}

// Unbox converts internal decode wrappers to plain exported values.
//
// This is primarily useful when strict tabular decode paths use packed
// internal cells for performance.
func Unbox(v Value) Value {
	return unboxValue(v)
}

func parsePrimitiveToken(token string) (Value, error) {
	return parsePrimitiveTokenTrimmed(trimSpaces(token))
}

func parsePrimitiveTokenTrimmed(token string) (Value, error) {
	if token == "" {
		return "", nil
	}

	c0 := token[0]
	if c0 == '"' {
		return parseQuoted(token)
	}

	switch len(token) {
	case 4:
		if token == "true" {
			return true, nil
		}
		if token == "null" {
			return nil, nil
		}
	case 5:
		if token == "false" {
			return false, nil
		}
	}

	// Most unquoted tokens are plain strings. Fast-reject non-numeric starters
	// before canonical number parsing in hot decode loops.
	if c0 != '-' && (c0 < '0' || c0 > '9') {
		return token, nil
	}

	if n, ok := parseIntegerTokenFast(token); ok {
		return n, nil
	}

	if isForbiddenLeadingZeroNumberToken(token) {
		return token, nil
	}

	if n, ok := canonicalizeNumberToken(token); ok {
		return n, nil
	}

	return token, nil
}

func parseIntegerTokenFast(token string) (Number, bool) {
	if len(token) == 0 {
		return "", false
	}

	i := 0
	neg := false
	if token[0] == '-' {
		neg = true
		i = 1
		if i >= len(token) {
			return "", false
		}
	}

	start := i
	for i < len(token) {
		c := token[i]
		if c < '0' || c > '9' {
			return "", false
		}
		i++
	}
	digits := len(token) - start
	if digits == 0 {
		return "", false
	}

	// Leading-zero multi-digit integers are not numbers per spec.
	if digits > 1 && token[start] == '0' {
		return "", false
	}

	// Canonicalize -0 to 0.
	if neg && digits == 1 && token[start] == '0' {
		return Number("0"), true
	}

	return Number(token), true
}

func parseUnquotedPrimitiveToken(token string) Value {
	var m Member
	assignUnquotedPrimitiveToken(&m, token)
	return m.Value
}

func assignCellValue(cell *cellValue, v Value) bool {
	v = unboxValue(v)
	switch t := v.(type) {
	case nil:
		cell.kind = cellKindNull
		cell.b = false
		cell.s = ""
	case bool:
		cell.kind = cellKindBool
		cell.b = t
		cell.s = ""
	case string:
		cell.kind = cellKindString
		cell.s = t
	case Number:
		cell.kind = cellKindNumber
		cell.s = string(t)
	default:
		return false
	}
	return true
}

func assignPrimitiveCellValue(m *Member, cell *cellValue, v Value) {
	if !assignCellValue(cell, v) {
		m.Value = unboxValue(v)
		return
	}
	m.Value = cell
}

func assignUnquotedCellToken(cell *cellValue, token string) {
	if token == "" {
		cell.kind = cellKindString
		cell.s = ""
		return
	}
	c0 := token[0]

	switch len(token) {
	case 4:
		if token == "true" {
			cell.kind = cellKindBool
			cell.b = true
			cell.s = ""
			return
		}
		if token == "null" {
			cell.kind = cellKindNull
			cell.b = false
			cell.s = ""
			return
		}
	case 5:
		if token == "false" {
			cell.kind = cellKindBool
			cell.b = false
			cell.s = ""
			return
		}
	}

	if c0 != '-' && (c0 < '0' || c0 > '9') {
		cell.kind = cellKindString
		cell.s = token
		return
	}
	if n, ok := parseIntegerTokenFast(token); ok {
		cell.kind = cellKindNumber
		cell.s = string(n)
		return
	}
	if isForbiddenLeadingZeroNumberToken(token) {
		cell.kind = cellKindString
		cell.s = token
		return
	}
	if n, ok := canonicalizeNumberToken(token); ok {
		cell.kind = cellKindNumber
		cell.s = string(n)
		return
	}
	cell.kind = cellKindString
	cell.s = token
}

func packedCellToValue(kind uint8, text string) Value {
	switch kind {
	case packedKindNull:
		return nil
	case packedKindFalse:
		return false
	case packedKindTrue:
		return true
	case packedKindNumber:
		return Number(text)
	default:
		return text
	}
}

func assignPackedCellValue(kind *uint8, text *string, v Value) bool {
	v = unboxValue(v)
	switch t := v.(type) {
	case nil:
		*kind = packedKindNull
		*text = ""
	case bool:
		if t {
			*kind = packedKindTrue
		} else {
			*kind = packedKindFalse
		}
		*text = ""
	case string:
		*kind = packedKindString
		*text = t
	case Number:
		*kind = packedKindNumber
		*text = string(t)
	default:
		return false
	}
	return true
}

func assignUnquotedPackedCellToken(kind *uint8, text *string, token string) {
	if token == "" {
		*kind = packedKindString
		*text = ""
		return
	}

	c0 := token[0]
	if c0 >= '1' && c0 <= '9' {
		for i := 1; i < len(token); i++ {
			c := token[i]
			if c < '0' || c > '9' {
				assignUnquotedPackedCellTokenSlow(kind, text, token)
				return
			}
		}
		*kind = packedKindNumber
		*text = token
		return
	}

	if c0 == '-' && len(token) > 1 {
		c1 := token[1]
		if c1 >= '1' && c1 <= '9' {
			for i := 2; i < len(token); i++ {
				c := token[i]
				if c < '0' || c > '9' {
					assignUnquotedPackedCellTokenSlow(kind, text, token)
					return
				}
			}
			*kind = packedKindNumber
			*text = token
			return
		}
		if c1 == '0' && len(token) == 2 {
			*kind = packedKindNumber
			*text = "0"
			return
		}
	}

	if (c0 < '0' || c0 > '9') && c0 != '-' {
		switch len(token) {
		case 4:
			if token == "true" {
				*kind = packedKindTrue
				*text = ""
				return
			}
			if token == "null" {
				*kind = packedKindNull
				*text = ""
				return
			}
		case 5:
			if token == "false" {
				*kind = packedKindFalse
				*text = ""
				return
			}
		}
		*kind = packedKindString
		*text = token
		return
	}

	assignUnquotedPackedCellTokenSlow(kind, text, token)
}

func assignUnquotedPackedCellTokenSlow(kind *uint8, text *string, token string) {
	if token == "" {
		*kind = packedKindString
		*text = ""
		return
	}
	c0 := token[0]

	switch len(token) {
	case 4:
		if token == "true" {
			*kind = packedKindTrue
			*text = ""
			return
		}
		if token == "null" {
			*kind = packedKindNull
			*text = ""
			return
		}
	case 5:
		if token == "false" {
			*kind = packedKindFalse
			*text = ""
			return
		}
	}

	if c0 >= '0' && c0 <= '9' {
		i := 1
		for i < len(token) {
			c := token[i]
			if c < '0' || c > '9' {
				break
			}
			i++
		}
		if i == len(token) {
			if len(token) > 1 && token[0] == '0' {
				*kind = packedKindString
				*text = token
				return
			}
			*kind = packedKindNumber
			*text = token
			return
		}
	} else if c0 == '-' && len(token) > 1 {
		i := 1
		for i < len(token) {
			c := token[i]
			if c < '0' || c > '9' {
				break
			}
			i++
		}
		if i == len(token) {
			if len(token) == 2 && token[1] == '0' {
				*kind = packedKindNumber
				*text = "0"
				return
			}
			if len(token) > 2 && token[1] == '0' {
				*kind = packedKindString
				*text = token
				return
			}
			*kind = packedKindNumber
			*text = token
			return
		}
	} else {
		*kind = packedKindString
		*text = token
		return
	}
	if isForbiddenLeadingZeroNumberToken(token) {
		*kind = packedKindString
		*text = token
		return
	}
	if n, ok := canonicalizeNumberToken(token); ok {
		*kind = packedKindNumber
		*text = string(n)
		return
	}
	*kind = packedKindString
	*text = token
}

func assignUnquotedPrimitiveTokenCell(m *Member, cell *cellValue, token string) {
	assignUnquotedCellToken(cell, token)
	m.Value = cell
}

func assignUnquotedPrimitiveToken(m *Member, token string) {
	if token == "" {
		m.Value = ""
		return
	}
	c0 := token[0]

	switch len(token) {
	case 4:
		if token == "true" {
			m.Value = true
			return
		}
		if token == "null" {
			m.Value = nil
			return
		}
	case 5:
		if token == "false" {
			m.Value = false
			return
		}
	}

	if c0 != '-' && (c0 < '0' || c0 > '9') {
		m.Value = token
		return
	}
	if n, ok := parseIntegerTokenFast(token); ok {
		m.Value = n
		return
	}
	if isForbiddenLeadingZeroNumberToken(token) {
		m.Value = token
		return
	}
	if n, ok := canonicalizeNumberToken(token); ok {
		m.Value = n
		return
	}
	m.Value = token
}

func encodePrimitiveToken(v Value, relevantDelim byte, checkDelim bool) (string, error) {
	v = unboxValue(v)
	switch t := v.(type) {
	case nil:
		return "null", nil
	case bool:
		if t {
			return "true", nil
		}
		return "false", nil
	case string:
		if needsQuotingForValue(t, relevantDelim, checkDelim) {
			return escapeQuoted(t)
		}
		return t, nil
	case Number:
		c, ok := parseNumberTokenToCanonical(string(t))
		if !ok {
			return "", &Error{Message: "invalid number"}
		}
		return c, nil
	default:
		return "", &Error{Message: "non-primitive value"}
	}
}

func asPrimitive(v Value) (Value, bool) {
	v = unboxValue(v)
	switch v.(type) {
	case nil, bool, string, Number:
		return v, true
	default:
		return nil, false
	}
}

func isAllPrimitives(a Array) bool {
	for _, v := range a {
		if _, ok := asPrimitive(v); !ok {
			return false
		}
	}
	return true
}

func hasDelimiterUnquoted(s string, delim byte) bool {
	return strings.IndexByte(s, delim) >= 0 && firstUnquotedIndex(s, delim) >= 0
}
