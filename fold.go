package toon

import "strings"

func foldValueSafe(v Value, flattenDepth int) Value {
	if t, ok := v.(*Object); ok && t == nil {
		return Object{}
	}
	if obj, ok := objectFromValue(v); ok {
		return foldObjectSafe(obj, flattenDepth)
	}
	if a, ok := arrayFromValue(v); ok {
		out := make(Array, 0, len(a))
		for _, x := range a {
			out = append(out, foldValueSafe(x, flattenDepth))
		}
		return out
	}
	return v
}

func foldObjectSafe(obj Object, flattenDepth int) Object {
	reserved := make(map[string]struct{}, len(obj.Members))
	for _, m := range obj.Members {
		reserved[m.Key] = struct{}{}
	}

	out := Object{Members: make([]Member, 0, len(obj.Members))}
	for _, m := range obj.Members {
		k, v, recurse := foldChainSafe(m.Key, m.Value, flattenDepth, reserved)
		if recurse {
			v = foldValueSafe(v, flattenDepth)
		}
		out.Members = append(out.Members, Member{Key: k, Value: v})
	}
	return out
}

func foldChainSafe(key string, v Value, flattenDepth int, reserved map[string]struct{}) (string, Value, bool) {
	if flattenDepth < 2 {
		return key, v, true
	}

	segments := []string{key}
	if !isIdentifierSegment(key) {
		return key, v, true
	}

	cur := v
	for len(segments) < flattenDepth {
		o, ok := objectFromValue(cur)
		if !ok {
			break
		}
		if len(o.Members) != 1 {
			break
		}
		nextKey := o.Members[0].Key
		if strings.Contains(nextKey, ".") {
			break
		}
		if !isIdentifierSegment(nextKey) {
			break
		}
		segments = append(segments, nextKey)
		cur = o.Members[0].Value
	}

	if len(segments) < 2 {
		return key, v, true
	}

	folded := strings.Join(segments, ".")
	if folded != key {
		if _, ok := reserved[folded]; ok {
			// Collision with a literal sibling key: keep the whole chain unchanged.
			return key, v, false
		}
		delete(reserved, key)
		reserved[folded] = struct{}{}
	}

	truncatedByDepth := false
	if len(segments) == flattenDepth {
		if o, ok := objectFromValue(cur); ok && len(o.Members) == 1 {
			nextKey := o.Members[0].Key
			if !strings.Contains(nextKey, ".") && isIdentifierSegment(nextKey) {
				truncatedByDepth = true
			}
		}
	}

	if len(segments) == 2 {
		if o, ok := objectFromValue(v); ok && len(o.Members) == 1 {
			return folded, o.Members[0].Value, !truncatedByDepth
		}
	}

	remainder := v
	for i := 1; i < len(segments); i++ {
		o, ok := objectFromValue(remainder)
		if !ok || len(o.Members) == 0 {
			return key, v, true
		}
		remainder = o.Members[0].Value
	}
	return folded, remainder, !truncatedByDepth
}
