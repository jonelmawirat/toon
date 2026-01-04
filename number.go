package toon

import (
	"strconv"
	"strings"
)

func looksForbiddenLeadingZeroDecimal(s string) bool {
	if len(s) >= 2 && s[0] == '0' && s[1] >= '0' && s[1] <= '9' {
		return true
	}
	return false
}

func isForbiddenLeadingZeroNumberToken(s string) bool {
	if len(s) == 0 {
		return false
	}
	i := 0
	if s[0] == '-' {
		i++
		if i >= len(s) {
			return false
		}
	}
	j := i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	if j-i <= 1 {
		return false
	}
	if s[i] != '0' {
		return false
	}
	if j < len(s) && s[j] == '.' {
		return false
	}
	if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
		return false
	}
	return true
}

func looksNumericLike(s string) bool {
	_, ok := parseNumberTokenToCanonical(s)
	return ok
}

func parseNumberTokenToCanonical(s string) (string, bool) {
	if len(s) == 0 {
		return "", false
	}

	i := 0
	neg := false
	if s[i] == '-' {
		neg = true
		i++
		if i >= len(s) {
			return "", false
		}
	}

	intStart := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == intStart {
		return "", false
	}
	intEnd := i

	fracStart := 0
	fracEnd := 0
	hasDot := false
	if i < len(s) && s[i] == '.' {
		hasDot = true
		i++
		fracStart = i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		fracEnd = i
	}

	exp := 0
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i >= len(s) {
			return "", false
		}
		expSign := 1
		if s[i] == '+' || s[i] == '-' {
			if s[i] == '-' {
				expSign = -1
			}
			i++
			if i >= len(s) {
				return "", false
			}
		}
		expStart := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == expStart {
			return "", false
		}
		e64, err := strconv.ParseInt(s[expStart:i], 10, 32)
		if err != nil {
			return "", false
		}
		exp = int(e64) * expSign
	}

	if i != len(s) {
		return "", false
	}

	intPart := s[intStart:intEnd]
	fracPart := ""
	if hasDot {
		fracPart = s[fracStart:fracEnd]
	}

	nonZero := false
	for j := 0; j < len(intPart); j++ {
		if intPart[j] != '0' {
			nonZero = true
			break
		}
	}
	if !nonZero {
		for j := 0; j < len(fracPart); j++ {
			if fracPart[j] != '0' {
				nonZero = true
				break
			}
		}
	}
	if !nonZero {
		return "0", true
	}

	if exp == 0 {
		if !hasDot {
			start := intStart
			for start < intEnd-1 && s[start] == '0' {
				start++
			}
			if !neg {
				return s[start:intEnd], true
			}
			if start == intStart {
				return s[:intEnd], true
			}
			var b strings.Builder
			b.Grow(1 + (intEnd - start))
			b.WriteByte('-')
			b.WriteString(s[start:intEnd])
			return b.String(), true
		}

		fracTrimEnd := fracEnd
		for fracTrimEnd > fracStart && s[fracTrimEnd-1] == '0' {
			fracTrimEnd--
		}

		intTrimStart := intStart
		for intTrimStart < intEnd-1 && s[intTrimStart] == '0' {
			intTrimStart++
		}

		if fracTrimEnd == fracStart {
			if !neg {
				return s[intTrimStart:intEnd], true
			}
			if intTrimStart == intStart {
				return s[:intEnd], true
			}
			var b strings.Builder
			b.Grow(1 + (intEnd - intTrimStart))
			b.WriteByte('-')
			b.WriteString(s[intTrimStart:intEnd])
			return b.String(), true
		}

		if !neg && intTrimStart == intStart && fracTrimEnd == fracEnd {
			return s, true
		}

		if !neg {
			return s[intTrimStart:fracTrimEnd], true
		}

		if intTrimStart == intStart && fracTrimEnd == fracEnd {
			return s, true
		}

		var b strings.Builder
		b.Grow(1 + (intEnd - intTrimStart) + 1 + (fracTrimEnd - fracStart))
		b.WriteByte('-')
		b.WriteString(s[intTrimStart:intEnd])
		b.WriteByte('.')
		b.WriteString(s[fracStart:fracTrimEnd])
		return b.String(), true
	}

	digitsLen := len(intPart) + len(fracPart)
	newPoint := len(intPart) + exp

	digitAt := func(idx int) byte {
		if idx < len(intPart) {
			return intPart[idx]
		}
		return fracPart[idx-len(intPart)]
	}

	if newPoint <= 0 {
		zeros := -newPoint
		trimmedDigitsLen := digitsLen
		for trimmedDigitsLen > 0 && digitAt(trimmedDigitsLen-1) == '0' {
			trimmedDigitsLen--
		}

		var b strings.Builder
		capEst := 1 + 1 + 1 + zeros + trimmedDigitsLen
		if neg {
			capEst++
		}
		b.Grow(capEst)
		if neg {
			b.WriteByte('-')
		}
		b.WriteByte('0')
		b.WriteByte('.')
		for i := 0; i < zeros; i++ {
			b.WriteByte('0')
		}
		for i := 0; i < trimmedDigitsLen; i++ {
			b.WriteByte(digitAt(i))
		}
		return b.String(), true
	}

	if newPoint >= digitsLen {
		zeros := newPoint - digitsLen
		intDigitsStart := 0
		for intDigitsStart < digitsLen-1 && digitAt(intDigitsStart) == '0' {
			intDigitsStart++
		}
		intDigitsLen := digitsLen - intDigitsStart

		var b strings.Builder
		capEst := intDigitsLen + zeros
		if neg {
			capEst++
		}
		b.Grow(capEst)
		if neg {
			b.WriteByte('-')
		}
		for i := intDigitsStart; i < digitsLen; i++ {
			b.WriteByte(digitAt(i))
		}
		for i := 0; i < zeros; i++ {
			b.WriteByte('0')
		}
		return b.String(), true
	}

	fracTrimEnd := digitsLen
	for fracTrimEnd > newPoint && digitAt(fracTrimEnd-1) == '0' {
		fracTrimEnd--
	}
	fracLen := fracTrimEnd - newPoint

	intDigitsStart := 0
	for intDigitsStart < newPoint-1 && digitAt(intDigitsStart) == '0' {
		intDigitsStart++
	}
	if intDigitsStart >= newPoint {
		intDigitsStart = newPoint - 1
	}
	intDigitsLen := newPoint - intDigitsStart

	if fracLen == 0 {
		var b strings.Builder
		capEst := intDigitsLen
		if neg {
			capEst++
		}
		b.Grow(capEst)
		if neg {
			b.WriteByte('-')
		}
		for i := intDigitsStart; i < newPoint; i++ {
			b.WriteByte(digitAt(i))
		}
		return b.String(), true
	}

	var b strings.Builder
	capEst := intDigitsLen + 1 + fracLen
	if neg {
		capEst++
	}
	b.Grow(capEst)
	if neg {
		b.WriteByte('-')
	}
	for i := intDigitsStart; i < newPoint; i++ {
		b.WriteByte(digitAt(i))
	}
	b.WriteByte('.')
	for i := newPoint; i < fracTrimEnd; i++ {
		b.WriteByte(digitAt(i))
	}
	return b.String(), true
}

func canonicalizeNumberToken(s string) (Number, bool) {
	c, ok := parseNumberTokenToCanonical(s)
	if !ok {
		return "", false
	}
	if c == "0" {
		return Number("0"), true
	}
	return Number(c), true
}
