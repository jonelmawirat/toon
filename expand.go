package toon

import "strings"

func expandPathsSafe(v Value, strict bool) (Value, error) {
	switch t := v.(type) {
	case Object:
		return expandObjectSafe(t, strict)
	case Array:
		out := make(Array, 0, len(t))
		for _, x := range t {
			y, err := expandPathsSafe(x, strict)
			if err != nil {
				return nil, err
			}
			out = append(out, y)
		}
		return out, nil
	default:
		return v, nil
	}
}

func expandObjectSafe(obj Object, strict bool) (Value, error) {
	out := Object{Members: make([]Member, 0, len(obj.Members))}
	for _, m := range obj.Members {
		ev, err := expandPathsSafe(m.Value, strict)
		if err != nil {
			return nil, err
		}

		if !strings.Contains(m.Key, ".") {
			if existing, ok := out.Get(m.Key); ok {
				ao, aok := existing.(Object)
				bo, bok := ev.(Object)
				if aok && bok {
					merged, err := deepMergeObjects(ao, bo, strict)
					if err != nil {
						return nil, err
					}
					out.Set(m.Key, merged)
					continue
				}
				if aok != bok {
					if strict {
						return nil, &Error{Message: "expansion conflict at path '" + m.Key + "'"}
					}
				}
			}
			out.Set(m.Key, ev)
			continue
		}

		parts := strings.Split(m.Key, ".")
		ok := true
		for _, p := range parts {
			if !isIdentifierSegment(p) {
				ok = false
				break
			}
		}
		if !ok {
			out.Set(m.Key, ev)
			continue
		}

		merged, err := insertExpandedPath(out, parts, ev, strict)
		if err != nil {
			return nil, err
		}
		out = merged
	}
	return out, nil
}

func insertExpandedPath(dst Object, path []string, v Value, strict bool) (Object, error) {
	if len(path) == 0 {
		return dst, nil
	}

	key := path[0]
	if len(path) == 1 {
		if existing, ok := dst.Get(key); ok {
			ao, aok := existing.(Object)
			bo, bok := v.(Object)
			if aok && bok {
				merged, err := deepMergeObjects(ao, bo, strict)
				if err != nil {
					return Object{}, err
				}
				dst.Set(key, merged)
				return dst, nil
			}
			if strict {
				return Object{}, &Error{Message: "expansion conflict at path '" + strings.Join(path, ".") + "'"}
			}
			dst.Set(key, v)
			return dst, nil
		}
		dst.Set(key, v)
		return dst, nil
	}

	existing, ok := dst.Get(key)
	var child Object
	if ok {
		if exObj, ok2 := existing.(Object); ok2 {
			child = exObj
		} else {
			if strict {
				return Object{}, &Error{Message: "expansion conflict at path '" + key + "'"}
			}
			child = Object{}
		}
	} else {
		child = Object{}
	}

	updatedChild, err := insertExpandedPath(child, path[1:], v, strict)
	if err != nil {
		return Object{}, err
	}
	dst.Set(key, updatedChild)
	return dst, nil
}

func deepMergeObjects(a Object, b Object, strict bool) (Object, error) {
	out := Object{Members: make([]Member, 0, len(a.Members)+len(b.Members))}
	for _, m := range a.Members {
		out.Members = append(out.Members, Member{Key: m.Key, Value: m.Value})
	}
	for _, m := range b.Members {
		if existing, ok := out.Get(m.Key); ok {
			ao, aok := existing.(Object)
			bo, bok := m.Value.(Object)
			if aok && bok {
				merged, err := deepMergeObjects(ao, bo, strict)
				if err != nil {
					return Object{}, err
				}
				out.Set(m.Key, merged)
				continue
			}
			if strict {
				return Object{}, &Error{Message: "expansion conflict at path '" + m.Key + "'"}
			}
			out.Set(m.Key, m.Value)
			continue
		}
		out.Set(m.Key, m.Value)
	}
	return out, nil
}
