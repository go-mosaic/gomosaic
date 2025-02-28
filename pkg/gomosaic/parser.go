package gomosaic

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
	"golang.org/x/tools/go/packages"

	"github.com/go-mosaic/gomosaic/pkg/annotation"
)

// PackageInfo информация о пакете
type PackageInfo struct {
	Name string // Имя пакета
	Path string // Путь пакета (например, "github.com/go-mosaic/gomosaic/pkg")
}

// BasicKind описывает вид базового типа.
type BasicKind int

const (
	Invalid BasicKind = iota // неверный тип

	// предварительно объявленные типы
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

	// типы для нетипизированных значений
	UntypedBool
	UntypedInt
	UntypedRune
	UntypedFloat
	UntypedComplex
	UntypedString
	UntypedNil

	// псевдонимы
	Byte = Uint8
	Rune = Int32
)

// BasicInfo представляет собой набор флагов, описывающих свойства базового типа.
type BasicInfo int

// Свойства базовых типов.
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

// TypeInfo описывает тип поля или параметра
type TypeInfo struct {
	Name      string         // Имя типа (например, "int", "MyStruct")
	Package   string         // Пакет типа (например, "github.com/user/project/pkg")
	BitSize   int            // представляет собой набор флагов, описывающих свойства базового типа.
	IsBasic   bool           // Является ли тип базовым
	BasicInfo BasicInfo      // вид базового типа.
	BasicKind BasicKind      // свойства базового типа.
	IsPtr     bool           // Является ли тип указателем.
	IsSlice   bool           // Является ли тип слайсом.
	IsArray   bool           // Является ли тип массивом.
	ArrayLen  int            // Длина массива (если IsArray == true).
	IsMap     bool           // Является ли тип мапой.
	IsChan    bool           // Является ли тип каналом.
	KeyType   *TypeInfo      // Тип ключа (если IsMap == true, может быть nil).
	ElemType  *TypeInfo      // Тип элемента (для каналов, слайсов, массивов, мап и указателей, может быть nil).
	Named     *TypeInfo      // Тип является именованым (например type Name <тип>, может быть nil).
	Struct    *StructInfo    // Тип структуры (может быть nil).
	Interface *InterfaceInfo // Тип интерфейса (может быть nil).
	Signature *SignatureInfo // Тип сигнатуры функции (может быть nil).
}

// String возвращает строковое представление типа
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

	if t.ElemType != nil {
		result += t.ElemType.String()
	}

	return result
}

type AnnotationInfo struct {
	*annotation.Annotation
	Position *PosInfo
}

type Annotations []*AnnotationInfo

func (ts *Annotations) GetSlice(key string) (annotations []*AnnotationInfo) {
	for _, annotation := range *ts {
		if annotation.Key == key {
			annotations = append(annotations, annotation)
		}
	}
	return
}

func (ts *Annotations) Get(key string) (*AnnotationInfo, bool) {
	for _, annotation := range *ts {
		if annotation.Key == key {
			return annotation, true
		}
	}
	return nil, false
}

func (ts *Annotations) Has(key string) bool {
	_, ok := ts.Get(key)
	return ok
}

// TypeInfo информация о типе
type NameTypeInfo struct {
	Package     *PackageInfo  // Информация о пакете
	Name        string        // Имя типа
	Title       string        // Заголовок
	Doc         string        // Документация (комментарии)
	Pos         *PosInfo      // Позиция в файле
	Type        *TypeInfo     // Тип
	Annotations Annotations   // Аннотации
	Methods     []*MethodInfo // Методы
}

// StructInfo информация о структуре
type StructInfo struct {
	Fields []*VarInfo // Поля (для структур)
}

// InterfaceInfo информация о интерфейсе
type InterfaceInfo struct {
	Methods []*MethodInfo // Методы
}

// SignatureInfo информация о сигнатуре функции
type SignatureInfo struct {
	Params  []*VarInfo
	Results []*VarInfo
}

// PosInfo информация о положении типа в файле
type PosInfo struct {
	IsValid  bool
	Filename string // имя файла если есть
	Line     int    // номер строки
	Column   int    // номе колонки
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

// MethodInfo информация о методе
type MethodInfo struct {
	Name         string      // Имя метода
	FullName     string      // Имя метода полное (например: )
	ShortName    string      // Имя метода сокращенное (например: )
	Params       []*VarInfo  // Параметры метода
	Results      []*VarInfo  // Возвращаемые значения
	Title        string      // Заголовок
	Doc          string      // Документация (комментарии)
	Pos          *PosInfo    // Позиция в файле
	Annotations  Annotations // Аннотации
	ReturnValues []*TypeAndValueInfo
}

// VarInfo информация о параметре метода либо поле структуры
type VarInfo struct {
	Package     *PackageInfo // Информация о пакете
	Name        string       // Имя параметра
	Type        *TypeInfo    // Тип
	Title       string       // Заголовок
	Doc         string       // Документация (комментарии)
	Pos         *PosInfo     // Позиция в файле
	IsContext   bool         // Является типом context.Context
	IsError     bool         // Является типом error
	Annotations Annotations  // Аннотации
	Tags        *structtag.Tags
}

// ValueKind описывает вид значении возвращаемом через return
type ValueKind int

const (
	UnknownValueKind ValueKind = iota

	BoolValueKind
	StringValueKind
	IntValueKind
	FloatValueKind
	ComplexValueKind
)

// TypeAndValueInfo информация о значении возвращаемом через return
type TypeAndValueInfo struct {
	Value string
	Kind  ValueKind
}

// CommentInfo информация о коментарии
type CommentInfo struct {
	Value        string
	IsTitle      bool
	IsAnnotation bool
	Position     token.Position
}

// ParsePackage парсит пакет и возвращает информацию о типах
func ParsePackage(dir string, patterns []string) (nameTypesInfo []*NameTypeInfo, err error) {
	for i := range patterns {
		patterns[i] = "pattern=" + patterns[i]
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}

	nameTypesInfo = make([]*NameTypeInfo, 0, 1024) //nolint: mnd

	for _, pkg := range pkgs {
		returnValues := parseReturnValues(pkg, pkg.Syntax)

		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if !obj.Exported() {
				continue
			}

			named, ok := obj.Type().(*types.Named)
			if !ok {
				continue
			}

			title, doc, annotations, err := findDocAndAnnotations(pkg, named.Obj().Name(), named.Obj().Pos())
			if err != nil {
				return nil, err
			}

			if !annotations.Has("gomosaic") {
				continue
			}

			typeInfo, err := typeToTypeInfo(pkg, obj.Type().Underlying())
			if err != nil {
				return nil, err
			}

			nameTypeInfo := &NameTypeInfo{
				Package:     packageToPackageInfo(named.Obj().Pkg()),
				Name:        named.Obj().Name(),
				Title:       title,
				Doc:         doc,
				Pos:         parsePosition(pkg.Fset.Position(obj.Pos())),
				Annotations: annotations,
				Type:        typeInfo,
			}

			for i := range named.NumMethods() {
				method := named.Method(i)
				if !method.Exported() {
					continue
				}

				methodInfo, err := funcToMethodInfo(pkg, method)
				if err != nil {
					return nil, err
				}

				if values, ok := returnValues[method.FullName()]; ok {
					methodInfo.ReturnValues = values
				}

				nameTypeInfo.Methods = append(nameTypeInfo.Methods, methodInfo)
			}

			nameTypesInfo = append(nameTypesInfo, nameTypeInfo)
		}
	}

	return nameTypesInfo, nil
}

// varToVarInfo преобразует types.Var в VarInfo
func varToVarInfo(pkg *packages.Package, v *types.Var) (*VarInfo, error) {
	title, doc, annotations, err := findDocAndAnnotations(pkg, v.Name(), v.Pos())
	if err != nil {
		return nil, err
	}

	typeInfo, err := typeToTypeInfo(pkg, v.Type())
	if err != nil {
		return nil, err
	}

	var packageInfo *PackageInfo
	if v.Pkg() != nil {
		packageInfo = packageToPackageInfo(v.Pkg())
	}

	return &VarInfo{
		Package:     packageInfo,
		Name:        v.Name(),
		Type:        typeInfo,
		IsContext:   isContext(typeInfo),
		IsError:     isError(typeInfo),
		Pos:         parsePosition(pkg.Fset.Position(v.Pos())),
		Title:       title,
		Doc:         doc,
		Annotations: annotations,
	}, nil
}

// funcToMethodInfo преобразует types.Func в MethodInfo
func funcToMethodInfo(pkg *packages.Package, method *types.Func) (*MethodInfo, error) {
	title, doc, annotations, err := findDocAndAnnotations(pkg, method.Name(), method.Pos())
	if err != nil {
		return nil, err
	}
	methodInfo := &MethodInfo{
		Name:        method.Name(),
		FullName:    method.FullName(),
		Params:      make([]*VarInfo, 0),
		Results:     make([]*VarInfo, 0),
		Title:       title,
		Doc:         doc,
		Pos:         parsePosition(pkg.Fset.Position(method.Pos())),
		Annotations: annotations,
	}

	if sig := method.Signature(); sig != nil {
		methodInfo.ShortName = method.Name()
		if named, ok := sig.Recv().Type().(*types.Named); ok {
			name := named.Obj().Name()
			if named.Obj().Pkg() != nil {
				name = named.Obj().Pkg().Name() + "." + name
			}
			methodInfo.ShortName = "(" + name + ")." + method.Name()
		}

		paramVarsInfo, err := tuplesToVarsInfo(pkg, sig.Params())
		if err != nil {
			return nil, err
		}

		methodInfo.Params = paramVarsInfo

		resultVarsInfo, err := tuplesToVarsInfo(pkg, sig.Results())
		if err != nil {
			return nil, err
		}

		methodInfo.Results = resultVarsInfo
	}

	return methodInfo, nil
}

// tuplesToVarsInfo преобразует types.Tuple в []VarInfo
func tuplesToVarsInfo(pkg *packages.Package, tuple *types.Tuple) (varsInfo []*VarInfo, err error) {
	for i := range tuple.Len() {
		v := tuple.At(i)
		varInfo, err := varToVarInfo(pkg, v)
		if err != nil {
			return nil, err
		}
		varsInfo = append(varsInfo, varInfo)
	}

	return varsInfo, nil
}

// packageToPackageInfo преобразует types.Package в PackageInfo
func packageToPackageInfo(pkg *types.Package) *PackageInfo {
	return &PackageInfo{
		Name: pkg.Name(),
		Path: pkg.Path(),
	}
}

// typeToTypeInfo преобразует types.Type в TypeInfo
func typeToTypeInfo(pkg *packages.Package, t types.Type) (*TypeInfo, error) {
	typeInfo := &TypeInfo{}

	switch t := t.(type) {
	case *types.Basic:
		typeInfo.Name = t.Name()
		typeInfo.IsBasic = true
		typeInfo.BasicInfo = BasicInfo(t.Info())
		typeInfo.BasicKind = BasicKind(t.Kind())

		switch t.Kind() {
		case types.Int8, types.Uint8:
			typeInfo.BitSize = 8
		case types.Int16, types.Uint16:
			typeInfo.BitSize = 16
		case types.Int32, types.Float32, types.Uint32:
			typeInfo.BitSize = 32
		default: // для types.Int, types.Uint, types.Float64, types.Uint64, types.Int64 и других.
			typeInfo.BitSize = 64
		}

	case *types.Chan:
		typeInfo.Name = "chan"
		typeInfo.IsChan = true
		if t.Dir() == types.SendOnly {
			typeInfo.Name += "<-"
		} else {
			typeInfo.Name = "<-" + typeInfo.Name
		}
		elemType, err := typeToTypeInfo(pkg, t.Elem())
		if err != nil {
			return nil, err
		}
		typeInfo.ElemType = elemType
	case *types.Pointer:
		typeInfo.IsPtr = true
		elemType, err := typeToTypeInfo(pkg, t.Elem())
		if err != nil {
			return nil, err
		}
		typeInfo.ElemType = elemType
	case *types.Slice:
		typeInfo.IsSlice = true
		elemType, err := typeToTypeInfo(pkg, t.Elem())
		if err != nil {
			return nil, err
		}
		typeInfo.ElemType = elemType
	case *types.Array:
		typeInfo.IsArray = true
		typeInfo.ArrayLen = int(t.Len())
		elemType, err := typeToTypeInfo(pkg, t.Elem())
		if err != nil {
			return nil, err
		}
		typeInfo.ElemType = elemType
	case *types.Map:
		typeInfo.IsMap = true
		keyType, err := typeToTypeInfo(pkg, t.Key())
		if err != nil {
			return nil, err
		}
		typeInfo.KeyType = keyType
		elemType, err := typeToTypeInfo(pkg, t.Elem())
		if err != nil {
			return nil, err
		}
		typeInfo.ElemType = elemType
	case *types.Named:
		typeInfo.Name = t.Obj().Name()
		if pkg := t.Obj().Pkg(); pkg != nil {
			typeInfo.Package = pkg.Path()
		}
		if t.Obj().Type() != nil {
			named, err := typeToTypeInfo(pkg, t.Obj().Type().Underlying())
			if err != nil {
				return nil, err
			}
			typeInfo.Named = named
		}
	case *types.Struct:
		typeInfo.Name = "struct"

		structInfo := &StructInfo{
			Fields: make([]*VarInfo, 0, 64), //nolint: mnd
		}

		// Обработка полей структуры
		for i := range t.NumFields() {
			field := t.Field(i)
			if !field.Exported() {
				continue
			}

			varInfo, err := varToVarInfo(pkg, field)
			if err != nil {
				return nil, err
			}

			if tags, err := structtag.Parse(t.Tag(i)); err == nil {
				varInfo.Tags = tags
			}

			structInfo.Fields = append(structInfo.Fields, varInfo)
		}
		typeInfo.Struct = structInfo
	case *types.Interface:
		typeInfo.Name = "interface"

		interfaceInfo := &InterfaceInfo{}

		// Обработка методов интерфейса
		for i := range t.NumMethods() {
			method := t.Method(i)
			if !method.Exported() {
				continue
			}

			methodInfo, err := funcToMethodInfo(pkg, method)
			if err != nil {
				return nil, err
			}

			interfaceInfo.Methods = append(interfaceInfo.Methods, methodInfo)
		}
		typeInfo.Interface = interfaceInfo
	case *types.Signature:
		typeInfo.Name = "func"

		paramVarsInfo, err := tuplesToVarsInfo(pkg, t.Params())
		if err != nil {
			return nil, err
		}

		resultVarsInfo, err := tuplesToVarsInfo(pkg, t.Results())
		if err != nil {
			return nil, err
		}

		typeInfo.Signature = &SignatureInfo{
			Params:  paramVarsInfo,
			Results: resultVarsInfo,
		}
	}

	return typeInfo, nil
}

// parseReturnValues ищет возвращаемые значения базового типа в функциях и мапит их на полное имя функции или метода структуры.
func parseReturnValues(pkg *packages.Package, files []*ast.File) (returnValues map[string][]*TypeAndValueInfo) {
	returnValues = make(map[string][]*TypeAndValueInfo, 128) //nolint: mnd

	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			fnDecl, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			obj := pkg.TypesInfo.ObjectOf(fnDecl.Name)
			if obj == nil {
				return true
			}

			fn, ok := obj.(*types.Func)
			if !ok {
				return true
			}

			if !fn.Exported() {
				return true
			}

			key := fn.FullName()

			ast.Inspect(fnDecl.Body, func(n ast.Node) bool {
				ret, ok := n.(*ast.ReturnStmt)
				if !ok {
					return true
				}

				typeAndValues := make([]*TypeAndValueInfo, 0, len(ret.Results))
				for _, result := range ret.Results {
					if tv, ok := pkg.TypesInfo.Types[result]; ok && tv.Value != nil {
						typeAndValues = append(typeAndValues, &TypeAndValueInfo{
							Value: tv.Value.String(),
							Kind:  parseKind(tv.Value.Kind()),
						})
					}
				}

				returnValues[key] = typeAndValues

				return true
			})

			return true
		})
	}

	return returnValues
}

// findDocAndAnnotations находит аннотации, заголовок и описание для поля, структуры, метода по позиции в AST
func findDocAndAnnotations(pkg *packages.Package, name string, pos token.Pos) (title, description string, annotations Annotations, err error) {
	var annotationComments []*CommentInfo
	allComments := findComments(pkg, name, pos)
	for _, comment := range allComments {
		switch {
		default:
			description += comment.Value + "\n"
		case comment.IsAnnotation:
			annotationComments = append(annotationComments, comment)
		case comment.IsTitle:
			title = comment.Value
		}
	}
	if len(annotationComments) > 0 {
		annotations, err = ParseAnnotations(annotationComments)
		if err != nil {
			return
		}
	}
	return
}

// findComments находит коментарии для поля, структуры, метода по позиции в AST
func findComments(pkg *packages.Package, name string, pos token.Pos) (commentsInfo []*CommentInfo) {
	position := pkg.Fset.Position(pos)

	for _, file := range pkg.Syntax {
		for _, commentGroup := range file.Comments {
			cg := pkg.Fset.Position(commentGroup.End())
			if cg.Line == position.Line-1 && cg.Filename == position.Filename {
				for _, comment := range commentGroup.List {
					text := strings.TrimLeft(strings.TrimLeft(comment.Text, "/"), " ")
					isTitle := strings.HasPrefix(text, name)
					isAnnotation := strings.HasPrefix(text, "@")
					if isTitle {
						text = strings.ReplaceAll(text, name+" ", "")
					}
					commentsInfo = append(commentsInfo, &CommentInfo{
						Value:        text,
						IsTitle:      isTitle,
						IsAnnotation: isAnnotation,
						Position:     pkg.Fset.Position(comment.End()),
					})
				}
			}
		}
	}

	return commentsInfo
}

func parseKind(kind constant.Kind) ValueKind {
	switch kind {
	default:
		return UnknownValueKind
	case constant.String:
		return StringValueKind
	case constant.Bool:
		return BoolValueKind
	case constant.Float:
		return FloatValueKind
	case constant.Complex:
		return ComplexValueKind
	}
}

func parsePosition(p token.Position) *PosInfo {
	return &PosInfo{
		IsValid:  p.IsValid(),
		Filename: p.Filename,
		Line:     p.Line,
		Column:   p.Column,
	}
}

func isContext(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "context" && typeInfo.Name == "Context"
}

func isError(typeInfo *TypeInfo) bool {
	return typeInfo.Name == "error" && typeInfo.Package == ""
}

func ParseAnnotations(comments []*CommentInfo) (annotations Annotations, err error) {
	for _, comment := range comments {
		s := strings.TrimSpace(comment.Value)

		a, err := annotation.Parse(s)
		if err != nil {
			return nil, err
		}
		posInfo := parsePosition(comment.Position)
		annotations = append(annotations, &AnnotationInfo{
			Annotation: a,
			Position:   posInfo,
		})
	}

	return annotations, nil
}
