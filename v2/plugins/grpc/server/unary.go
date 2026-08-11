package server

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
)

func (g *generator) generateUnaryMethod(serviceName string, svc *ServiceOpt, m *MethodOpt) jen.Code {
	group := jen.NewFile("")

	params := m.Func.Params
	results := m.Func.Results

	if len(params) < 2 {
		return group
	}

	if len(results) < 1 {
		return group
	}

	reqParam := params[1]
	respParam := results[0]

	reqTypeName := getBaseTypeName(reqParam.Type)
	respTypeName := getBaseTypeName(respParam.Type)

	var reqType jen.Code
	var respType jen.Code

	if svc.ProtoPackage != "" {
		if reqTypeName != "" {
			reqType = jen.Op("*").Qual(svc.ProtoPackage, reqTypeName)
		} else {
			reqType = jenutils.TypeInfoQual(reqParam.Type, g.qual)
		}
		if respTypeName != "" {
			respType = jen.Op("*").Qual(svc.ProtoPackage, respTypeName)
		} else {
			respType = jenutils.TypeInfoQual(respParam.Type, g.qual)
		}
	} else {
		reqType = jenutils.TypeInfoQual(reqParam.Type, g.qual)
		respType = jenutils.TypeInfoQual(respParam.Type, g.qual)
	}

	methodName := m.Func.Name

	group.Func().Params(
		jen.Id("s").Op("*").Id(serviceName+"Server"),
	).Id(methodName).Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("req").Add(reqType),
	).Params(
		respType,
		jen.Error(),
	).BlockFunc(func(body *jen.Group) {
		if svc.ProtoPackage != "" {
			reqConv := g.generateParamConversion(reqParam.Type, jen.Id("req"), reqTypeName, svc, true)
			body.Add(reqConv)

			body.List(jen.Id("serviceResp"), jen.Err()).Op(":=").Id("s").Dot("svc").Dot(methodName).Call(
				jen.Id("ctx"),
				jen.Id("serviceReq"),
			)
			body.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Err()),
			)

			respConv := g.generateResultConversion(respParam.Type, jen.Id("serviceResp"), respTypeName, svc)
			body.Id("resp").Op(":=").Add(respConv)
			body.Return(jen.Id("resp"), jen.Nil())
		} else {
			body.List(jen.Id("resp"), jen.Err()).Op(":=").Id("s").Dot("svc").Dot(methodName).Call(
				jen.Id("ctx"),
				jen.Id("req"),
			)
			body.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Err()),
			)
			body.Return(jen.Id("resp"), jen.Nil())
		}
	})

	return group
}

// generateParamConversion генерирует код конвертации параметра из proto в DTO.
func (g *generator) generateParamConversion(paramType *gomosaic.TypeInfo, protoValue jen.Code, typeName string, svc *ServiceOpt, isFromProto bool) jen.Code {
	actualType := paramType
	for actualType.IsPtr && actualType.ElemType != nil {
		actualType = actualType.ElemType
	}

	if actualType.IsBasic {
		if isFromProto {
			return jen.Id("serviceReq").Op(":=").Add(protoValue)
		}
		return protoValue
	}

	if typeName != "" {
		if _, hasDTO := g.allTypes[typeName]; hasDTO {
			if isFromProto {
				return jen.Id("serviceReq").Op(":=").Id("convert" + typeName + "FromProto").Call(protoValue)
			}
			return jen.Id("convert" + typeName + "ToProto").Call(protoValue)
		}
	}

	if isFromProto {
		return jen.Id("serviceReq").Op(":=").Add(protoValue)
	}
	return protoValue
}

// generateResultConversion генерирует код конвертации результата из DTO в proto.
func (g *generator) generateResultConversion(resultType *gomosaic.TypeInfo, dtoValue jen.Code, typeName string, svc *ServiceOpt) jen.Code {
	actualType := resultType
	for actualType.IsPtr && actualType.ElemType != nil {
		actualType = actualType.ElemType
	}

	if actualType.IsBasic {
		return dtoValue
	}

	if typeName != "" {
		if _, hasDTO := g.allTypes[typeName]; hasDTO {
			return jen.Id("convert" + typeName + "ToProto").Call(dtoValue)
		}
	}

	return dtoValue
}
