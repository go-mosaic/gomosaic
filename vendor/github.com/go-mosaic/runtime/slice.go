package runtime

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/constraints"
)

func Split(s, sep string, recv *[]string) (err error) {
	if s == "" {
		return nil
	}
	*recv = strings.Split(s, sep)
	return nil
}

func SplitInt[V constraints.Signed](s, sep string, base, bitSize int, out *[]V) (err error) {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make([]V, len(parts))
	for idx, v := range parts {
		var i V
		if err := ParseInt[V](v, base, bitSize, &i); err != nil {
			return fmt.Errorf("parsing error idx %d value %s: %w", idx, v, err)
		}
		resultOut[idx] = i
	}

	*out = resultOut

	return
}

func SplitUint[V constraints.Unsigned](s, sep string, base, bitSize int, out *[]V) (err error) {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make([]V, len(parts))
	for idx, v := range parts {
		var i V
		err := ParseUint[V](v, base, bitSize, &i)
		if err != nil {
			return fmt.Errorf("parsing error idx %d value %s: %w", idx, v, err)
		}
		resultOut[idx] = i
	}

	*out = resultOut

	return
}

func SplitFloat[V constraints.Float](s, sep string, bitSize int, out *[]V) (err error) {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make([]V, len(parts))
	for idx, v := range parts {
		var i V
		err := ParseFloat[V](v, bitSize, &i)
		if err != nil {
			return fmt.Errorf("parsing error idx %d value %s: %w", idx, v, err)
		}
		resultOut[idx] = i
	}

	*out = resultOut

	return
}

func SplitTime(s, sep, sepKV, layout string, out *[]time.Time) (err error) {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make([]time.Time, len(parts))
	for idx, v := range parts {
		var t time.Time
		if err := ParseTime(layout, v, &t); err != nil {
			return err
		}
		resultOut[idx] = t
	}

	*out = resultOut

	return
}

func SplitDuration(s, sep string, out *[]time.Duration) (err error) {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make([]time.Duration, len(parts))
	for idx, v := range parts {
		var t time.Duration
		if err := ParseDuration(v, &t); err != nil {
			return err
		}
		resultOut[idx] = t
	}

	*out = resultOut

	return
}

func SplitUUID[T any](s, sep string, parseUUID UUIDParser[T], out *[]T) (err error) {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	resultOut := make([]T, len(parts))
	for idx, v := range parts {
		var t T
		if err := ParseUUID(v, parseUUID, &t); err != nil {
			return err
		}
		resultOut[idx] = t
	}

	*out = resultOut

	return
}

func JoinInt[V constraints.Integer](values []V, sep string, base int) (result string) {
	for i, v := range values {
		if i > 0 {
			result += sep
		}
		result += strconv.FormatInt(int64(v), base)
	}
	return
}

func JoinFloat[V constraints.Float](values []V, sep string, fmt byte, prec int, bitSize int) (result string) {
	for i, v := range values {
		if i > 0 {
			result += sep
		}
		result += strconv.FormatFloat(float64(v), fmt, prec, bitSize)
	}
	return
}
