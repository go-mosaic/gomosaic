package server

import (
	"strings"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// isStreamType проверяет, является ли параметр стрим-типом.
func isStreamType(v *gomosaic.VarInfo) bool {
	name := v.Type.Name
	return strings.Contains(name, "Server") || strings.Contains(name, "Client")
}

// getBaseTypeName возвращает имя базового типа, игнорируя указатели и другие модификаторы.
func getBaseTypeName(t *gomosaic.TypeInfo) string {
	if t == nil {
		return ""
	}
	if t.IsPtr && t.ElemType != nil {
		return getBaseTypeName(t.ElemType)
	}
	if t.IsSlice && t.ElemType != nil {
		return getBaseTypeName(t.ElemType)
	}
	if t.IsArray && t.ElemType != nil {
		return getBaseTypeName(t.ElemType)
	}
	if t.IsMap && t.ElemType != nil {
		return getBaseTypeName(t.ElemType)
	}
	return t.Name
}
