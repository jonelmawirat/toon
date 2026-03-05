package toon

import "strings"

func setObjectMemberPreserveOrder(obj *Object, key string, v Value, quoted bool) {
	for i := range obj.Members {
		if obj.Members[i].Key == key {
			obj.Members[i].Value = v
			if quoted {
				obj.Members[i].quoted = true
			}
			return
		}
	}
	obj.Members = append(obj.Members, Member{Key: key, Value: v, quoted: quoted})
}

func expandPathsSafe(v Value, strict bool) (Value, error) {
	if t, ok := v.(*Object); ok && t == nil {
		return Object{}, nil
	}
	if obj, ok := objectFromValue(v); ok {
		return expandObjectSafe(obj, strict)
	}
	if a, ok := arrayFromValue(v); ok {
		out := make(Array, 0, len(a))
		for _, x := range a {
			y, err := expandPathsSafe(x, strict)
			if err != nil {
				return nil, err
			}
			out = append(out, y)
		}
		return out, nil
	}
	return v, nil
}

func expandObjectSafe(obj Object, strict bool) (Value, error) {
	out := Object{Members: make([]Member, 0, len(obj.Members))}
	for _, m := range obj.Members {
		ev, err := expandPathsSafe(m.Value, strict)
		if err != nil {
			return nil, err
		}

		if m.quoted || !strings.Contains(m.Key, ".") {
			if existing, ok := out.Get(m.Key); ok {
				ao, aok := objectFromValue(existing)
				bo, bok := objectFromValue(ev)
				if aok && bok {
					merged, err := deepMergeObjects(ao, bo, strict)
					if err != nil {
						return nil, err
					}
					setObjectMemberPreserveOrder(&out, m.Key, merged, m.quoted)
					continue
				}
				if aok != bok {
					if strict {
						return nil, &Error{Message: "expansion conflict at path '" + m.Key + "'"}
					}
				}
			}
			setObjectMemberPreserveOrder(&out, m.Key, ev, m.quoted)
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
			setObjectMemberPreserveOrder(&out, m.Key, ev, m.quoted)
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
			ao, aok := objectFromValue(existing)
			bo, bok := objectFromValue(v)
			if aok && bok {
				merged, err := deepMergeObjects(ao, bo, strict)
				if err != nil {
					return Object{}, err
				}
				setObjectMemberPreserveOrder(&dst, key, merged, false)
				return dst, nil
			}
			if strict {
				return Object{}, &Error{Message: "expansion conflict at path '" + strings.Join(path, ".") + "'"}
			}
			setObjectMemberPreserveOrder(&dst, key, v, false)
			return dst, nil
		}
		setObjectMemberPreserveOrder(&dst, key, v, false)
		return dst, nil
	}

	existing, ok := dst.Get(key)
	var child Object
	if ok {
		if exObj, ok2 := objectFromValue(existing); ok2 {
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
	setObjectMemberPreserveOrder(&dst, key, updatedChild, false)
	return dst, nil
}

func deepMergeObjects(a Object, b Object, strict bool) (Object, error) {
	out := Object{Members: make([]Member, 0, len(a.Members)+len(b.Members))}
	for _, m := range a.Members {
		out.Members = append(out.Members, Member{Key: m.Key, Value: m.Value})
	}
	for _, m := range b.Members {
		if existing, ok := out.Get(m.Key); ok {
			ao, aok := objectFromValue(existing)
			bo, bok := objectFromValue(m.Value)
			if aok && bok {
				merged, err := deepMergeObjects(ao, bo, strict)
				if err != nil {
					return Object{}, err
				}
				setObjectMemberPreserveOrder(&out, m.Key, merged, m.quoted)
				continue
			}
			if strict {
				return Object{}, &Error{Message: "expansion conflict at path '" + m.Key + "'"}
			}
			setObjectMemberPreserveOrder(&out, m.Key, m.Value, m.quoted)
			continue
		}
		setObjectMemberPreserveOrder(&out, m.Key, m.Value, m.quoted)
	}
	return out, nil
}
