package toon

import (
	"io"
	"strings"
	"unsafe"
)

type parser struct {
	source           string
	lines            []scannedLine
	pos              int
	strict           bool
	arrayStack       []int32
	arrayStackBuf    [8]int32
	fieldsScratch    []string
	fieldsScratchBuf [16]string
}

func decodeDocumentReader(r io.Reader, indent int, strict bool, expand ExpandPathsMode) (Value, error) {
	data, err := readAllOptimized(r)
	if err != nil {
		return nil, err
	}
	return decodeDocumentBytes(data, indent, strict, expand)
}

func decodeDocumentBytes(data []byte, indent int, strict bool, expand ExpandPathsMode) (Value, error) {
	s := bytesToStringNoCopy(data)
	scanBuf := borrowScannedLines(0)
	lines, err := scanString(s, indent, strict, scanBuf.lines[:0])
	if err != nil {
		releaseScannedLines(scanBuf)
		return nil, err
	}
	scanBuf.lines = lines
	defer releaseScannedLines(scanBuf)

	p := parser{
		source: s,
		lines:  lines,
		strict: strict,
	}
	p.arrayStack = p.arrayStackBuf[:0]
	p.fieldsScratch = p.fieldsScratchBuf[:0]

	val, err := p.parseRoot()
	if err != nil {
		return nil, err
	}
	if expand == ExpandPathsSafe {
		val, err = expandPathsSafe(val, strict)
		if err != nil {
			return nil, err
		}
	}
	return val, nil
}

func bytesToStringNoCopy(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(data), len(data))
}

func (p *parser) lineContent(ln scannedLine) string {
	return p.source[ln.start:ln.end]
}

func (p *parser) pushArrayDepth(depth int32) {
	p.arrayStack = append(p.arrayStack, depth)
}

func (p *parser) popArrayDepth() {
	p.arrayStack = p.arrayStack[:len(p.arrayStack)-1]
}

func (p *parser) blankInside(depth int32) bool {
	j := p.pos + 1
	for j < len(p.lines) && p.lines[j].start == p.lines[j].end {
		j++
	}
	if j >= len(p.lines) {
		return false
	}
	return p.lines[j].depth > depth
}

func (p *parser) skipBlank() error {
	for p.pos < len(p.lines) && p.lines[p.pos].start == p.lines[p.pos].end {
		if p.strict && len(p.arrayStack) > 0 {
			d := p.arrayStack[len(p.arrayStack)-1]
			if p.blankInside(d) {
				return &Error{Line: p.pos + 1, Message: "blank line inside array"}
			}
		}
		p.pos++
	}
	return nil
}

func (p *parser) parseRoot() (Value, error) {
	if err := p.skipBlank(); err != nil {
		return nil, err
	}
	if p.pos >= len(p.lines) {
		return Object{}, nil
	}
	first := p.lines[p.pos]
	firstLine := p.pos + 1
	if first.depth != 0 {
		return nil, &Error{Line: firstLine, Message: "invalid indentation at root"}
	}
	firstContent := p.lineContent(first)

	if looksLikeRootArrayHeaderLine(firstContent) {
		h, ok, err := parseHeaderLine(firstContent, p.strict, &p.fieldsScratch)
		if err != nil {
			return nil, withLine(err, firstLine)
		}
		if ok && !h.hasKey {
			p.pos++
			arr, err := p.parseArrayFromHeader(h, 0, false)
			if err != nil {
				return nil, err
			}
			if err := p.skipBlank(); err != nil {
				return nil, err
			}
			if p.pos < len(p.lines) && p.strict {
				return nil, &Error{Line: p.pos + 1, Message: "unexpected content after root array"}
			}
			return arr, nil
		}
	}

	if h, ok, err := parseHeaderLine(firstContent, p.strict, &p.fieldsScratch); err != nil {
		return nil, withLine(err, firstLine)
	} else if ok && h.hasKey {
		return p.parseObjectAtDepth(0)
	}

	nonBlank0 := 0
	invalid0 := 0
	tmpPos := p.pos
	for tmpPos < len(p.lines) {
		ln := p.lines[tmpPos]
		if ln.start == ln.end {
			tmpPos++
			continue
		}
		if ln.depth != 0 {
			tmpPos++
			continue
		}
		nonBlank0++
		content := p.lineContent(ln)
		isHdr := looksLikeRootArrayHeaderLine(content)
		isKV := looksLikeKeyValue(content)
		if !isHdr && !isKV {
			invalid0++
		}
		tmpPos++
	}

	if nonBlank0 == 1 && !looksLikeKeyValue(firstContent) && !looksLikeRootArrayHeaderLine(firstContent) {
		p.pos++
		v, err := parsePrimitiveToken(firstContent)
		if err != nil {
			return nil, withLine(err, firstLine)
		}
		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		if p.pos < len(p.lines) && p.strict {
			return nil, &Error{Line: p.pos + 1, Message: "unexpected content after root primitive"}
		}
		return v, nil
	}

	if p.strict && invalid0 >= 2 {
		return nil, &Error{Line: firstLine, Message: "invalid root form"}
	}

	return p.parseObjectAtDepth(0)
}

func looksLikeKeyValue(line string) bool {
	colonPos := firstUnquotedIndex(line, ':')
	if colonPos < 0 {
		return false
	}
	keyPart := line[:colonPos]
	keyPart = trimSpaces(keyPart)
	if keyPart == "" {
		return false
	}
	if keyPart[0] == '"' {
		_, err := parseQuoted(keyPart)
		return err == nil
	}
	return true
}

func withLine(err error, line int) error {
	if err == nil {
		return nil
	}
	if te, ok := err.(*Error); ok {
		if te.Line == 0 {
			return &Error{Line: line, Column: te.Column, Message: te.Message}
		}
	}
	return err
}

func (p *parser) parseObjectAtDepth(depth int32) (Value, error) {
	obj := Object{}
	for {
		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		if p.pos >= len(p.lines) {
			break
		}
		ln := p.lines[p.pos]
		lineNo := p.pos + 1
		content := p.lineContent(ln)
		if ln.depth < depth {
			break
		}
		if ln.depth > depth {
			if p.strict {
				return nil, &Error{Line: lineNo, Message: "unexpected indentation"}
			}
			p.pos++
			continue
		}

		h, ok, err := parseHeaderLine(content, p.strict, &p.fieldsScratch)
		if err != nil {
			return nil, withLine(err, lineNo)
		}
		if ok && h.hasKey {
			p.pos++
			v, err := p.parseArrayFromHeader(h, depth, false)
			if err != nil {
				return nil, err
			}
			obj.setWithQuoted(h.key, v, h.keyQuoted)
			continue
		}

		colonPos := firstUnquotedIndex(content, ':')
		if colonPos < 0 {
			return nil, &Error{Line: lineNo, Message: "missing colon after key"}
		}
		keyPart := content[:colonPos]
		valPart := content[colonPos+1:]
		key, keyQuoted, err := parseKeyTokenWithQuoted(keyPart)
		if err != nil {
			return nil, withLine(err, lineNo)
		}
		valPart = trimSpaces(valPart)
		p.pos++

		if valPart == "" {
			nested, err := p.parseObjectAtDepth(depth + 1)
			if err != nil {
				return nil, err
			}
			obj.setWithQuoted(key, nested, keyQuoted)
			continue
		}

		val, err := parsePrimitiveTokenTrimmed(valPart)
		if err != nil {
			return nil, withLine(err, lineNo)
		}
		obj.setWithQuoted(key, val, keyQuoted)
	}
	return obj, nil
}

func (p *parser) parseArrayFromHeader(h header, headerDepth int32, inListItemTabularFirst bool) (Value, error) {
	if h.fields != nil {
		return p.parseTabularArray(h, headerDepth, inListItemTabularFirst)
	}
	if h.inlineTail != "" {
		return p.parseInlinePrimitiveArray(h)
	}
	if h.length == 0 {
		return Array{}, nil
	}
	return p.parseExpandedArray(h, headerDepth)
}

func (p *parser) parseInlinePrimitiveArray(h header) (Value, error) {
	values := make(Array, 0, h.length)
	if strings.IndexByte(h.inlineTail, '"') < 0 {
		start := 0
		for i := 0; i <= len(h.inlineTail); i++ {
			if i < len(h.inlineTail) && h.inlineTail[i] != h.delimiter {
				continue
			}
			tok := trimSpaces(h.inlineTail[start:i])
			values = append(values, parseUnquotedPrimitiveToken(tok))
			start = i + 1
		}
		if p.strict && len(values) != h.length {
			return nil, &Error{Message: "expected " + itoa(h.length) + " inline array values, but got " + itoa(len(values))}
		}
		return values, nil
	}

	it := newDelimitedScanner(h.inlineTail, h.delimiter)
	for {
		tok, ok := it.next()
		if !ok {
			break
		}
		v, err := parsePrimitiveTokenTrimmed(tok)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	if p.strict && len(values) != h.length {
		return nil, &Error{Message: "expected " + itoa(h.length) + " inline array values, but got " + itoa(len(values))}
	}
	return values, nil
}

func (p *parser) parseExpandedArray(h header, headerDepth int32) (Value, error) {
	p.pushArrayDepth(headerDepth)
	defer p.popArrayDepth()

	values := make(Array, 0, h.length)
	for {
		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		if p.pos >= len(p.lines) {
			break
		}
		ln := p.lines[p.pos]
		lineNo := p.pos + 1
		if ln.depth <= headerDepth {
			break
		}
		if ln.depth != headerDepth+1 {
			return nil, &Error{Line: lineNo, Message: "unexpected indentation inside array"}
		}
		v, err := p.parseListItem(headerDepth + 1)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	if p.strict && len(values) != h.length {
		return nil, &Error{Message: "expected " + itoa(h.length) + " list array items, but got " + itoa(len(values))}
	}
	return values, nil
}

func (p *parser) parseListItem(itemDepth int32) (Value, error) {
	ln := p.lines[p.pos]
	lineNo := p.pos + 1
	if ln.depth != itemDepth {
		return nil, &Error{Line: lineNo, Message: "unexpected list item indentation"}
	}
	s := trimSpaces(p.lineContent(ln))
	if s == "-" {
		p.pos++
		return Object{}, nil
	}
	if len(s) < 2 || s[0] != '-' || s[1] != ' ' {
		return nil, &Error{Line: lineNo, Message: "invalid list item marker"}
	}
	rest := s[2:]
	p.pos++

	if rest == "" {
		return nil, &Error{Line: lineNo, Message: "empty list item"}
	}

	h, ok, err := parseHeaderLine(rest, p.strict, &p.fieldsScratch)
	if err != nil {
		return nil, withLine(err, lineNo)
	}
	if ok && !h.hasKey {
		arr, err := p.parseArrayFromHeader(h, itemDepth, false)
		if err != nil {
			return nil, err
		}
		return arr, nil
	}

	if looksLikeField(rest) {
		return p.parseListItemObject(itemDepth, rest)
	}

	v, err := parsePrimitiveToken(rest)
	if err != nil {
		return nil, withLine(err, lineNo)
	}
	return v, nil
}

func looksLikeField(rest string) bool {
	var buf [8]string
	scratch := buf[:0]
	h, ok, _ := parseHeaderLine(rest, false, &scratch)
	if ok && h.hasKey {
		return true
	}
	colonPos := firstUnquotedIndex(rest, ':')
	if colonPos < 0 {
		return false
	}
	keyPart := trimSpaces(rest[:colonPos])
	if keyPart == "" {
		return false
	}
	if keyPart[0] == '"' {
		_, err := parseQuoted(keyPart)
		return err == nil
	}
	return true
}

func (p *parser) parseListItemObject(itemDepth int32, firstLine string) (Value, error) {
	obj := Object{}
	firstHeader, ok, err := parseHeaderLine(firstLine, p.strict, &p.fieldsScratch)
	if err != nil {
		return nil, err
	}

	if ok && firstHeader.hasKey {
		if firstHeader.fields != nil {
			arr, err := p.parseTabularArray(firstHeader, itemDepth, true)
			if err != nil {
				return nil, err
			}
			obj.setWithQuoted(firstHeader.key, arr, firstHeader.keyQuoted)
			for {
				if err := p.skipBlank(); err != nil {
					return nil, err
				}
				if p.pos >= len(p.lines) {
					break
				}
				ln := p.lines[p.pos]
				lineNo := p.pos + 1
				if ln.depth <= itemDepth {
					break
				}
				if ln.depth != itemDepth+1 {
					return nil, &Error{Line: lineNo, Message: "unexpected indentation inside list-item object"}
				}
				v, err := p.parseObjectFieldAtDepth(itemDepth + 1)
				if err != nil {
					return nil, err
				}
				m := v.(Member)
				obj.setWithQuoted(m.Key, m.Value, m.quoted)
			}
			return obj, nil
		}

		arr, err := p.parseArrayFromHeader(firstHeader, itemDepth+1, false)
		if err != nil {
			return nil, err
		}
		obj.setWithQuoted(firstHeader.key, arr, firstHeader.keyQuoted)
	} else {
		colonPos := firstUnquotedIndex(firstLine, ':')
		if colonPos < 0 {
			return nil, &Error{Message: "missing colon after key"}
		}
		keyPart := firstLine[:colonPos]
		valPart := firstLine[colonPos+1:]
		key, keyQuoted, err := parseKeyTokenWithQuoted(keyPart)
		if err != nil {
			return nil, err
		}
		valPart = trimSpaces(valPart)
		if valPart == "" {
			nested, err := p.parseObjectAtDepth(itemDepth + 2)
			if err != nil {
				return nil, err
			}
			obj.setWithQuoted(key, nested, keyQuoted)
		} else {
			v, err := parsePrimitiveTokenTrimmed(valPart)
			if err != nil {
				return nil, err
			}
			obj.setWithQuoted(key, v, keyQuoted)
		}
	}

	for {
		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		if p.pos >= len(p.lines) {
			break
		}
		ln := p.lines[p.pos]
		lineNo := p.pos + 1
		if ln.depth <= itemDepth {
			break
		}
		if ln.depth != itemDepth+1 {
			return nil, &Error{Line: lineNo, Message: "unexpected indentation inside list-item object"}
		}
		v, err := p.parseObjectFieldAtDepth(itemDepth + 1)
		if err != nil {
			return nil, err
		}
		m := v.(Member)
		obj.setWithQuoted(m.Key, m.Value, m.quoted)
	}
	return obj, nil
}

func (p *parser) parseObjectFieldAtDepth(depth int32) (Value, error) {
	ln := p.lines[p.pos]
	lineNo := p.pos + 1
	if ln.depth != depth {
		return nil, &Error{Line: lineNo, Message: "unexpected indentation"}
	}
	content := p.lineContent(ln)

	h, ok, err := parseHeaderLine(content, p.strict, &p.fieldsScratch)
	if err != nil {
		return nil, withLine(err, lineNo)
	}
	if ok && h.hasKey {
		p.pos++
		v, err := p.parseArrayFromHeader(h, depth, false)
		if err != nil {
			return nil, err
		}
		return Member{Key: h.key, Value: v, quoted: h.keyQuoted}, nil
	}

	colonPos := firstUnquotedIndex(content, ':')
	if colonPos < 0 {
		return nil, &Error{Line: lineNo, Message: "missing colon after key"}
	}
	keyPart := content[:colonPos]
	valPart := content[colonPos+1:]
	key, keyQuoted, err := parseKeyTokenWithQuoted(keyPart)
	if err != nil {
		return nil, withLine(err, lineNo)
	}
	valPart = trimSpaces(valPart)
	p.pos++

	if valPart == "" {
		nested, err := p.parseObjectAtDepth(depth + 1)
		if err != nil {
			return nil, err
		}
		return Member{Key: key, Value: nested, quoted: keyQuoted}, nil
	}

	v, err := parsePrimitiveTokenTrimmed(valPart)
	if err != nil {
		return nil, withLine(err, lineNo)
	}
	return Member{Key: key, Value: v, quoted: keyQuoted}, nil
}

func (p *parser) parseTabularArray(h header, headerDepth int32, inListItemTabularFirst bool) (Value, error) {
	p.pushArrayDepth(headerDepth)
	defer p.popArrayDepth()

	rowDepth := headerDepth + 1
	if inListItemTabularFirst {
		rowDepth = headerDepth + 2
	}

	fields := h.fields
	fieldCount := len(fields)
	if p.strict && fieldCount > 0 {
		if fieldCount == 2 {
			schema2 := &tabularSchema2{
				key0:  fields[0],
				key1:  fields[1],
				kinds: make([]uint8, h.length*fieldCount),
				texts: make([]string, h.length*fieldCount),
			}
			return p.parseTabularArray2Strict(h, rowDepth, schema2)
		}
		keysCopy := make([]string, fieldCount)
		copy(keysCopy, fields)
		schema := &tabularSchema{
			keys:       keysCopy,
			fieldCount: fieldCount,
			kinds:      make([]uint8, h.length*fieldCount),
			texts:      make([]string, h.length*fieldCount),
		}
		return p.parseTabularArrayStrict(h, rowDepth, schema)
	}

	out := make(Array, 0, h.length)
	rowCount := 0
	for {
		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		if p.pos >= len(p.lines) {
			break
		}
		ln := p.lines[p.pos]
		lineNo := p.pos + 1
		content := p.lineContent(ln)
		if ln.depth < rowDepth {
			break
		}
		if ln.depth > rowDepth {
			return nil, &Error{Line: lineNo, Message: "unexpected indentation inside tabular rows"}
		}
		if p.strict {
			if rowCount >= h.length {
				if !isTabularRowLine(content, h.delimiter) {
					break
				}
				return nil, &Error{Line: lineNo, Message: "expected " + itoa(h.length) + " tabular rows, but got more"}
			}
			// For single-field tabular arrays we must keep row/key disambiguation strict.
			if fieldCount == 1 && !isTabularRowLine(content, h.delimiter) {
				break
			}
		} else if !isTabularRowLine(content, h.delimiter) {
			break
		}

		var rowMembers []Member
		rowMembers = make([]Member, fieldCount)

		tokCount := 0
		membersLen := 0
		handledFast := false
		if fieldCount == 2 {
			tok0, tok1, n, okNoQuote := parseTwoFieldRowNoQuote(content, h.delimiter)
			if okNoQuote {
				if n >= 1 {
					rowMembers[0].Key = fields[0]
					assignUnquotedPrimitiveToken(&rowMembers[0], tok0)
					membersLen = 1
				}
				if n >= 2 {
					rowMembers[1].Key = fields[1]
					assignUnquotedPrimitiveToken(&rowMembers[1], tok1)
					membersLen = 2
				}
				tokCount = n
				handledFast = true
			}
		}
		if !handledFast {
			if strings.IndexByte(content, '"') < 0 {
				start := 0
				for i := 0; i <= len(content); i++ {
					if i < len(content) && content[i] != h.delimiter {
						continue
					}
					tok := trimSpaces(content[start:i])
					if tokCount < fieldCount {
						rowMembers[tokCount].Key = fields[tokCount]
						assignUnquotedPrimitiveToken(&rowMembers[tokCount], tok)
						membersLen++
					}
					tokCount++
					start = i + 1
				}
			} else {
				it := newDelimitedScanner(content, h.delimiter)
				for {
					tok, ok := it.next()
					if !ok {
						break
					}
					if tokCount < fieldCount {
						v, err := parsePrimitiveTokenTrimmed(tok)
						if err != nil {
							return nil, withLine(err, lineNo)
						}
						rowMembers[tokCount].Key = fields[tokCount]
						rowMembers[tokCount].Value = v
						membersLen++
					}
					tokCount++
				}
			}
		}
		if p.strict && tokCount != fieldCount {
			return nil, &Error{Line: lineNo, Message: "expected " + itoa(fieldCount) + " values in row, but got " + itoa(tokCount)}
		}

		out = append(out, Object{Members: rowMembers[:membersLen]})
		p.pos++
		rowCount++
	}

	if p.strict && rowCount != h.length {
		return nil, &Error{Message: "expected " + itoa(h.length) + " tabular rows, but got " + itoa(rowCount)}
	}

	return out, nil
}

func (p *parser) parseTabularArrayStrict(h header, rowDepth int32, schema *tabularSchema) (Value, error) {
	fieldCount := schema.fieldCount

	rowCount := 0
	for {
		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		if p.pos >= len(p.lines) {
			break
		}
		ln := p.lines[p.pos]
		lineNo := p.pos + 1
		content := p.lineContent(ln)
		if ln.depth < rowDepth {
			break
		}
		if ln.depth > rowDepth {
			return nil, &Error{Line: lineNo, Message: "unexpected indentation inside tabular rows"}
		}
		if rowCount >= h.length {
			if !isTabularRowLine(content, h.delimiter) {
				break
			}
			return nil, &Error{Line: lineNo, Message: "expected " + itoa(h.length) + " tabular rows, but got more"}
		}
		if fieldCount == 1 && !isTabularRowLine(content, h.delimiter) {
			break
		}

		rowBase := rowCount * fieldCount
		tokCount := 0
		handledFast := false
		if fieldCount == 2 {
			tok0, tok1, n, okNoQuote := parseTwoFieldRowNoQuote(content, h.delimiter)
			if okNoQuote {
				if n >= 1 {
					assignUnquotedPackedCellToken(&schema.kinds[rowBase], &schema.texts[rowBase], tok0)
				}
				if n >= 2 {
					assignUnquotedPackedCellToken(&schema.kinds[rowBase+1], &schema.texts[rowBase+1], tok1)
				}
				tokCount = n
				handledFast = true
			}
		}

		if !handledFast {
			if strings.IndexByte(content, '"') < 0 {
				start := 0
				for i := 0; i <= len(content); i++ {
					if i < len(content) && content[i] != h.delimiter {
						continue
					}
					if tokCount < fieldCount {
						idx := rowBase + tokCount
						assignUnquotedPackedCellToken(&schema.kinds[idx], &schema.texts[idx], trimSpaces(content[start:i]))
					}
					tokCount++
					start = i + 1
				}
			} else {
				it := newDelimitedScanner(content, h.delimiter)
				for {
					tok, ok := it.next()
					if !ok {
						break
					}
					if tokCount < fieldCount {
						v, err := parsePrimitiveTokenTrimmed(tok)
						if err != nil {
							return nil, withLine(err, lineNo)
						}
						idx := rowBase + tokCount
						if !assignPackedCellValue(&schema.kinds[idx], &schema.texts[idx], v) {
							return nil, &Error{Line: lineNo, Message: "non-primitive tabular value"}
						}
					}
					tokCount++
				}
			}
		}

		if tokCount != fieldCount {
			return nil, &Error{Line: lineNo, Message: "expected " + itoa(fieldCount) + " values in row, but got " + itoa(tokCount)}
		}

		p.pos++
		rowCount++
	}

	if rowCount != h.length {
		return nil, &Error{Message: "expected " + itoa(h.length) + " tabular rows, but got " + itoa(rowCount)}
	}

	return &tabularArray{schema: schema, len: rowCount}, nil
}

func (p *parser) parseTabularArray2Strict(h header, rowDepth int32, schema2 *tabularSchema2) (Value, error) {
	rowCount := 0
	for {
		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		if p.pos >= len(p.lines) {
			break
		}
		ln := p.lines[p.pos]
		lineNo := p.pos + 1
		content := p.lineContent(ln)
		if ln.depth < rowDepth {
			break
		}
		if ln.depth > rowDepth {
			return nil, &Error{Line: lineNo, Message: "unexpected indentation inside tabular rows"}
		}
		if rowCount >= h.length {
			if !isTabularRowLine(content, h.delimiter) {
				break
			}
			return nil, &Error{Line: lineNo, Message: "expected " + itoa(h.length) + " tabular rows, but got more"}
		}

		rowBase := rowCount * 2
		tok0, tok1, tokCount, okNoQuote := parseTwoFieldRowNoQuote(content, h.delimiter)
		if okNoQuote {
			if tokCount >= 1 {
				assignUnquotedPackedCellToken(&schema2.kinds[rowBase], &schema2.texts[rowBase], tok0)
			}
			if tokCount >= 2 {
				assignUnquotedPackedCellToken(&schema2.kinds[rowBase+1], &schema2.texts[rowBase+1], tok1)
			}
		} else {
			tokCount = 0
			it := newDelimitedScanner(content, h.delimiter)
			for {
				tok, ok := it.next()
				if !ok {
					break
				}
				if tokCount < 2 {
					v, err := parsePrimitiveTokenTrimmed(tok)
					if err != nil {
						return nil, withLine(err, lineNo)
					}
					idx := rowBase + tokCount
					if !assignPackedCellValue(&schema2.kinds[idx], &schema2.texts[idx], v) {
						return nil, &Error{Line: lineNo, Message: "non-primitive tabular value"}
					}
				}
				tokCount++
			}
		}

		if tokCount != 2 {
			return nil, &Error{Line: lineNo, Message: "expected 2 values in row, but got " + itoa(tokCount)}
		}
		p.pos++
		rowCount++
	}

	if rowCount != h.length {
		return nil, &Error{Message: "expected " + itoa(h.length) + " tabular rows, but got " + itoa(rowCount)}
	}

	return &tabularArray2{schema: schema2, len: rowCount}, nil
}

func parseTwoFieldRowNoQuote(line string, delim byte) (tok0 string, tok1 string, tokCount int, ok bool) {
	first := -1
	second := -1
	delimCount := 0
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '"' {
			return "", "", 0, false
		}
		if c != delim {
			continue
		}
		if delimCount == 0 {
			first = i
		} else if delimCount == 1 {
			second = i
		}
		delimCount++
	}

	switch delimCount {
	case 0:
		return trimSpaces(line), "", 1, true
	case 1:
		return trimSpaces(line[:first]), trimSpaces(line[first+1:]), 2, true
	default:
		return trimSpaces(line[:first]), trimSpaces(line[first+1 : second]), delimCount + 1, true
	}
}

func countDelims(s string, delim byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == delim {
			n++
		}
	}
	return n
}

func isTabularRowLine(line string, delim byte) bool {
	dpos := firstUnquotedIndex(line, delim)
	cpos := firstUnquotedIndex(line, ':')
	if cpos < 0 {
		return true
	}
	if dpos < 0 {
		return false
	}
	return dpos < cpos
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	buf := make([]byte, 0, 16)
	for n > 0 {
		buf = append(buf, byte('0'+(n%10)))
		n /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return sign + string(buf)
}
