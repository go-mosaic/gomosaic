// Package option предоставляет парсер опций из аннотаций.
package option

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// Unmarshal заполняет структуру opts значениями из аннотаций с префиксом prefix.
func Unmarshal(prefix string, annotations gomosaic.Annotations, opts interface{}) error {
	v := reflect.ValueOf(opts)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("opts должен быть указателем на структуру")
	}
	v = v.Elem()
	t := v.Type()

	for _, ann := range annotations {
		if !strings.HasPrefix(ann.Key, prefix+"-") {
			continue
		}
		optionName := strings.TrimPrefix(ann.Key, prefix+"-")
		unmarshalStruct(v, t, ann, optionName)
	}
	return nil
}

func unmarshalStruct(v reflect.Value, t reflect.Type, ann *gomosaic.AnnotationInfo, optionName string) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("option")
		if tag == "" {
			continue
		}

		tp := parseTag(tag)

		// Для inline-полей — обрабатываем аннотацию на вложенной структуре
		if tp.inline && tp.name == "" {
			fv := v.Field(i)
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					fv.Set(reflect.New(fv.Type().Elem()))
				}
				fv = fv.Elem()
			}
			if fv.Kind() == reflect.Struct {
				if err := unmarshalStruct(fv, fv.Type(), ann, optionName); err != nil {
					return err
				}
			}
			continue
		}

		if tp.name != optionName {
			continue
		}

		fv := v.Field(i)
		if err := setFieldValue(fv, ann, tp); err != nil {
			return fmt.Errorf("поле %s.%s: %w", t.Name(), field.Name, err)
		}
	}
	return nil
}

type tagParts struct {
	name       string
	fromValue  bool
	fromOption bool
	fromParam  bool
	asFlag     bool
	inline     bool
}

func parseTag(tag string) tagParts {
	parts := strings.Split(tag, ",")
	tp := tagParts{}
	for i, p := range parts {
		switch {
		case i == 0 && p != "":
			tp.name = p
		case p == "fromValue":
			tp.fromValue = true
		case p == "fromOption":
			tp.fromOption = true
		case p == "fromParam":
			tp.fromParam = true
		case p == "asFlag":
			tp.asFlag = true
		case p == "inline":
			tp.inline = true
		}
	}
	return tp
}

func setFieldValue(field reflect.Value, ann *gomosaic.AnnotationInfo, tp tagParts) error {
	if !field.CanSet() {
		return nil
	}

	if tp.fromParam {
		// Читаем из именованных параметров аннотации
		return setFieldFromParams(field, ann)
	}

	switch field.Kind() {
	case reflect.String:
		if tp.asFlag {
			field.SetString("true")
		} else if tp.fromValue && len(ann.Params) > 0 {
			field.SetString(ann.Params[0])
		} else if len(ann.Params) > 0 {
			field.SetString(strings.Join(ann.Params, " "))
		}
	case reflect.Bool:
		// asFlag: устанавливаем true только если нет позиционных параметров
		// (чисто флаговая аннотация типа @http-required)
		if tp.asFlag && len(ann.Params) == 0 && len(ann.NamedParams) == 0 {
			field.SetBool(true)
		} else if !tp.asFlag && len(ann.Params) > 0 {
			v, err := strconv.ParseBool(ann.Params[0])
			if err != nil {
				return err
			}
			field.SetBool(v)
		}
	case reflect.Int:
		if len(ann.Params) > 0 {
			v, err := strconv.Atoi(ann.Params[0])
			if err != nil {
				return err
			}
			field.SetInt(int64(v))
		}
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			if tp.fromOption {
				// Для fromOption собираем значения из аннотаций
				vals := make([]string, 0)
				for _, p := range ann.Params {
					vals = append(vals, p)
				}
				field.Set(reflect.ValueOf(vals))
			} else {
				field.Set(reflect.ValueOf(ann.Params))
			}
		}
	case reflect.Ptr:
		// Для указателей создаём экземпляр и заполняем
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldValue(field.Elem(), ann, tp)
	case reflect.Struct:
		// Для вложенных структур
		if tp.inline {
			return unmarshalStruct(field, field.Type(), ann, "")
		}
	}
	return nil
}

func setFieldFromParams(field reflect.Value, ann *gomosaic.AnnotationInfo) error {
	if field.Kind() != reflect.String {
		return nil
	}
	// Для строковых полей ищем первый именованный параметр
	for _, val := range ann.NamedParams {
		field.SetString(val)
		return nil
	}
	return nil
}
