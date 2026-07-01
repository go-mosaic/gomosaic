// Package gomosaic предоставляет основу для генерации кода на основе Go-интерфейсов.
package gomosaic

import (
	"fmt"
	"go/token"
	"strconv"

	"github.com/fatih/structtag"

	"github.com/go-mosaic/gomosaic/v2/pkg/annotation"
)

// PackageInfo содержит информацию о пакете Go.
type PackageInfo struct {
	Name string
	Path string
}

// BasicKind описывает вид базового типа.
type BasicKind int

const (
	Invalid BasicKind = iota

	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	String
	UnsafePointer

	UntypedBool
	UntypedInt
	UntypedRune
	UntypedFloat
	UntypedComplex
	UntypedString
	UntypedNil

	Byte = Uint8
	Rune = Int32
)

// BasicInfo представляет собой набор флагов, описывающих свойства базового типа.
type BasicInfo int

const (
	IsBoolean BasicInfo = 1 << iota
	IsInteger
	IsUnsigned
	IsFloat
	IsComplex
	IsString
	IsUntyped

	IsOrdered   = IsInteger | IsFloat | IsString
	IsNumeric   = IsInteger | IsFloat | IsComplex
	IsConstType = IsBoolean | IsNumeric | IsString
)

// TypeInfo описывает тип поля или параметра.
type TypeInfo struct {
	Name           string
	Package        string
	BitSize        int
	IsBasic        bool
	BasicInfo      BasicInfo
	BasicKind      BasicKind
	IsAlias        bool
	IsPtr          bool
	IsSlice        bool
	IsArray        bool
	ArrayLen       int
	IsMap          bool
	IsChan         bool
	IsNamed        bool
	IsTypeParam    bool
	IsUnion        bool
	IsInstantiated bool
	KeyType        *TypeInfo
	ElemType       *TypeInfo
	Struct         *StructInfo
	Interface      *InterfaceInfo
	Signature      *SignatureInfo
	TypeParams     []*TypeInfo
	UnionTerms     []*TypeInfo
}

func (t *TypeInfo) String() string {
	var result string

	if t.IsPtr {
		result += "*"
	}
	if t.IsSlice {
		result += "[]"
	}
	if t.IsArray {
		result += fmt.Sprintf("[%d]", t.ArrayLen)
	}
	if t.IsMap {
		result += fmt.Sprintf("map[%s]", t.KeyType.String())
	}

	if t.Package != "" {
		result += t.Package + "."
	}
	result += t.Name

	if !t.IsInstantiated && len(t.TypeParams) > 0 {
		result += "["
		for i, param := range t.TypeParams {
			if i > 0 {
				result += ", "
			}
			result += param.String()
		}
		result += "]"
	}

	if t.IsInstantiated && len(t.TypeParams) > 0 {
		result += "["
		for i, arg := range t.TypeParams {
			if i > 0 {
				result += ", "
			}
			result += arg.String()
		}
		result += "]"
	}

	if t.IsUnion && len(t.UnionTerms) > 0 {
		result += "("
		for i, term := range t.UnionTerms {
			if i > 0 {
				result += " | "
			}
			result += term.String()
		}
		result += ")"
	}

	if t.ElemType != nil && !t.IsTypeParam && !t.IsInstantiated {
		result += t.ElemType.String()
	}

	if t.IsTypeParam && t.ElemType != nil {
		result += " " + t.ElemType.String()
	}

	return result
}

// AnnotationInfo содержит информацию об аннотации.
type AnnotationInfo struct {
	*annotation.Annotation
	Position *PosInfo
}

// Annotations представляет список аннотаций.
type Annotations []*AnnotationInfo

func (ts *Annotations) GetSlice(key string) (annotations []*AnnotationInfo) {
	for _, a := range *ts {
		if a.Key == key {
			annotations = append(annotations, a)
		}
	}
	return
}

func (ts *Annotations) Get(key string) (*AnnotationInfo, bool) {
	for _, a := range *ts {
		if a.Key == key {
			return a, true
		}
	}
	return nil, false
}

func (ts *Annotations) Has(key string) bool {
	_, ok := ts.Get(key)
	return ok
}

// NameTypeInfo содержит информацию об именованном типе.
type NameTypeInfo struct {
	Package     *PackageInfo
	Name        string
	Title       string
	Doc         string
	Pos         *PosInfo
	Type        *TypeInfo
	Annotations Annotations
	Methods     []*MethodInfo
}

// StructInfo содержит информацию о структуре.
type StructInfo struct {
	Fields []*VarInfo
}

// InterfaceInfo содержит информацию об интерфейсе.
type InterfaceInfo struct {
	Methods []*MethodInfo
}

// SignatureInfo содержит информацию о сигнатуре функции.
type SignatureInfo struct {
	Params     []*VarInfo
	Results    []*VarInfo
	TypeParams []*TypeInfo
}

// PosInfo содержит информацию о положении в файле.
type PosInfo struct {
	IsValid  bool
	Filename string
	Line     int
	Column   int
}

func (pos *PosInfo) String() string {
	s := pos.Filename
	if pos.IsValid {
		if s != "" {
			s += ":"
		}
		s += strconv.Itoa(pos.Line)
		if pos.Column != 0 {
			s += fmt.Sprintf(":%d", pos.Column)
		}
	}
	if s == "" {
		s = "-"
	}
	return s
}

// MethodInfo содержит информацию о методе.
type MethodInfo struct {
	Name         string
	FullName     string
	ShortName    string
	Params       []*VarInfo
	Results      []*VarInfo
	Title        string
	Doc          string
	Pos          *PosInfo
	Annotations  Annotations
	ReturnValues []*TypeAndValueInfo
}

// VarInfo содержит информацию о переменной (параметре или поле).
type VarInfo struct {
	Package     *PackageInfo
	Name        string
	Type        *TypeInfo
	Title       string
	Doc         string
	Pos         *PosInfo
	IsContext   bool
	IsError     bool
	Annotations Annotations
	Tags        *structtag.Tags
}

// ValueKind описывает вид значения.
type ValueKind int

const (
	UnknownValueKind ValueKind = iota
	BoolValueKind
	StringValueKind
	IntValueKind
	FloatValueKind
	ComplexValueKind
)

// TypeAndValueInfo содержит информацию о типе и значении.
type TypeAndValueInfo struct {
	Value string
	Kind  ValueKind
}

// CommentInfo содержит информацию о комментарии.
type CommentInfo struct {
	Value        string
	IsTitle      bool
	IsAnnotation bool
	Position     token.Position
}
