package toon

import "strings"

func foldValueSafe(v Value, flattenDepth int) Value {
	switch t := v.(type) {
	case Object:
		return foldObjectSafe(t, flattenDepth)
	case Array:
		out := make(Array, 0, len(t))
		for _, x := range t {
			out = append(out, foldValueSafe(x, flattenDepth))
		}
		return out
	default:
		return v
	}
}

func foldObjectSafe(obj Object, flattenDepth int) Object {
	reserved := make(map[string]struct{}, len(obj.Members))
	for _, m := range obj.Members {
		reserved[m.Key] = struct{}{}
	}

	out := Object{Members: make([]Member, 0, len(obj.Members))}
	for _, m := range obj.Members {
		k, v := foldChainSafe(m.Key, m.Value, flattenDepth, reserved)
		switch vv := v.(type) {
		case Object:
			v = foldObjectSafe(vv, flattenDepth)
		case Array:
			v = foldValueSafe(vv, flattenDepth)
		}
		out.Members = append(out.Members, Member{Key: k, Value: v})
	}
	return out
}

func foldChainSafe(key string, v Value, flattenDepth int, reserved map[string]struct{}) (string, Value) {
	if flattenDepth < 2 {
		return key, v
	}

	segments := []string{key}
	if !isIdentifierSegment(key) {
		return key, v
	}

	cur := v
	for len(segments) < flattenDepth {
		o, ok := cur.(Object)
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
		return key, v
	}

	folded := strings.Join(segments, ".")
	if folded != key {
		if _, ok := reserved[folded]; ok {
			return key, v
		}
		delete(reserved, key)
		reserved[folded] = struct{}{}
	}

	if len(segments) == 2 {
		if o, ok := v.(Object); ok && len(o.Members) == 1 {
			return folded, o.Members[0].Value
		}
	}

	remainder := v
	for i := 1; i < len(segments); i++ {
		o := remainder.(Object)
		remainder = o.Members[0].Value
	}
	return folded, remainder
}
