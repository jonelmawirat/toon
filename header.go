package toon

import (
	"strconv"
	"strings"
)

type header struct {
	key        string
	length     int
	delimiter  byte
	fields     []string
	inlineTail string
}

func parseKeyToken(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", &Error{Message: "empty key"}
	}
	if s[0] == '"' {
		return parseQuoted(s)
	}
	return s, nil
}

func parseHeaderLine(line string, strict bool, fieldsScratch *[]string) (header, bool, error) {
	colonPos := firstUnquotedIndex(line, ':')
	if colonPos < 0 {
		return header{}, false, nil
	}

	beforeColon := line[:colonPos]
	afterColon := line[colonPos+1:]
	beforeColon = strings.TrimSpace(beforeColon)

	bracketPos := firstUnquotedIndex(beforeColon, '[')
	if bracketPos < 0 {
		return header{}, false, nil
	}

	keyPart := strings.TrimSpace(beforeColon[:bracketPos])
	rest := beforeColon[bracketPos:]
	if len(rest) < 3 || rest[0] != '[' {
		return header{}, false, nil
	}

	closeBracket := firstUnquotedIndex(rest, ']')
	if closeBracket < 0 {
		return header{}, false, nil
	}

	bracketContent := rest[1:closeBracket]
	delim := byte(',')
	if len(bracketContent) > 0 {
		last := bracketContent[len(bracketContent)-1]
		if last == '|' || last == '\t' {
			delim = last
			bracketContent = bracketContent[:len(bracketContent)-1]
		}
	}
	if bracketContent == "" {
		return header{}, false, nil
	}
	for i := 0; i < len(bracketContent); i++ {
		if bracketContent[i] < '0' || bracketContent[i] > '9' {
			return header{}, false, nil
		}
	}
	n64, err := strconv.ParseInt(bracketContent, 10, 0)
	if err != nil || n64 < 0 {
		return header{}, false, &Error{Message: "invalid array length"}
	}

	restAfterBracket := rest[closeBracket+1:]
	fields := []string(nil)

	if len(restAfterBracket) > 0 && restAfterBracket[0] == '{' {
		closeBrace := firstUnquotedIndex(restAfterBracket, '}')
		if closeBrace < 0 {
			return header{}, false, &Error{Message: "unterminated fields segment"}
		}
		fieldsContent := restAfterBracket[1:closeBrace]
		if strict {
			if delim == ',' {
				if firstUnquotedIndex(fieldsContent, '\t') >= 0 || firstUnquotedIndex(fieldsContent, '|') >= 0 {
					return header{}, false, &Error{Message: "delimiter mismatch between bracket and fields segment"}
				}
			} else {
				if firstUnquotedIndex(fieldsContent, ',') >= 0 {
					return header{}, false, &Error{Message: "delimiter mismatch between bracket and fields segment"}
				}
			}
		}

		var fs []string
		if fieldsScratch != nil {
			fs = (*fieldsScratch)[:0]
		} else {
			fs = make([]string, 0, 4)
		}

		it := newDelimitedScanner(fieldsContent, delim)
		for {
			ft, ok := it.next()
			if !ok {
				break
			}
			k, err := parseKeyToken(ft)
			if err != nil {
				return header{}, false, err
			}
			fs = append(fs, k)
		}
		if len(fs) == 0 {
			return header{}, false, &Error{Message: "empty field list"}
		}
		fields = fs
		if fieldsScratch != nil {
			*fieldsScratch = fs
		}
		restAfterBracket = restAfterBracket[closeBrace+1:]
	}

	if strings.TrimSpace(restAfterBracket) != "" {
		return header{}, false, nil
	}

	key := ""
	if keyPart != "" {
		k, err := parseKeyToken(keyPart)
		if err != nil {
			return header{}, false, err
		}
		key = k
	}

	inline := strings.TrimLeft(afterColon, " ")
	return header{
		key:        key,
		length:     int(n64),
		delimiter:  delim,
		fields:     fields,
		inlineTail: inline,
	}, true, nil
}

func looksLikeRootArrayHeaderLine(line string) bool {
	colonPos := firstUnquotedIndex(line, ':')
	if colonPos < 0 {
		return false
	}

	beforeColon := strings.TrimSpace(line[:colonPos])
	if beforeColon == "" {
		return false
	}

	bracketPos := firstUnquotedIndex(beforeColon, '[')
	if bracketPos != 0 {
		return false
	}

	rest := beforeColon[bracketPos:]
	if len(rest) < 3 || rest[0] != '[' {
		return false
	}

	closeBracket := firstUnquotedIndex(rest, ']')
	if closeBracket < 0 {
		return false
	}

	bracketContent := rest[1:closeBracket]
	if bracketContent == "" {
		return false
	}

	last := bracketContent[len(bracketContent)-1]
	if last == '|' || last == '\t' {
		bracketContent = bracketContent[:len(bracketContent)-1]
		if bracketContent == "" {
			return false
		}
	}

	for i := 0; i < len(bracketContent); i++ {
		c := bracketContent[i]
		if c < '0' || c > '9' {
			return false
		}
	}

	restAfterBracket := rest[closeBracket+1:]
	if len(restAfterBracket) > 0 && restAfterBracket[0] == '{' {
		closeBrace := firstUnquotedIndex(restAfterBracket, '}')
		if closeBrace < 0 {
			return false
		}
		restAfterBracket = restAfterBracket[closeBrace+1:]
	}

	return strings.TrimSpace(restAfterBracket) == ""
}
