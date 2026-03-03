package runtime

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/constraints"
)

func SplitKeyValInt[V constraints.Signed](s, sep, sepKV string, base, bitSize int, out *map[string]V) (err error) {
	if s == "" {
		*out = map[string]V{}
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make(map[string]V, len(parts))
	for _, v := range parts {
		kv := strings.Split(v, sepKV)
		if len(kv) != 2 { //nolint:mnd
			return errors.New("invalid string format, should be 'key" + sepKV + "val" + sep + "key" + sepKV + "val'")
		}
		var i V
		if err := ParseInt(kv[1], base, bitSize, &i); err != nil {
			return err
		}
		resultOut[kv[0]] = i
	}

	*out = resultOut

	return
}

func SplitKeyValUint[V constraints.Unsigned](s, sep, sepKV string, base, bitSize int, out *map[string]V) (err error) {
	if s == "" {
		*out = map[string]V{}
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make(map[string]V, len(parts))
	for _, v := range parts {
		kv := strings.Split(v, sepKV)
		if len(kv) != 2 { //nolint: mnd
			return errors.New("invalid string format, should be 'key" + sepKV + "val" + sep + "key" + sepKV + "val'")
		}
		var i V
		if err := ParseUint(kv[1], base, bitSize, &i); err != nil {
			return err
		}
		resultOut[kv[0]] = i
	}

	*out = resultOut

	return
}

func SplitKeyValFloat[V constraints.Float](s, sep, sepKV string, bitSize int, out *map[string]V) (err error) {
	if s == "" {
		*out = map[string]V{}
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make(map[string]V, len(parts))
	for _, v := range parts {
		kv := strings.Split(v, sepKV)
		if len(kv) != 2 { //nolint: mnd
			return errors.New("invalid string format, should be 'key" + sepKV + "val" + sep + "key" + sepKV + "val'")
		}
		i, err := strconv.ParseFloat(kv[1], bitSize)
		if err != nil {
			return err
		}
		resultOut[kv[0]] = V(i)
	}

	*out = resultOut

	return
}

func SplitKeyValString[V ~string](s, sep, sepKV string, out *map[string]V) (err error) {
	if s == "" {
		*out = map[string]V{}
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make(map[string]V, len(parts))
	for _, v := range parts {
		kv := strings.Split(v, sepKV)
		if len(kv) != 2 { //nolint: mnd
			return errors.New("invalid string format, should be 'key" + sepKV + "val" + sep + "key" + sepKV + "val'")
		}
		resultOut[kv[0]] = V(kv[1])
	}

	*out = resultOut

	return
}

func SplitKeyValTime[V time.Time](s, sep, sepKV, layout string, out *map[string]V) (err error) {
	if s == "" {
		*out = map[string]V{}
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make(map[string]V, len(parts))
	for _, v := range parts {
		kv := strings.Split(v, sepKV)
		if len(kv) != 2 { //nolint: mnd
			return errors.New("invalid string format, should be 'key" + sepKV + "val" + sep + "key" + sepKV + "val'")
		}
		var t time.Time
		if err := ParseTime(layout, kv[1], &t); err != nil {
			return err
		}
		resultOut[kv[0]] = V(t)
	}

	*out = resultOut

	return
}

func SplitKeyValDuration[V time.Duration](s, sep, sepKV string, out *map[string]V) (err error) {
	if s == "" {
		*out = map[string]V{}
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make(map[string]V, len(parts))
	for _, v := range parts {
		kv := strings.Split(v, sepKV)
		if len(kv) != 2 { //nolint: mnd
			return errors.New("invalid string format, should be 'key" + sepKV + "val" + sep + "key" + sepKV + "val'")
		}
		var t time.Duration
		if err := ParseDuration(kv[1], &t); err != nil {
			return err
		}
		resultOut[kv[0]] = V(t)
	}

	*out = resultOut

	return
}

func SplitKeyValUUID[T any](s, sep, sepKV string, uuidParse UUIDParser[T], out *map[string]T) (err error) {
	if s == "" {
		*out = map[string]T{}
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make(map[string]T, len(parts))
	for _, v := range parts {
		kv := strings.Split(v, sepKV)
		if len(kv) != 2 { //nolint: mnd
			return errors.New("invalid string format, should be 'key" + sepKV + "val" + sep + "key" + sepKV + "val'")
		}
		var t T
		if err := ParseUUID(kv[1], uuidParse, &t); err != nil {
			return err
		}
		resultOut[kv[0]] = t
	}

	*out = resultOut

	return
}

func JoinKeyValInt[V constraints.Integer](values map[string]V, sep, sepKV string, base int) (result string) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			result += sep
		}
		result += k + sepKV + strconv.FormatInt(int64(values[k]), base)
	}
	return
}

func JoinKeyValUint[V constraints.Unsigned](values map[string]V, sep, sepKV string, base int) (result string) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			result += sep
		}
		result += k + sepKV + strconv.FormatUint(uint64(values[k]), base)
	}
	return
}

func JoinKeyValFloat[V constraints.Float](values map[string]V, sep, sepKV string, fmt byte, prec, bitSize int) (result string) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			result += sep
		}
		result += k + sepKV + strconv.FormatFloat(float64(values[k]), fmt, prec, bitSize)
	}
	return
}

func JoinKeyValString[V string](values map[string]V, sep, sepKV string) (result string) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			result += sep
		}
		result += k + sepKV + string(values[k])
	}
	return
}

func JoinKeyValTime[V time.Time](values map[string]V, sep, sepKV string, layout string) (result string) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			result += sep
		}
		result += k + sepKV + time.Time(values[k]).Format(layout)
	}
	return
}

func JoinKeyValDuration[V time.Duration](values map[string]V, sep, sepKV string) (result string) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			result += sep
		}
		result += k + sepKV + time.Duration(values[k]).String()
	}
	return
}

func JoinKeyValUUID[V uuid.UUID](values map[string]V, sep, sepKV string) (result string) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			result += sep
		}
		result += k + sepKV + uuid.UUID(values[k]).String()
	}
	return
}
