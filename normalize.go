package toon

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

func normalize(v any) (Value, error) {
	return normalizeAny(v)
}

func normalizeValueAny(v any) (Value, error) {
	return normalizeAny(v)
}

func normalizeAny(v any) (Value, error) {
	v = unboxValue(v)
	if arr, ok := arrayFromValue(v); ok {
		return normalizeArrayValue(arr)
	}
	switch t := v.(type) {
	case nil:
		return nil, nil
	case bool:
		return t, nil
	case string:
		return t, nil
	case Number:
		c, ok := parseNumberTokenToCanonical(string(t))
		if !ok {
			return nil, &Error{Message: "invalid number"}
		}
		return Number(c), nil
	case json.Number:
		c, ok := parseNumberTokenToCanonical(t.String())
		if !ok {
			return nil, &Error{Message: "invalid json.Number"}
		}
		return Number(c), nil
	case time.Time:
		return t.UTC().Format(time.RFC3339Nano), nil
	case big.Int:
		return Number(t.String()), nil
	case *big.Int:
		if t == nil {
			return nil, nil
		}
		return Number(t.String()), nil
	case int:
		return Number(strconv.FormatInt(int64(t), 10)), nil
	case int8:
		return Number(strconv.FormatInt(int64(t), 10)), nil
	case int16:
		return Number(strconv.FormatInt(int64(t), 10)), nil
	case int32:
		return Number(strconv.FormatInt(int64(t), 10)), nil
	case int64:
		return Number(strconv.FormatInt(t, 10)), nil
	case uint:
		return Number(strconv.FormatUint(uint64(t), 10)), nil
	case uint8:
		return Number(strconv.FormatUint(uint64(t), 10)), nil
	case uint16:
		return Number(strconv.FormatUint(uint64(t), 10)), nil
	case uint32:
		return Number(strconv.FormatUint(uint64(t), 10)), nil
	case uint64:
		return Number(strconv.FormatUint(t, 10)), nil
	case uintptr:
		return Number(strconv.FormatUint(uint64(t), 10)), nil
	case float32:
		if math.IsNaN(float64(t)) || math.IsInf(float64(t), 0) {
			return nil, nil
		}
		if t == 0 {
			return Number("0"), nil
		}
		s := strconvFloatShortest(float64(t), true)
		c, ok := parseNumberTokenToCanonical(s)
		if !ok {
			return nil, &Error{Message: "invalid float representation"}
		}
		return Number(c), nil
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return nil, nil
		}
		if t == 0 {
			return Number("0"), nil
		}
		s := strconvFloatShortest(t, false)
		c, ok := parseNumberTokenToCanonical(s)
		if !ok {
			return nil, &Error{Message: "invalid float representation"}
		}
		return Number(c), nil
	case Object:
		obj, _ := objectFromValue(t)
		return normalizeObjectValue(obj)
	case *Object:
		obj, ok := objectFromValue(t)
		if !ok {
			return nil, nil
		}
		return normalizeObjectValue(obj)
	case Array:
		return normalizeArrayValue(t)
	case []any:
		out := make(Array, 0, len(t))
		for _, x := range t {
			nv, err := normalizeAny(x)
			if err != nil {
				return nil, err
			}
			out = append(out, nv)
		}
		return out, nil
	case []Value:
		out := make(Array, 0, len(t))
		for _, x := range t {
			nv, err := normalizeAny(x)
			if err != nil {
				return nil, err
			}
			out = append(out, nv)
		}
		return out, nil
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		obj := Object{Members: make([]Member, 0, len(keys))}
		for _, k := range keys {
			nv, err := normalizeAny(t[k])
			if err != nil {
				return nil, err
			}
			obj.Members = append(obj.Members, Member{Key: k, Value: nv})
		}
		return obj, nil
	case map[string]Value:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		obj := Object{Members: make([]Member, 0, len(keys))}
		for _, k := range keys {
			nv, err := normalizeAny(t[k])
			if err != nil {
				return nil, err
			}
			obj.Members = append(obj.Members, Member{Key: k, Value: nv})
		}
		return obj, nil
	}

	return normalizeReflect(reflect.ValueOf(v))
}

func normalizeObjectValue(obj Object) (Value, error) {
	out := Object{Members: make([]Member, 0, len(obj.Members))}
	for _, m := range obj.Members {
		nv, err := normalizeValueAny(m.Value)
		if err != nil {
			return nil, err
		}
		out.Members = append(out.Members, Member{Key: m.Key, Value: nv})
	}
	return out, nil
}

func normalizeArrayValue(arr Array) (Value, error) {
	out := make(Array, 0, len(arr))
	for _, x := range arr {
		nv, err := normalizeValueAny(x)
		if err != nil {
			return nil, err
		}
		out = append(out, nv)
	}
	return out, nil
}

func normalizeReflect(v reflect.Value) (Value, error) {
	if !v.IsValid() {
		return nil, nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	if v.CanInterface() {
		if obj, ok := v.Interface().(Object); ok {
			return normalizeObjectValue(obj)
		}
		if arr, ok := v.Interface().(Array); ok {
			return normalizeArrayValue(arr)
		}
		if n, ok := v.Interface().(Number); ok {
			c, ok2 := parseNumberTokenToCanonical(string(n))
			if !ok2 {
				return nil, &Error{Message: "invalid number"}
			}
			return Number(c), nil
		}
		if n, ok := v.Interface().(json.Number); ok {
			c, ok2 := parseNumberTokenToCanonical(n.String())
			if !ok2 {
				return nil, &Error{Message: "invalid json.Number"}
			}
			return Number(c), nil
		}
		if t, ok := v.Interface().(time.Time); ok {
			return t.UTC().Format(time.RFC3339Nano), nil
		}
		if bi, ok := v.Interface().(big.Int); ok {
			return Number(bi.String()), nil
		}
		if bip, ok := v.Interface().(*big.Int); ok && bip != nil {
			return Number(bip.String()), nil
		}
	}

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Number(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return Number(strconv.FormatUint(v.Uint(), 10)), nil
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, nil
		}
		if f == 0 {
			return Number("0"), nil
		}
		s := strconvFloatShortest(f, v.Kind() == reflect.Float32)
		c, ok := parseNumberTokenToCanonical(s)
		if !ok {
			return nil, &Error{Message: "invalid float representation"}
		}
		return Number(c), nil
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return nil, nil
		}
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			return nil, &Error{Message: "[]byte is not supported"}
		}
		n := v.Len()
		a := make(Array, 0, n)
		for i := 0; i < n; i++ {
			x, err := normalizeReflect(v.Index(i))
			if err != nil {
				return nil, err
			}
			a = append(a, x)
		}
		return a, nil
	case reflect.Map:
		if v.IsNil() {
			return nil, nil
		}
		keys := v.MapKeys()
		type kv struct {
			k string
			v reflect.Value
		}
		kvs := make([]kv, 0, len(keys))
		seen := make(map[string]struct{}, len(keys))
		for _, k := range keys {
			var ks string
			if k.Kind() == reflect.String {
				ks = k.String()
			} else if k.CanInterface() {
				ks = fmt.Sprint(k.Interface())
			} else {
				ks = fmt.Sprint(k)
			}
			if _, ok := seen[ks]; ok {
				return nil, &Error{Message: "map key collision after stringification"}
			}
			seen[ks] = struct{}{}
			kvs = append(kvs, kv{k: ks, v: v.MapIndex(k)})
		}
		sort.Slice(kvs, func(i, j int) bool { return kvs[i].k < kvs[j].k })
		obj := Object{Members: make([]Member, 0, len(kvs))}
		for _, entry := range kvs {
			x, err := normalizeReflect(entry.v)
			if err != nil {
				return nil, err
			}
			obj.Members = append(obj.Members, Member{Key: entry.k, Value: x})
		}
		return obj, nil
	case reflect.Struct:
		t := v.Type()
		obj := Object{Members: make([]Member, 0, t.NumField())}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			name := f.Name
			if tag, ok := f.Tag.Lookup("json"); ok {
				parts := strings.Split(tag, ",")
				if parts[0] == "-" {
					continue
				}
				if parts[0] != "" {
					name = parts[0]
				}
			}
			x, err := normalizeReflect(v.Field(i))
			if err != nil {
				return nil, err
			}
			obj.Members = append(obj.Members, Member{Key: name, Value: x})
		}
		return obj, nil
	default:
		return nil, &Error{Message: "unsupported Go value type"}
	}
}

func strconvFloatShortest(f float64, is32 bool) string {
	if is32 {
		return strconv.FormatFloat(float64(float32(f)), 'g', -1, 32)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}
