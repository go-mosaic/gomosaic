package option

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-mosaic/runtime"
	"github.com/hashicorp/go-multierror"
	"github.com/vmihailenco/tagparser/v2"
	"golang.org/x/exp/constraints"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "json: Unmarshal(nil)"
	}
	if e.Type.Kind() != reflect.Pointer {
		return "json: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "json: Unmarshal(nil " + e.Type.String() + ")"
}

type decodeState struct {
	annotations gomosaic.Annotations
	fieldTag    map[string]*gomosaic.AnnotationInfo
	errs        error
}

func (d *decodeState) init(annotations gomosaic.Annotations) {
	d.annotations = annotations
	d.fieldTag = make(map[string]*gomosaic.AnnotationInfo, 512) //nolint: mnd
}

func Unmarshal(prefix string, annotations gomosaic.Annotations, v any) error {
	var d decodeState
	d.init(annotations)

	rv := reflect.ValueOf(v)
	if (rv.Kind() != reflect.Pointer || rv.IsNil()) && rv.Kind() != reflect.Struct {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}
	t := reflect.TypeOf(v)

	if err := d.unmarshal(t.Elem().Name(), prefix, t, rv); err != nil {
		return err
	}

	if d.errs != nil {
		return d.errs
	}

	return nil
}

func (d *decodeState) unmarshal(path, prefix string, t reflect.Type, rv reflect.Value) error {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	rv = reflect.Indirect(rv)

	for i := range rv.NumField() {
		fieldType := t.Field(i)
		name, options, ok := parseTag(fieldType)
		if !ok || name == "" {
			continue
		}
		nameWithPrefix := prefix + "-" + name

		t, tagExists := d.annotations.Get(nameWithPrefix)
		if tagExists {
			d.fieldTag[path+"."+fieldType.Name] = t
		}

		switch fieldType.Type.Kind() {
		default:
			if tagExists {
				if slices.Contains(options, "asFlag") {
					rv.Field(i).Set(reflect.ValueOf(true))
				} else {
					fieldValue, err := parseValue(fieldType.Type, t.Value())
					if err != nil {
						return err
					}
					rv.Field(i).Set(reflect.ValueOf(fieldValue))
				}

				d.validateValue(rv.Field(i), fieldType, t)
			}
		case reflect.Struct:
			if hasInlineOption(options) {
				if tagExists {
					newVal, err := d.newInlineElem(path+"."+fieldType.Name, t, fieldType.Type)
					if err != nil {
						return err
					}
					rv.Field(i).Set(newVal)
				}
			} else if err := d.unmarshal(path+"."+fieldType.Name, nameWithPrefix, fieldType.Type, rv.Field(i)); err != nil {
				return err
			}
		case reflect.Slice:
			switch fieldType.Type.Elem().Kind() {
			default:
				if tagExists {
					options := []string{}
					options = append(options, t.Options...)
					fieldValue, err := parseValues(fieldType.Type.Elem(), options)
					if err != nil {
						return err
					}
					rv.Field(i).Set(reflect.ValueOf(fieldValue))
					d.validateValue(rv.Field(i), fieldType, t)
				}
			case reflect.Struct:
				if hasInlineOption(options) {
					if err := d.unmarshalInline(path+"."+fieldType.Name, nameWithPrefix, fieldType.Type.Elem(), rv.Field(i)); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (d *decodeState) validateValue(v reflect.Value, field reflect.StructField, t *gomosaic.AnnotationInfo) {
	defaultValue, _ := field.Tag.Lookup("default")

	validTag, ok := field.Tag.Lookup("valid")
	if !ok {
		return
	}
	if tag := tagparser.Parse(validTag); tag != nil {
		switch tag.Name {
		case "required":
			if v.IsZero() {
				d.errs = multierror.Append(d.errs, fmt.Errorf("%s is required: %s", t.Key, t.Position))
			}
		case "in":
			value := v.Interface()
			if v.IsZero() {
				value = defaultValue
			}

			params := strings.Split(tag.Options["params"], " ")
			if !isIn(value, params...) {
				d.errs = multierror.Append(d.errs, fmt.Errorf("%s valid only params (%s): %s", t.Key, tag.Options["params"], t.Position))
			}
		}
	}
}

func (d *decodeState) unmarshalInline(path string, nameWithPrefix string, t reflect.Type, rv reflect.Value) error {
	newSlice := reflect.MakeSlice(reflect.SliceOf(t), 0, 10) //nolint: mnd
	for i, tag := range d.annotations.GetSlice(nameWithPrefix) {
		newVal, err := d.newInlineElem(fmt.Sprintf("%s[%d]", path, i), tag, t)
		if err != nil {
			return err
		}
		newSlice = reflect.Append(newSlice, newVal)
	}
	rv.Set(newSlice)

	return nil
}

func (d *decodeState) newInlineElem(path string, tag *gomosaic.AnnotationInfo, t reflect.Type) (reflect.Value, error) {
	newVal := reflect.New(t).Elem()
	for j := range t.NumField() {
		name, options, ok := parseTag(t.Field(j))
		if !ok {
			continue
		}

		optionMap := make(map[string]struct{})
		for _, o := range options {
			optionMap[o] = struct{}{}
		}

		var (
			v   any
			err error
		)
		if _, ok := optionMap["fromParam"]; ok {
			v, err = parseValue(t.Field(j).Type, tag.Params[name])
			if err != nil {
				return reflect.Value{}, err
			}
		} else if _, ok := optionMap["fromOptions"]; ok {
			v = tag.Options
		} else if _, ok := optionMap["fromOption"]; ok {
			if slices.Contains(tag.Options, name) {
				v = true
			}
		} else if _, ok := optionMap["fromValue"]; ok {
			v, err = parseValue(t.Field(j).Type, tag.Value())
			if err != nil {
				return reflect.Value{}, err
			}
		}

		fieldValue := newVal.FieldByName(t.Field(j).Name)

		if v != nil {
			fieldValue.Set(reflect.ValueOf(v))

			d.fieldTag[path+"."+t.Field(j).Name] = tag
		}

		d.validateValue(fieldValue, t.Field(j), tag)
	}

	return newVal, nil
}

func parseValues(t reflect.Type, elems []string) (any, error) {
	switch t.Kind() {
	default:
		return nil, &InvalidUnmarshalError{Type: t}
	case reflect.String:
		return elems, nil
	case reflect.Int:
		return parseIntValues[int](elems, 64) //nolint: mnd
	case reflect.Int8:
		return parseIntValues[int8](elems, 8) //nolint: mnd
	case reflect.Int16:
		return parseIntValues[int16](elems, 16) //nolint: mnd
	case reflect.Int32:
		return parseIntValues[int32](elems, 32) //nolint: mnd
	case reflect.Int64:
		return parseIntValues[int64](elems, 64) //nolint: mnd
	case reflect.Uint:
		return parseUintValues[uint](elems, 64) //nolint: mnd
	case reflect.Uint8:
		return parseUintValues[uint8](elems, 8) //nolint: mnd
	case reflect.Uint16:
		return parseUintValues[uint16](elems, 16) //nolint: mnd
	case reflect.Uint32:
		return parseUintValues[uint32](elems, 32) //nolint: mnd
	case reflect.Uint64:
		return parseUintValues[uint64](elems, 64) //nolint: mnd
	case reflect.Float32:
		return parseFloatValues[float32](elems, 32) //nolint: mnd
	case reflect.Float64:
		return parseFloatValues[float64](elems, 64) //nolint: mnd
	}
}

func parseFloatValues[T constraints.Float](elems []string, bitSize int) (any, error) {
	var values []T
	for _, s := range elems {
		var v T
		err := runtime.ParseFloat(s, bitSize, &v)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

func parseIntValues[T constraints.Signed](elems []string, bitSize int) (any, error) {
	var values []T
	for _, s := range elems {
		var v T
		err := runtime.ParseInt(s, 10, bitSize, &v) //nolint: mnd
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

func parseUintValues[T constraints.Unsigned](elems []string, bitSize int) (any, error) {
	var values []T
	for _, s := range elems {
		var v T
		err := runtime.ParseUint(s, 10, bitSize, &v) //nolint: mnd
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

func parseValue(t reflect.Type, s string) (any, error) {
	switch t.Kind() {
	default:
		return nil, &InvalidUnmarshalError{}
	case reflect.Int, reflect.Int64:
		var i int
		return i, runtime.ParseInt(s, 10, 64, &i) //nolint: mnd
	case reflect.Int8:
		var i int8
		return i, runtime.ParseInt(s, 10, 8, &i) //nolint: mnd
	case reflect.Int16:
		var i int16
		return i, runtime.ParseInt(s, 10, 16, &i) //nolint: mnd
	case reflect.Int32:
		var i int32
		return i, runtime.ParseInt(s, 10, 32, &i) //nolint: mnd
	case reflect.Uint, reflect.Uint64:
		var i uint
		return i, runtime.ParseUint(s, 10, 64, &i) //nolint: mnd
	case reflect.Uint8:
		var i uint8
		return i, runtime.ParseUint(s, 10, 8, &i) //nolint: mnd
	case reflect.Uint16:
		var i uint16
		return i, runtime.ParseUint(s, 10, 16, &i) //nolint: mnd
	case reflect.Uint32:
		var i uint32
		return i, runtime.ParseUint(s, 10, 32, &i) //nolint: mnd
	case reflect.Float32:
		var i float32
		return i, runtime.ParseFloat(s, 32, &i) //nolint: mnd
	case reflect.Float64:
		var i float64
		return i, runtime.ParseFloat(s, 64, &i) //nolint: mnd
	case reflect.String:
		return s, nil
	case reflect.Bool:
		var b bool
		if s == "" {
			return false, nil
		}
		return b, runtime.ParseBool(s, &b)
	}
}
