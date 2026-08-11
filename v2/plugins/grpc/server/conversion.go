package server

import (
	"sort"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
)

func (g *generator) generateConversionFunctions(svc *ServiceOpt) jen.Code {
	group := jen.NewFile("")

	// Собираем уникальные типы из всех методов вместе с их TypeInfo
	typeMap := make(map[string]*gomosaic.TypeInfo)
	for _, m := range svc.Methods {
		for _, p := range m.Func.Params {
			if !p.IsContext && !isStreamType(p) {
				g.collectTypes(p.Type, typeMap, svc)
			}
		}

		for _, r := range m.Func.Results {
			if !r.IsError {
				g.collectTypes(r.Type, typeMap, svc)
			}
		}
	}

	// Генерируем функции конвертации для каждого уникального типа
	typeNames := make([]string, 0, len(typeMap))
	for tn := range typeMap {
		typeNames = append(typeNames, tn)
	}
	sort.Strings(typeNames)

	for _, typeName := range typeNames {
		typeInfo := typeMap[typeName]
		dtoType, hasDTO := g.allTypes[typeName]

		if hasDTO && dtoType.Type.Struct != nil {
			group.Add(g.generateStructConversionFunctions(svc, typeName, dtoType))
		} else if typeInfo.IsBasic {
		} else {
			group.Add(g.generateStubConversionFunctions(svc, typeName))
		}
	}

	return group
}

// collectTypes собирает уникальные именованные типы, требующие функций конвертации.
func (g *generator) collectTypes(t *gomosaic.TypeInfo, typeMap map[string]*gomosaic.TypeInfo, svc *ServiceOpt) {
	if t == nil {
		return
	}

	if t.IsPtr && t.ElemType != nil {
		g.collectTypes(t.ElemType, typeMap, svc)
		return
	}

	if t.IsBasic || t.Name == "" || t.Name == "struct" || t.Name == "interface" {
		return
	}

	if t.Package == "" {
		if t.ElemType != nil {
			g.collectTypes(t.ElemType, typeMap, svc)
		}
		if t.KeyType != nil {
			g.collectTypes(t.KeyType, typeMap, svc)
		}
		return
	}

	baseName := getBaseTypeName(t)
	if baseName == "" || baseName == "struct" || baseName == "interface" {
		return
	}

	if _, exists := typeMap[baseName]; !exists {
		typeMap[baseName] = t
	}

	if t.ElemType != nil {
		g.collectTypes(t.ElemType, typeMap, svc)
	}
	if t.KeyType != nil {
		g.collectTypes(t.KeyType, typeMap, svc)
	}
}

// generateStructConversionFunctions генерирует функции конвертации для структуры.
func (g *generator) generateStructConversionFunctions(svc *ServiceOpt, typeName string, dtoType *gomosaic.NameTypeInfo) jen.Code {
	group := jen.NewFile("")

	structInfo := dtoType.Type.Struct
	if structInfo == nil || len(structInfo.Fields) == 0 {
		return group
	}

	servicePkg := dtoType.Package.Path
	protoPkg := svc.ProtoPackage

	group.Func().Id("convert" + typeName + "FromProto").
		Params(jen.Id("proto").Op("*").Qual(protoPkg, typeName)).
		Params(jen.Op("*").Do(g.qual(servicePkg, typeName))).
		BlockFunc(func(body *jen.Group) {
			body.If(jen.Id("proto").Op("==").Nil()).Block(
				jen.Return(jen.Nil()),
			)
			body.Return(
				jen.Op("&").Do(g.qual(servicePkg, typeName)).Values(
					jen.DictFunc(func(dict jen.Dict) {
						for _, field := range structInfo.Fields {
							if field.IsContext || field.IsError {
								continue
							}
							protoField := jen.Id("proto").Dot(field.Name)
							dict[jen.Id(field.Name)] = g.generateFieldFromProto(field.Type, protoField, svc)
						}
					}),
				),
			)
		})

	group.Func().Id("convert" + typeName + "ToProto").
		Params(jen.Id("dto").Op("*").Do(g.qual(servicePkg, typeName))).
		Params(jen.Op("*").Qual(protoPkg, typeName)).
		BlockFunc(func(body *jen.Group) {
			body.If(jen.Id("dto").Op("==").Nil()).Block(
				jen.Return(jen.Nil()),
			)
			body.Return(
				jen.Op("&").Qual(protoPkg, typeName).Values(
					jen.DictFunc(func(dict jen.Dict) {
						for _, field := range structInfo.Fields {
							if field.IsContext || field.IsError {
								continue
							}
							dtoField := jen.Id("dto").Dot(field.Name)
							dict[jen.Id(field.Name)] = g.generateFieldToProto(field.Type, dtoField, svc)
						}
					}),
				),
			)
		})

	return group
}

// generateFieldFromProto генерирует код конвертации поля из proto в DTO.
func (g *generator) generateFieldFromProto(fieldType *gomosaic.TypeInfo, protoField jen.Code, svc *ServiceOpt) jen.Code {
	actualType := fieldType
	for actualType.IsPtr && actualType.ElemType != nil {
		actualType = actualType.ElemType
	}

	switch {
	case actualType.IsSlice && actualType.ElemType != nil:
		elemType := actualType.ElemType
		if elemType.IsBasic {
			return jen.Append(jen.Index().Add(jenutils.TypeInfoQual(elemType, g.qual)).Values(), jen.Id("proto").Op("..."))
		}

		baseName := getBaseTypeName(elemType)

		return g.generateSliceFromProto(baseName, elemType, protoField, svc)

	case actualType.IsMap && actualType.ElemType != nil:
		elemType := actualType.ElemType
		if elemType.IsBasic {
			return protoField
		}

		baseName := getBaseTypeName(elemType)
		return g.generateMapFromProto(baseName, elemType, protoField, svc)

	case actualType.IsBasic:
		return protoField

	case actualType.Struct != nil || actualType.Package != "":
		baseName := getBaseTypeName(actualType)
		if _, hasDTO := g.allTypes[baseName]; hasDTO {
			return jen.Id("convert" + baseName + "FromProto").Call(protoField)
		}

		if g.transformRegistry != nil {
			tr := g.transformRegistry.For(actualType).SetValueID(protoField).SetQualFunc(g.qual)
			formatted := tr.Format()
			if formatted != nil {
				return formatted
			}
		}
		return protoField

	default:
		return protoField
	}
}

// generateFieldToProto генерирует код конвертации поля из DTO в proto.
func (g *generator) generateFieldToProto(fieldType *gomosaic.TypeInfo, dtoField jen.Code, svc *ServiceOpt) jen.Code {
	actualType := fieldType
	for actualType.IsPtr && actualType.ElemType != nil {
		actualType = actualType.ElemType
	}

	switch {
	case actualType.IsSlice && actualType.ElemType != nil:
		elemType := actualType.ElemType
		if elemType.IsBasic {
			return jen.Append(jen.Index().Add(jenutils.TypeInfoQual(elemType, g.qual)).Values(), jen.Id("dto").Op("..."))
		}
		baseName := getBaseTypeName(elemType)
		return g.generateSliceToProto(baseName, elemType, dtoField, svc)

	case actualType.IsMap && actualType.ElemType != nil:
		elemType := actualType.ElemType
		if elemType.IsBasic {
			return dtoField
		}
		baseName := getBaseTypeName(elemType)
		return g.generateMapToProto(baseName, elemType, dtoField, svc)

	case actualType.IsBasic:
		return dtoField

	case actualType.Struct != nil || actualType.Package != "":
		baseName := getBaseTypeName(actualType)
		if _, hasDTO := g.allTypes[baseName]; hasDTO {
			return jen.Id("convert" + baseName + "ToProto").Call(dtoField)
		}

		if g.transformRegistry != nil {
			tr := g.transformRegistry.For(actualType).SetValueID(dtoField).SetQualFunc(g.qual)
			formatted := tr.Format()
			if formatted != nil {
				return formatted
			}
		}

		return dtoField

	default:
		return dtoField
	}
}

// generateSliceFromProto генерирует inline-конвертацию слайса из proto в DTO.
func (g *generator) generateSliceFromProto(elemBaseName string, elemType *gomosaic.TypeInfo, protoField jen.Code, svc *ServiceOpt) jen.Code {
	dtoQual := g.getDTOQual(elemBaseName)
	return jen.Func().Params().Params(jen.Index().Add(dtoQual)).BlockFunc(func(body *jen.Group) {
		body.Id("result").Op(":=").Make(jen.Index().Add(dtoQual), jen.Len(protoField))
		body.For(jen.List(jen.Id("i"), jen.Id("item")).Op(":=").Range().Add(protoField)).Block(
			jen.Id("result").Index(jen.Id("i")).Op("=").Id("convert" + elemBaseName + "FromProto").Call(jen.Id("item")),
		)
		body.Return(jen.Id("result"))
	}).Call()
}

// generateSliceToProto генерирует inline-конвертацию слайса из DTO в proto.
func (g *generator) generateSliceToProto(elemBaseName string, elemType *gomosaic.TypeInfo, dtoField jen.Code, svc *ServiceOpt) jen.Code {
	return jen.Func().Params().Params(jen.Index().Qual(svc.ProtoPackage, elemBaseName)).BlockFunc(func(body *jen.Group) {
		body.Id("result").Op(":=").Make(jen.Index().Qual(svc.ProtoPackage, elemBaseName), jen.Len(dtoField))
		body.For(jen.List(jen.Id("i"), jen.Id("item")).Op(":=").Range().Add(dtoField)).Block(
			jen.Id("result").Index(jen.Id("i")).Op("=").Id("convert" + elemBaseName + "ToProto").Call(jen.Id("item")),
		)
		body.Return(jen.Id("result"))
	}).Call()
}

// generateMapFromProto генерирует inline-конвертацию map из proto в DTO.
func (g *generator) generateMapFromProto(elemBaseName string, elemType *gomosaic.TypeInfo, protoField jen.Code, svc *ServiceOpt) jen.Code {
	dtoQual := g.getDTOQual(elemBaseName)
	return jen.Func().Params().Params(jen.Map(jen.String()).Add(dtoQual)).BlockFunc(func(body *jen.Group) {
		body.Id("result").Op(":=").Make(jen.Map(jen.String()).Add(dtoQual), jen.Len(protoField))
		body.For(jen.List(jen.Id("k"), jen.Id("v")).Op(":=").Range().Add(protoField)).Block(
			jen.Id("result").Index(jen.Id("k")).Op("=").Id("convert" + elemBaseName + "FromProto").Call(jen.Id("v")),
		)
		body.Return(jen.Id("result"))
	}).Call()
}

// generateMapToProto генерирует inline-конвертацию map из DTO в proto.
func (g *generator) generateMapToProto(elemBaseName string, elemType *gomosaic.TypeInfo, dtoField jen.Code, svc *ServiceOpt) jen.Code {
	return jen.Func().Params().Params(jen.Map(jen.String()).Qual(svc.ProtoPackage, elemBaseName)).BlockFunc(func(body *jen.Group) {
		body.Id("result").Op(":=").Make(jen.Map(jen.String()).Qual(svc.ProtoPackage, elemBaseName), jen.Len(dtoField))
		body.For(jen.List(jen.Id("k"), jen.Id("v")).Op(":=").Range().Add(dtoField)).Block(
			jen.Id("result").Index(jen.Id("k")).Op("=").Id("convert" + elemBaseName + "ToProto").Call(jen.Id("v")),
		)
		body.Return(jen.Id("result"))
	}).Call()
}

// getDTOQual возвращает квалифицированную ссылку на DTO тип.
func (g *generator) getDTOQual(typeName string) jen.Code {
	if nt, ok := g.allTypes[typeName]; ok && nt.Package != nil {
		return jen.Do(g.qual(nt.Package.Path, typeName))
	}
	return jen.Id(typeName)
}

// generateStubConversionFunctions генерирует заглушки для типов, структура которых неизвестна.
func (g *generator) generateStubConversionFunctions(svc *ServiceOpt, typeName string) jen.Code {
	group := jen.NewFile("")

	group.Func().Id("convert" + typeName + "FromProto").
		Params(jen.Id("proto").Op("*").Qual(svc.ProtoPackage, typeName)).
		Params(jen.Op("*").Do(g.qual(svc.NameTypeInfo.Package.Path, typeName))).
		Block(
			jen.Return(jen.Op("&").Do(g.qual(svc.NameTypeInfo.Package.Path, typeName)).Values()),
		)

	group.Func().Id("convert" + typeName + "ToProto").
		Params(jen.Id("dto").Op("*").Do(g.qual(svc.NameTypeInfo.Package.Path, typeName))).
		Params(jen.Op("*").Qual(svc.ProtoPackage, typeName)).
		Block(
			jen.Return(jen.Op("&").Qual(svc.ProtoPackage, typeName).Values()),
		)

	return group
}
