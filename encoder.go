package toon

import (
	"strings"
)

type encodeCtx struct {
	indent       int
	docDelim     byte
	defaultDelim byte
}

func encodeDocument(v Value, indent int, docDelim Delimiter, arrDelim Delimiter) (string, error) {
	ctx := encodeCtx{
		indent:       indent,
		docDelim:     byte(docDelim),
		defaultDelim: byte(arrDelim),
	}
	lines := make([]string, 0, 32)
	err := encodeRootValue(&lines, v, 0, ctx)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func indentPrefix(depth, indent int) string {
	if depth <= 0 {
		return ""
	}
	return strings.Repeat(" ", depth*indent)
}

func encodeRootValue(lines *[]string, v Value, depth int, ctx encodeCtx) error {
	switch t := v.(type) {
	case Object:
		if len(t.Members) == 0 {
			return nil
		}
		return encodeObject(lines, t, depth, ctx)
	case Array:
		return encodeArray(lines, "", t, depth, ctx, ctx.defaultDelim)
	default:
		s, err := encodePrimitiveToken(v, ctx.docDelim, true)
		if err != nil {
			return err
		}
		*lines = append(*lines, s)
		return nil
	}
}

func encodeObject(lines *[]string, obj Object, depth int, ctx encodeCtx) error {
	for _, m := range obj.Members {
		if err := encodeObjectField(lines, m.Key, m.Value, depth, ctx); err != nil {
			return err
		}
	}
	return nil
}

func encodeObjectField(lines *[]string, key string, val Value, depth int, ctx encodeCtx) error {
	ek, err := encodeKey(key)
	if err != nil {
		return err
	}

	switch t := val.(type) {
	case Object:
		prefix := indentPrefix(depth, ctx.indent)
		*lines = append(*lines, prefix+ek+":")
		if len(t.Members) == 0 {
			return nil
		}
		return encodeObject(lines, t, depth+1, ctx)
	case Array:
		return encodeArray(lines, key, t, depth, ctx, ctx.defaultDelim)
	default:
		prefix := indentPrefix(depth, ctx.indent)
		s, err := encodePrimitiveToken(val, ctx.docDelim, true)
		if err != nil {
			return err
		}
		*lines = append(*lines, prefix+ek+": "+s)
		return nil
	}
}

func encodeHeader(key string, length int, delim byte, fields []string) (string, error) {
	var b strings.Builder

	if key != "" {
		ek, err := encodeKey(key)
		if err != nil {
			return "", err
		}
		b.WriteString(ek)
	}

	b.WriteByte('[')
	b.WriteString(itoa(length))
	if delim == '\t' || delim == '|' {
		b.WriteByte(delim)
	}
	b.WriteByte(']')

	if fields != nil {
		b.WriteByte('{')
		for i, f := range fields {
			if i > 0 {
				b.WriteByte(delim)
			}
			ef, err := encodeKey(f)
			if err != nil {
				return "", err
			}
			b.WriteString(ef)
		}
		b.WriteByte('}')
	}

	b.WriteByte(':')
	return b.String(), nil
}

func arrayKind(a Array) string {
	if len(a) == 0 {
		return "inline"
	}
	allPrim := true
	allArrPrim := true
	for _, v := range a {
		if _, ok := asPrimitive(v); !ok {
			allPrim = false
		}
		av, ok := v.(Array)
		if !ok || !isAllPrimitives(av) {
			allArrPrim = false
		}
	}
	if allPrim {
		return "inline"
	}
	if allArrPrim {
		return "arrayofarrays"
	}
	if fields, ok := tabularFields(a); ok {
		_ = fields
		return "tabular"
	}
	return "expanded"
}

func tabularFields(a Array) ([]string, bool) {
	if len(a) == 0 {
		return nil, false
	}
	first, ok := a[0].(Object)
	if !ok {
		return nil, false
	}
	if len(first.Members) == 0 {
		return nil, false
	}
	fields := make([]string, 0, len(first.Members))
	fieldSet := make(map[string]struct{}, len(first.Members))
	for _, m := range first.Members {
		fields = append(fields, m.Key)
		fieldSet[m.Key] = struct{}{}
		if _, ok := asPrimitive(m.Value); !ok {
			return nil, false
		}
	}
	for i := 1; i < len(a); i++ {
		obj, ok := a[i].(Object)
		if !ok {
			return nil, false
		}
		if len(obj.Members) != len(fields) {
			return nil, false
		}
		for _, m := range obj.Members {
			if _, ok := fieldSet[m.Key]; !ok {
				return nil, false
			}
			if _, ok := asPrimitive(m.Value); !ok {
				return nil, false
			}
		}
	}
	return fields, true
}

func objectValueByKey(o Object, key string) (Value, bool) {
	for _, m := range o.Members {
		if m.Key == key {
			return m.Value, true
		}
	}
	return nil, false
}

func encodeArray(lines *[]string, key string, a Array, depth int, ctx encodeCtx, delim byte) error {
	kind := arrayKind(a)
	switch kind {
	case "inline":
		return encodeInlinePrimitiveArray(lines, key, a, depth, ctx, delim)
	case "arrayofarrays":
		return encodeArrayOfArrays(lines, key, a, depth, ctx, delim)
	case "tabular":
		return encodeTabularArray(lines, key, a, depth, ctx, delim)
	default:
		return encodeExpandedArray(lines, key, a, depth, ctx, delim)
	}
}

func encodeInlinePrimitiveArray(lines *[]string, key string, a Array, depth int, ctx encodeCtx, delim byte) error {
	h, err := encodeHeader(key, len(a), delim, nil)
	if err != nil {
		return err
	}
	prefix := indentPrefix(depth, ctx.indent)
	if len(a) == 0 {
		*lines = append(*lines, prefix+h)
		return nil
	}
	parts := make([]string, 0, len(a))
	for _, v := range a {
		s, err := encodePrimitiveToken(v, delim, true)
		if err != nil {
			return err
		}
		parts = append(parts, s)
	}
	joined := strings.Join(parts, string([]byte{delim}))
	*lines = append(*lines, prefix+h+" "+joined)
	return nil
}

func encodeArrayOfArrays(lines *[]string, key string, a Array, depth int, ctx encodeCtx, delim byte) error {
	h, err := encodeHeader(key, len(a), delim, nil)
	if err != nil {
		return err
	}
	prefix := indentPrefix(depth, ctx.indent)
	*lines = append(*lines, prefix+h)

	for _, v := range a {
		inner := v.(Array)
		ih, err := encodeHeader("", len(inner), delim, nil)
		if err != nil {
			return err
		}
		if len(inner) == 0 {
			*lines = append(*lines, indentPrefix(depth+1, ctx.indent)+"- "+ih)
			continue
		}
		parts := make([]string, 0, len(inner))
		for _, pv := range inner {
			s, err := encodePrimitiveToken(pv, delim, true)
			if err != nil {
				return err
			}
			parts = append(parts, s)
		}
		joined := strings.Join(parts, string([]byte{delim}))
		*lines = append(*lines, indentPrefix(depth+1, ctx.indent)+"- "+ih+" "+joined)
	}
	return nil
}

func encodeTabularArray(lines *[]string, key string, a Array, depth int, ctx encodeCtx, delim byte) error {
	fields, ok := tabularFields(a)
	if !ok {
		return encodeExpandedArray(lines, key, a, depth, ctx, delim)
	}
	h, err := encodeHeader(key, len(a), delim, fields)
	if err != nil {
		return err
	}
	prefix := indentPrefix(depth, ctx.indent)
	*lines = append(*lines, prefix+h)

	for _, rowV := range a {
		row := rowV.(Object)
		parts := make([]string, 0, len(fields))
		for _, f := range fields {
			v, _ := objectValueByKey(row, f)
			s, err := encodePrimitiveToken(v, delim, true)
			if err != nil {
				return err
			}
			parts = append(parts, s)
		}
		joined := strings.Join(parts, string([]byte{delim}))
		*lines = append(*lines, indentPrefix(depth+1, ctx.indent)+joined)
	}
	return nil
}

func encodeExpandedArray(lines *[]string, key string, a Array, depth int, ctx encodeCtx, delim byte) error {
	h, err := encodeHeader(key, len(a), delim, nil)
	if err != nil {
		return err
	}
	prefix := indentPrefix(depth, ctx.indent)
	*lines = append(*lines, prefix+h)

	for _, item := range a {
		if err := encodeListItem(lines, item, depth+1, ctx, delim); err != nil {
			return err
		}
	}
	return nil
}

func encodeListItem(lines *[]string, item Value, itemDepth int, ctx encodeCtx, delim byte) error {
	switch t := item.(type) {
	case Object:
		return encodeListItemObject(lines, t, itemDepth, ctx, delim)
	case Array:
		if isAllPrimitives(t) {
			h, err := encodeHeader("", len(t), delim, nil)
			if err != nil {
				return err
			}
			if len(t) == 0 {
				*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"- "+h)
				return nil
			}
			parts := make([]string, 0, len(t))
			for _, v := range t {
				s, err := encodePrimitiveToken(v, delim, true)
				if err != nil {
					return err
				}
				parts = append(parts, s)
			}
			joined := strings.Join(parts, string([]byte{delim}))
			*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"- "+h+" "+joined)
			return nil
		}
		h, err := encodeHeader("", len(t), delim, nil)
		if err != nil {
			return err
		}
		*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"- "+h)
		for _, v := range t {
			if err := encodeListItem(lines, v, itemDepth+1, ctx, delim); err != nil {
				return err
			}
		}
		return nil
	default:
		s, err := encodePrimitiveToken(item, 0, false)
		if err != nil {
			return err
		}
		*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"- "+s)
		return nil
	}
}

func encodeListItemObject(lines *[]string, obj Object, itemDepth int, ctx encodeCtx, delim byte) error {
	if len(obj.Members) == 0 {
		*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"-")
		return nil
	}

	first := obj.Members[0]
	if a, ok := first.Value.(Array); ok {
		if _, tab := tabularFields(a); tab {
			fields, _ := tabularFields(a)
			h, err := encodeHeader(first.Key, len(a), delim, fields)
			if err != nil {
				return err
			}
			*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"- "+h)
			for _, rowV := range a {
				row := rowV.(Object)
				parts := make([]string, 0, len(fields))
				for _, f := range fields {
					v, _ := objectValueByKey(row, f)
					s, err := encodePrimitiveToken(v, delim, true)
					if err != nil {
						return err
					}
					parts = append(parts, s)
				}
				joined := strings.Join(parts, string([]byte{delim}))
				*lines = append(*lines, indentPrefix(itemDepth+2, ctx.indent)+joined)
			}
			for i := 1; i < len(obj.Members); i++ {
				m := obj.Members[i]
				if err := encodeObjectField(lines, m.Key, m.Value, itemDepth+1, ctx); err != nil {
					return err
				}
			}
			return nil
		}
	}

	if err := encodeListItemFirstField(lines, first, itemDepth, ctx, delim); err != nil {
		return err
	}

	for i := 1; i < len(obj.Members); i++ {
		m := obj.Members[i]
		if err := encodeObjectField(lines, m.Key, m.Value, itemDepth+1, ctx); err != nil {
			return err
		}
	}
	return nil
}

func encodeListItemFirstField(lines *[]string, m Member, itemDepth int, ctx encodeCtx, delim byte) error {
	ek, err := encodeKey(m.Key)
	if err != nil {
		return err
	}
	prefix := indentPrefix(itemDepth, ctx.indent) + "- "

	switch t := m.Value.(type) {
	case Object:
		*lines = append(*lines, prefix+ek+":")
		if len(t.Members) == 0 {
			return nil
		}
		return encodeObject(lines, t, itemDepth+2, ctx)
	case Array:
		if _, tab := tabularFields(t); tab {
			return &Error{Message: "tabular array first-field must use special list-item form"}
		}
		kind := arrayKind(t)
		if kind == "inline" {
			h, err := encodeHeader(m.Key, len(t), delim, nil)
			if err != nil {
				return err
			}
			if len(t) == 0 {
				*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"- "+h)
				return nil
			}
			parts := make([]string, 0, len(t))
			for _, v := range t {
				s, err := encodePrimitiveToken(v, delim, true)
				if err != nil {
					return err
				}
				parts = append(parts, s)
			}
			joined := strings.Join(parts, string([]byte{delim}))
			*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"- "+h+" "+joined)
			return nil
		}
		h, err := encodeHeader(m.Key, len(t), delim, nil)
		if err != nil {
			return err
		}
		*lines = append(*lines, indentPrefix(itemDepth, ctx.indent)+"- "+h)
		for _, item := range t {
			if err := encodeListItem(lines, item, itemDepth+2, ctx, delim); err != nil {
				return err
			}
		}
		return nil
	default:
		s, err := encodePrimitiveToken(m.Value, ctx.docDelim, true)
		if err != nil {
			return err
		}
		*lines = append(*lines, prefix+ek+": "+s)
		return nil
	}
}
