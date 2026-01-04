package toon

func trimSpaces(s string) string {
	i := 0
	j := len(s)
	for i < j && s[i] == ' ' {
		i++
	}
	for j > i && s[j-1] == ' ' {
		j--
	}
	return s[i:j]
}

func firstUnquotedIndex(s string, target byte) int {
	inQuotes := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuotes {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inQuotes = false
			}
			continue
		}
		if c == '"' {
			inQuotes = true
			continue
		}
		if c == target {
			return i
		}
	}
	return -1
}

func firstUnquotedAny(s string, targets ...byte) (int, byte) {
	inQuotes := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuotes {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inQuotes = false
			}
			continue
		}
		if c == '"' {
			inQuotes = true
			continue
		}
		for _, t := range targets {
			if c == t {
				return i, t
			}
		}
	}
	return -1, 0
}

func forEachDelimited(s string, delim byte, fn func(tok string) error) error {
	if len(s) == 0 {
		return nil
	}

	inQuotes := false
	esc := false
	start := 0

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuotes {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inQuotes = false
			}
			continue
		}
		if c == '"' {
			inQuotes = true
			continue
		}
		if c == delim {
			if err := fn(trimSpaces(s[start:i])); err != nil {
				return err
			}
			start = i + 1
		}
	}

	return fn(trimSpaces(s[start:]))
}

func splitDelimited(s string, delim byte) []string {
	if len(s) == 0 {
		return nil
	}
	tokens := make([]string, 0, 8)
	_ = forEachDelimited(s, delim, func(tok string) error {
		tokens = append(tokens, tok)
		return nil
	})
	return tokens
}

type delimitedScanner struct {
	s        string
	delim    byte
	i        int
	start    int
	inQuotes bool
	esc      bool
	done     bool
}

func newDelimitedScanner(s string, delim byte) delimitedScanner {
	return delimitedScanner{s: s, delim: delim}
}

func (d *delimitedScanner) next() (string, bool) {
	if d.done {
		return "", false
	}
	if len(d.s) == 0 {
		d.done = true
		return "", false
	}

	for d.i < len(d.s) {
		c := d.s[d.i]
		if d.inQuotes {
			if d.esc {
				d.esc = false
				d.i++
				continue
			}
			if c == '\\' {
				d.esc = true
				d.i++
				continue
			}
			if c == '"' {
				d.inQuotes = false
			}
			d.i++
			continue
		}
		if c == '"' {
			d.inQuotes = true
			d.i++
			continue
		}
		if c == d.delim {
			tok := trimSpaces(d.s[d.start:d.i])
			d.i++
			d.start = d.i
			return tok, true
		}
		d.i++
	}

	tok := trimSpaces(d.s[d.start:])
	d.done = true
	return tok, true
}
