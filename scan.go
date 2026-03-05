package toon

import (
	"sync"
)

type scannedLine struct {
	start uint32
	end   uint32
	depth int32
}

type scannedLineBuffer struct {
	lines []scannedLine
}

var scannedLinePool = sync.Pool{
	New: func() any {
		return &scannedLineBuffer{lines: make([]scannedLine, 0, 128)}
	},
}

func borrowScannedLines(minCap int) *scannedLineBuffer {
	buf := scannedLinePool.Get().(*scannedLineBuffer)
	if cap(buf.lines) < minCap {
		buf.lines = make([]scannedLine, 0, minCap)
		return buf
	}
	buf.lines = buf.lines[:0]
	return buf
}

func releaseScannedLines(buf *scannedLineBuffer) {
	if cap(buf.lines) > 1<<20 {
		buf.lines = nil
		return
	}
	buf.lines = buf.lines[:0]
	scannedLinePool.Put(buf)
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
		blank := j >= lineEnd
		if strict && !blank && indent > 0 && spaces%indent != 0 {
			return nil, &Error{Line: lineNo, Column: 1, Message: "indentation must be an exact multiple of indent size"}
		}

		depth := 0
		if indent > 0 {
			depth = spaces / indent
		}

		dst = append(dst, scannedLine{
			start: uint32(j),
			end:   uint32(lineEnd),
			depth: int32(depth),
		})

		if end == len(s) {
			break
		}
		start = end + 1
		lineNo++
	}

	return dst, nil
}
