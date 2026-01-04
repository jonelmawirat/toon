package toon

import (
	"io"
)

type parser struct {
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
	s := string(data)
	var linesBuf [64]scannedLine
	lines, err := scanString(s, indent, strict, linesBuf[:0])
	if err != nil {
		return nil, err
	}

	p := parser{
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

func (p *parser) pushArrayDepth(depth int32) {
	p.arrayStack = append(p.arrayStack, depth)
}

func (p *parser) popArrayDepth() {
	p.arrayStack = p.arrayStack[:len(p.arrayStack)-1]
}

func (p *parser) blankInside(depth int32) bool {
	j := p.pos + 1
	for j < len(p.lines) && p.lines[j].blank {
		j++
	}
	if j >= len(p.lines) {
		return false
	}
	return p.lines[j].depth > depth
}

func (p *parser) skipBlank() error {
	for p.pos < len(p.lines) && p.lines[p.pos].blank {
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

	if looksLikeRootArrayHeaderLine(first.content) {
		h, ok, err := parseHeaderLine(first.content, p.strict, &p.fieldsScratch)
		if err != nil {
			return nil, withLine(err, firstLine)
		}
		if ok && h.key == "" {
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

	nonBlank0 := 0
	invalid0 := 0
	tmpPos := p.pos
	for tmpPos < len(p.lines) {
		ln := p.lines[tmpPos]
		if ln.blank {
			tmpPos++
			continue
		}
		if ln.depth != 0 {
			tmpPos++
			continue
		}
		nonBlank0++
		isHdr := looksLikeRootArrayHeaderLine(ln.content)
		isKV := looksLikeKeyValue(ln.content)
		if !isHdr && !isKV {
			invalid0++
		}
		tmpPos++
	}

	if nonBlank0 == 1 && !looksLikeKeyValue(first.content) && !looksLikeRootArrayHeaderLine(first.content) {
		p.pos++
		v, err := parsePrimitiveToken(first.content)
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

		h, ok, err := parseHeaderLine(ln.content, p.strict, &p.fieldsScratch)
		if err != nil {
			return nil, withLine(err, lineNo)
		}
		if ok && h.key != "" {
			p.pos++
			v, err := p.parseArrayFromHeader(h, depth, false)
			if err != nil {
				return nil, err
			}
			obj.Set(h.key, v)
			continue
		}

		colonPos := firstUnquotedIndex(ln.content, ':')
		if colonPos < 0 {
			return nil, &Error{Line: lineNo, Message: "missing colon after key"}
		}
		keyPart := ln.content[:colonPos]
		valPart := ln.content[colonPos+1:]
		key, err := parseKeyToken(keyPart)
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
			obj.Set(key, nested)
			continue
		}

		val, err := parsePrimitiveToken(valPart)
		if err != nil {
			return nil, withLine(err, lineNo)
		}
		obj.Set(key, val)
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
	it := newDelimitedScanner(h.inlineTail, h.delimiter)
	for {
		tok, ok := it.next()
		if !ok {
			break
		}
		v, err := parsePrimitiveToken(tok)
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
	s := trimSpaces(ln.content)
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
	if ok && h.key == "" {
		if h.inlineTail == "" && h.length != 0 {
			return nil, &Error{Line: lineNo, Message: "inner array must be inline"}
		}
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
	if ok && h.key != "" {
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

	if ok && firstHeader.key != "" {
		if firstHeader.fields != nil {
			arr, err := p.parseTabularArray(firstHeader, itemDepth, true)
			if err != nil {
				return nil, err
			}
			obj.Set(firstHeader.key, arr)
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
				obj.Set(m.Key, m.Value)
			}
			return obj, nil
		}

		arr, err := p.parseArrayFromHeader(firstHeader, itemDepth+1, false)
		if err != nil {
			return nil, err
		}
		obj.Set(firstHeader.key, arr)
	} else {
		colonPos := firstUnquotedIndex(firstLine, ':')
		if colonPos < 0 {
			return nil, &Error{Message: "missing colon after key"}
		}
		keyPart := firstLine[:colonPos]
		valPart := firstLine[colonPos+1:]
		key, err := parseKeyToken(keyPart)
		if err != nil {
			return nil, err
		}
		valPart = trimSpaces(valPart)
		if valPart == "" {
			nested, err := p.parseObjectAtDepth(itemDepth + 2)
			if err != nil {
				return nil, err
			}
			obj.Set(key, nested)
		} else {
			v, err := parsePrimitiveToken(valPart)
			if err != nil {
				return nil, err
			}
			obj.Set(key, v)
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
		obj.Set(m.Key, m.Value)
	}
	return obj, nil
}

func (p *parser) parseObjectFieldAtDepth(depth int32) (Value, error) {
	ln := p.lines[p.pos]
	lineNo := p.pos + 1
	if ln.depth != depth {
		return nil, &Error{Line: lineNo, Message: "unexpected indentation"}
	}

	h, ok, err := parseHeaderLine(ln.content, p.strict, &p.fieldsScratch)
	if err != nil {
		return nil, withLine(err, lineNo)
	}
	if ok && h.key != "" {
		p.pos++
		v, err := p.parseArrayFromHeader(h, depth, false)
		if err != nil {
			return nil, err
		}
		return Member{Key: h.key, Value: v}, nil
	}

	colonPos := firstUnquotedIndex(ln.content, ':')
	if colonPos < 0 {
		return nil, &Error{Line: lineNo, Message: "missing colon after key"}
	}
	keyPart := ln.content[:colonPos]
	valPart := ln.content[colonPos+1:]
	key, err := parseKeyToken(keyPart)
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
		return Member{Key: key, Value: nested}, nil
	}

	v, err := parsePrimitiveToken(valPart)
	if err != nil {
		return nil, withLine(err, lineNo)
	}
	return Member{Key: key, Value: v}, nil
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

	out := make(Array, 0, h.length)
	var memberStore []Member
	if p.strict && h.length > 0 && fieldCount > 0 {
		memberStore = make([]Member, 0, h.length*fieldCount)
	}

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
		if ln.depth < rowDepth {
			break
		}
		if ln.depth > rowDepth {
			return nil, &Error{Line: lineNo, Message: "unexpected indentation inside tabular rows"}
		}
		if !isTabularRowLine(ln.content, h.delimiter) {
			break
		}

		if p.strict && rowCount >= h.length {
			return nil, &Error{Line: lineNo, Message: "expected " + itoa(h.length) + " tabular rows, but got more"}
		}

		var rowMembers []Member
		if memberStore != nil {
			base := len(memberStore)
			memberStore = memberStore[:base+fieldCount]
			rowMembers = memberStore[base : base+fieldCount : base+fieldCount]
		} else {
			rowMembers = make([]Member, fieldCount)
		}

		tokCount := 0
		membersLen := 0
		it := newDelimitedScanner(ln.content, h.delimiter)
		for {
			tok, ok := it.next()
			if !ok {
				break
			}
			if tokCount < fieldCount {
				v, err := parsePrimitiveToken(tok)
				if err != nil {
					return nil, withLine(err, lineNo)
				}
				rowMembers[tokCount] = Member{Key: fields[tokCount], Value: v}
				membersLen++
			}
			tokCount++
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
