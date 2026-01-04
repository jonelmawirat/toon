package toon

import "strings"

func parsePrimitiveToken(token string) (Value, error) {
	token = trimSpaces(token)
	if token == "" {
		return "", nil
	}

	if token[0] == '"' {
		return parseQuoted(token)
	}

	switch token {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}

	if isForbiddenLeadingZeroNumberToken(token) {
		return token, nil
	}

	if n, ok := canonicalizeNumberToken(token); ok {
		return n, nil
	}

	return token, nil
}

func encodePrimitiveToken(v Value, relevantDelim byte, checkDelim bool) (string, error) {
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
