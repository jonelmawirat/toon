package toon

import (
	"strings"
	"unicode"
)

func isKeyUnquotedSafe(s string) bool {
	if len(s) == 0 {
		return false
	}
	r0 := rune(s[0])
	if !(r0 == '_' || (r0 >= 'A' && r0 <= 'Z') || (r0 >= 'a' && r0 <= 'z')) {
		return false
	}
	for i := 1; i < len(s); i++ {
		r := rune(s[i])
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func isIdentifierSegment(s string) bool {
	if len(s) == 0 {
		return false
	}
	r0 := rune(s[0])
	if !(r0 == '_' || (r0 >= 'A' && r0 <= 'Z') || (r0 >= 'a' && r0 <= 'z')) {
		return false
	}
	for i := 1; i < len(s); i++ {
		r := rune(s[i])
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func parseQuoted(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' {
		return "", &Error{Message: "unterminated string: missing opening quote"}
	}
	out := make([]byte, 0, len(s))
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '"' {
			if i != len(s)-1 {
				return "", &Error{Message: "invalid quoted token: trailing characters after closing quote"}
			}
			return string(out), nil
		}
		if c == '\\' {
			i++
			if i >= len(s) {
				return "", &Error{Message: "unterminated string: missing closing quote"}
			}
			esc := s[i]
			switch esc {
			case '\\':
				out = append(out, '\\')
			case '"':
				out = append(out, '"')
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			default:
				return "", &Error{Message: "invalid escape sequence: \\" + string(esc)}
			}
			i++
			continue
		}
		out = append(out, c)
		i++
	}
	return "", &Error{Message: "unterminated string: missing closing quote"}
}

func escapeQuoted(s string) (string, error) {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			if r < 0x20 {
				return "", &Error{Message: "string contains unsupported control character"}
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String(), nil
}

func needsQuotingForValue(s string, relevantDelim byte, checkDelim bool) bool {
	if len(s) == 0 {
		return true
	}

	var first, last rune
	for i, r := range s {
		if i == 0 {
			first = r
		}
		last = r
	}
	if unicode.IsSpace(first) || unicode.IsSpace(last) {
		return true
	}

	if s == "true" || s == "false" || s == "null" {
		return true
	}

	if looksNumericLike(s) || looksForbiddenLeadingZeroDecimal(s) {
		return true
	}

	if strings.ContainsAny(s, ":\"\\[]{}") {
		return true
	}

	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			return true
		}
	}

	if checkDelim && relevantDelim != 0 {
		if strings.IndexByte(s, relevantDelim) >= 0 {
			return true
		}
	}

	if s == "-" {
		return true
	}
	if len(s) > 0 && s[0] == '-' {
		return true
	}

	return false
}

func encodeKey(key string) (string, error) {
	if isKeyUnquotedSafe(key) {
		return key, nil
	}
	return escapeQuoted(key)
}
