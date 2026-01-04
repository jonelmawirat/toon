package toon

type scannedLine struct {
	content string
	depth   int32
	blank   bool
}

func scanString(s string, indent int, strict bool, dst []scannedLine) ([]scannedLine, error) {
	lineNo := 1
	start := 0
	for start <= len(s) {
		end := start
		for end < len(s) && s[end] != '\n' {
			end++
		}

		lineEnd := end
		if lineEnd > start && s[lineEnd-1] == '\r' {
			lineEnd--
			if lineEnd > start && s[lineEnd-1] == '\r' {
				lineEnd--
			}
		}

		spaces := 0
		j := start
		for j < lineEnd {
			if s[j] == ' ' {
				spaces++
				j++
				continue
			}
			if s[j] == '\t' {
				if strict {
					return nil, &Error{Line: lineNo, Column: (j - start) + 1, Message: "tabs are not allowed in indentation"}
				}
				spaces += indent
				j++
				continue
			}
			break
		}

		if strict && indent > 0 && spaces%indent != 0 {
			return nil, &Error{Line: lineNo, Column: 1, Message: "indentation must be an exact multiple of indent size"}
		}

		depth := 0
		if indent > 0 {
			depth = spaces / indent
		}

		content := s[j:lineEnd]
		blank := true
		for k := 0; k < len(content); k++ {
			if content[k] != ' ' && content[k] != '\t' {
				blank = false
				break
			}
		}

		dst = append(dst, scannedLine{
			content: content,
			depth:   int32(depth),
			blank:   blank,
		})

		if end == len(s) {
			break
		}
		start = end + 1
		lineNo++
	}

	return dst, nil
}
