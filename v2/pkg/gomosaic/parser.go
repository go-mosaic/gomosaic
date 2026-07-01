package gomosaic

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"github.com/fatih/structtag"
	"golang.org/x/tools/go/packages"

	"github.com/go-mosaic/gomosaic/v2/pkg/annotation"
)

func ParsePackage(dir string, paths []string) (nameTypesInfo []*NameTypeInfo, err error) {
	patterns := make([]string, len(paths))
	for i := range paths {
		patterns[i] = "pattern=" + paths[i]
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}

		returnValues := parseReturnValues(pkg, pkg.Syntax)
		scope := pkg.Types.Scope()
		visited := make(map[*types.Named]bool)

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

			typeInfo, err := typeToTypeInfoVisited(pkg, obj.Type().Underlying(), visited)
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

			for method := range named.Methods() {
				if !method.Exported() {
					continue
				}

				methodInfo, err := funcToMethodInfoVisited(pkg, method, visited)
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

// varToVarInfoVisited преобразует types.Var в VarInfo с отслеживанием посещённых типов.
func varToVarInfoVisited(pkg *packages.Package, v *types.Var, visited map[*types.Named]bool) (*VarInfo, error) {
	title, doc, annotations, err := findDocAndAnnotations(pkg, v.Name(), v.Pos())
	if err != nil {
		return nil, err
	}

	typeInfo, err := typeToTypeInfoVisited(pkg, v.Type(), visited)
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

// funcToMethodInfoVisited преобразует types.Func в MethodInfo с отслеживанием посещённых типов.
func funcToMethodInfoVisited(
	pkg *packages.Package,
	method *types.Func,
	visited map[*types.Named]bool,
) (*MethodInfo, error) {
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

		if recv := sig.Recv(); recv != nil {
			if named, ok := recv.Type().(*types.Named); ok {
				name := named.Obj().Name()
				if named.Obj().Pkg() != nil {
					name = named.Obj().Pkg().Name() + "." + name
				}

				methodInfo.ShortName = "(" + name + ")." + method.Name()
			}
		}

		paramVarsInfo, err := tuplesToVarsInfoVisited(pkg, sig.Params(), visited)
		if err != nil {
			return nil, err
		}

		methodInfo.Params = paramVarsInfo

		resultVarsInfo, err := tuplesToVarsInfoVisited(pkg, sig.Results(), visited)
		if err != nil {
			return nil, err
		}

		methodInfo.Results = resultVarsInfo
	}

	return methodInfo, nil
}

// tuplesToVarsInfoVisited преобразует types.Tuple в []VarInfo с отслеживанием посещённых типов.
func tuplesToVarsInfoVisited(pkg *packages.Package, tuple *types.Tuple, visited map[*types.Named]bool) (varsInfo []*VarInfo, err error) {
	if tuple == nil {
		return nil, nil
	}

	for v := range tuple.Variables() {
		varInfo, err := varToVarInfoVisited(pkg, v, visited)
		if err != nil {
			return nil, err
		}

		varsInfo = append(varsInfo, varInfo)
	}

	return varsInfo, nil
}

func packageToPackageInfo(pkg *types.Package) *PackageInfo {
	if pkg == nil {
		return &PackageInfo{}
	}

	return &PackageInfo{
		Name: pkg.Name(),
		Path: pkg.Path(),
	}
}

// typeToTypeInfoVisited преобразует types.Type в TypeInfo с отслеживанием посещённых *types.Named типов
// для предотвращения бесконечной рекурсии при циклических ссылках.
func typeToTypeInfoVisited(pkg *packages.Package, t types.Type, visited map[*types.Named]bool) (*TypeInfo, error) {
	typeInfo := &TypeInfo{}

	switch t := t.(type) {
	default:
		return typeInfo, nil
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
		default:
			typeInfo.BitSize = 64
		}
	case *types.Alias:
		typeInfo.Name = t.Obj().Name()
		typeInfo.IsAlias = true
	case *types.Chan:
		typeInfo.Name = "chan"
		typeInfo.IsChan = true

		if t.Dir() == types.SendOnly {
			typeInfo.Name += "<-"
		} else {
			typeInfo.Name = "<-" + typeInfo.Name
		}

		elemType, err := typeToTypeInfoVisited(pkg, t.Elem(), visited)
		if err != nil {
			return nil, err
		}

		typeInfo.ElemType = elemType
	case *types.Pointer:
		typeInfo.IsPtr = true

		elemType, err := typeToTypeInfoVisited(pkg, t.Elem(), visited)
		if err != nil {
			return nil, err
		}

		typeInfo.ElemType = elemType
	case *types.Slice:
		typeInfo.IsSlice = true

		elemType, err := typeToTypeInfoVisited(pkg, t.Elem(), visited)
		if err != nil {
			return nil, err
		}

		typeInfo.ElemType = elemType
	case *types.Array:
		typeInfo.IsArray = true
		typeInfo.ArrayLen = int(t.Len())

		elemType, err := typeToTypeInfoVisited(pkg, t.Elem(), visited)
		if err != nil {
			return nil, err
		}

		typeInfo.ElemType = elemType
	case *types.Map:
		typeInfo.IsMap = true

		keyType, err := typeToTypeInfoVisited(pkg, t.Key(), visited)
		if err != nil {
			return nil, err
		}

		typeInfo.KeyType = keyType

		elemType, err := typeToTypeInfoVisited(pkg, t.Elem(), visited)
		if err != nil {
			return nil, err
		}

		typeInfo.ElemType = elemType
	case *types.Named:
		typeInfo.IsNamed = true
		typeInfo.Name = t.Obj().Name()

		if pkg := t.Obj().Pkg(); pkg != nil {
			typeInfo.Package = pkg.Path()
		}

		if t.TypeArgs() != nil {
			typeInfo.IsInstantiated = true
			typeArgs := make([]*TypeInfo, 0, t.TypeArgs().Len())

			for typeArg := range t.TypeArgs().Types() {
				argInfo, err := typeToTypeInfoVisited(pkg, typeArg, visited)
				if err != nil {
					return nil, err
				}

				typeArgs = append(typeArgs, argInfo)
			}

			typeInfo.TypeParams = typeArgs
		}

		if t.Obj().Type() != nil {
			if visited != nil && visited[t] {
				// Уже посещали этот тип — пропускаем для предотвращения цикла
				break
			}
			if visited != nil {
				visited[t] = true
			}

			named, err := typeToTypeInfoVisited(pkg, t.Obj().Type().Underlying(), visited)
			if err != nil {
				return nil, err
			}

			typeInfo.ElemType = named
		}
	case *types.Struct:
		typeInfo.Name = "struct"
		structInfo := &StructInfo{
			Fields: make([]*VarInfo, 0, 64),
		}

		for i := range t.NumFields() {
			field := t.Field(i)
			if !field.Exported() {
				continue
			}

			varInfo, err := varToVarInfoVisited(pkg, field, visited)
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

		for method := range t.Methods() {
			if !method.Exported() {
				continue
			}

			// Для каждого метода создаём отдельную карту посещённых типов,
			// чтобы типы из одного метода не влияли на резолвинг в другом.
			methodVisited := make(map[*types.Named]bool)

			methodInfo, err := funcToMethodInfoVisited(pkg, method, methodVisited)
			if err != nil {
				return nil, err
			}

			interfaceInfo.Methods = append(interfaceInfo.Methods, methodInfo)
		}

		typeInfo.Interface = interfaceInfo
	case *types.Signature:
		typeInfo.Name = "func"
		var typeParams []*TypeInfo

		if t.TypeParams() != nil {
			typeParams = make([]*TypeInfo, 0, t.TypeParams().Len())

			for typeParam := range t.TypeParams().TypeParams() {
				paramInfo, err := typeToTypeInfoVisited(pkg, typeParam, visited)
				if err != nil {
					return nil, err
				}

				typeParams = append(typeParams, paramInfo)
			}
		}

		paramVarsInfo, err := tuplesToVarsInfoVisited(pkg, t.Params(), visited)
		if err != nil {
			return nil, err
		}

		resultVarsInfo, err := tuplesToVarsInfoVisited(pkg, t.Results(), visited)
		if err != nil {
			return nil, err
		}

		typeInfo.Signature = &SignatureInfo{
			Params:     paramVarsInfo,
			Results:    resultVarsInfo,
			TypeParams: typeParams,
		}
	case *types.TypeParam:
		typeInfo.IsTypeParam = true
		typeInfo.Name = t.Obj().Name()

		if constraint := t.Constraint(); constraint != nil {
			constraintInfo, err := typeToTypeInfoVisited(pkg, constraint, visited)
			if err != nil {
				return nil, err
			}

			typeInfo.ElemType = constraintInfo
		}
	case *types.Union:
		typeInfo.IsUnion = true
		typeInfo.Name = "union"

		terms := make([]*TypeInfo, 0, t.Len())
		for term := range t.Terms() {
			termInfo, err := typeToTypeInfoVisited(pkg, term.Type(), visited)
			if err != nil {
				return nil, err
			}

			terms = append(terms, termInfo)
		}

		typeInfo.UnionTerms = terms
	}

	return typeInfo, nil
}

func parseReturnValues(pkg *packages.Package, files []*ast.File) map[string][]*TypeAndValueInfo {
	returnValues := make(map[string][]*TypeAndValueInfo, 128)

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
			if !ok || !fn.Exported() {
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

func findComments(pkg *packages.Package, name string, pos token.Pos) (commentsInfo []*CommentInfo) {
	position := pkg.Fset.Position(pos)

	for _, file := range pkg.Syntax {
		for _, commentGroup := range file.Comments {
			cg := pkg.Fset.Position(commentGroup.End())
			if cg.Line == position.Line-1 && cg.Filename == position.Filename {
				for _, comment := range commentGroup.List {
					text := strings.TrimLeft(strings.TrimLeft(comment.Text, "/"), " ")
					isAnnotation := strings.HasPrefix(text, "@")

					isTitle := strings.HasPrefix(text, name)
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
	case constant.String:
		return StringValueKind
	case constant.Bool:
		return BoolValueKind
	case constant.Float:
		return FloatValueKind
	case constant.Complex:
		return ComplexValueKind
	case constant.Int:
		return IntValueKind
	default:
		return UnknownValueKind
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
		if !comment.IsAnnotation {
			continue
		}

		s := strings.TrimSpace(comment.Value)

		a, err := annotation.Parse(s)
		if err != nil {
			continue
		}

		posInfo := parsePosition(comment.Position)
		annotations = append(annotations, &AnnotationInfo{
			Annotation: a,
			Position:   posInfo,
		})
	}

	return annotations, nil
}
