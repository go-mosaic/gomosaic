package server

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/v2/pkg/jenutils"
	"github.com/go-mosaic/gomosaic/v2/pkg/strcase"
)

func (g *generator) generateStreamMethod(serviceName string, svc *ServiceOpt, m *MethodOpt) jen.Code {
	group := jen.NewFile("")
	params := m.Func.Params

	methodName := m.Func.Name

	group.Func().Params(
		jen.Id("s").Op("*").Id(serviceName + "Server"),
	).Id(methodName).ParamsFunc(func(pg *jen.Group) {
		for _, p := range params {
			switch {
			case p.IsContext:
				pg.Id("ctx").Qual("context", "Context")
			case !isStreamType(p):
				if svc.ProtoPackage != "" && getBaseTypeName(p.Type) != "" {
					pg.Id(p.Name).Op("*").Qual(svc.ProtoPackage, getBaseTypeName(p.Type))
				} else {
					pg.Id(p.Name).Add(g.qualType(p.Type))
				}
			default:
				pg.Id(p.Name).Add(g.qualType(p.Type))
			}
		}
	}).Error().BlockFunc(func(body *jen.Group) {
		if svc.ProtoPackage != "" && m.MethodType == ServerStream {
			body.Add(g.generateServerStreamWrapper(serviceName, svc, m))
		} else if svc.ProtoPackage != "" && m.MethodType == ClientStream {
			body.Add(g.generateClientStreamWrapper(serviceName, svc, m))
		} else if svc.ProtoPackage != "" && m.MethodType == BidiStream {
			body.Add(g.generateBidiStreamWrapper(serviceName, svc, m))
		} else {
			callArgs := make([]jen.Code, 0)
			for _, p := range params {
				callArgs = append(callArgs, jen.Id(p.Name))
			}
			body.Return(jen.Id("s").Dot("svc").Dot(methodName).Call(callArgs...))
		}
	})

	return group
}

// generateServerStreamWrapper генерирует обёртку для серверного стриминга.
func (g *generator) generateServerStreamWrapper(serviceName string, svc *ServiceOpt, m *MethodOpt) jen.Code {
	methodName := m.Func.Name
	params := m.Func.Params

	var streamParam *gomosaic.VarInfo
	var otherParams []*gomosaic.VarInfo
	for _, p := range params {
		if isStreamType(p) {
			streamParam = p
		} else if !p.IsContext {
			otherParams = append(otherParams, p)
		}
	}

	group := jen.NewFile("")

	for _, p := range otherParams {
		typeName := getBaseTypeName(p.Type)
		if typeName != "" {
			group.Id("svc" + strcase.ToCamel(p.Name)).Op(":=").Id("convert" + typeName + "FromProto").Call(jen.Id(p.Name))
		}
	}

	streamVarName := streamParam.Name
	adapterName := strcase.ToLowerCamel(streamParam.Name) + "Adapter"

	group.Id(adapterName).Op(":=").Op("&").Id(serviceName + methodName + "Adapter").Values(
		jen.Dict{
			jen.Id(streamVarName): jen.Id(streamVarName),
		},
	)

	callArgs := []jen.Code{jen.Id("ctx")}
	for _, p := range otherParams {
		callArgs = append(callArgs, jen.Id("svc"+strcase.ToCamel(p.Name)))
	}
	callArgs = append(callArgs, jen.Id(adapterName))

	group.Return(jen.Id("s").Dot("svc").Dot(methodName).Call(callArgs...))

	return group
}

// generateClientStreamWrapper генерирует обёртку для клиентского стриминга.
func (g *generator) generateClientStreamWrapper(serviceName string, svc *ServiceOpt, m *MethodOpt) jen.Code {
	methodName := m.Func.Name
	params := m.Func.Params

	var streamParam *gomosaic.VarInfo
	var otherParams []*gomosaic.VarInfo
	for _, p := range params {
		if isStreamType(p) {
			streamParam = p
		} else if !p.IsContext {
			otherParams = append(otherParams, p)
		}
	}

	group := jen.NewFile("")

	for _, p := range otherParams {
		typeName := getBaseTypeName(p.Type)
		if typeName != "" {
			group.Id("svc" + strcase.ToCamel(p.Name)).Op(":=").Id("convert" + typeName + "FromProto").Call(jen.Id(p.Name))
		}
	}

	streamVarName := streamParam.Name
	adapterName := strcase.ToLowerCamel(streamParam.Name) + "Adapter"

	group.Id(adapterName).Op(":=").Op("&").Id(serviceName + methodName + "Adapter").Values(
		jen.Dict{
			jen.Id(streamVarName): jen.Id(streamVarName),
		},
	)

	callArgs := []jen.Code{jen.Id("ctx")}
	for _, p := range otherParams {
		callArgs = append(callArgs, jen.Id("svc"+strcase.ToCamel(p.Name)))
	}
	callArgs = append(callArgs, jen.Id(adapterName))

	group.Return(jen.Id("s").Dot("svc").Dot(methodName).Call(callArgs...))

	return group
}

// generateBidiStreamWrapper генерирует обёртку для двунаправленного стриминга.
func (g *generator) generateBidiStreamWrapper(serviceName string, svc *ServiceOpt, m *MethodOpt) jen.Code {
	methodName := m.Func.Name
	params := m.Func.Params

	var streamParam *gomosaic.VarInfo
	for _, p := range params {
		if isStreamType(p) {
			streamParam = p
			break
		}
	}

	streamVarName := streamParam.Name
	adapterName := strcase.ToLowerCamel(streamParam.Name) + "Adapter"

	group := jen.NewFile("")

	group.Id(adapterName).Op(":=").Op("&").Id(serviceName + methodName + "Adapter").Values(
		jen.Dict{
			jen.Id(streamVarName): jen.Id(streamVarName),
		},
	)

	group.Return(jen.Id("s").Dot("svc").Dot(methodName).Call(
		jen.Id("ctx"),
		jen.Id(adapterName),
	))

	return group
}

// generateStreamAdapters генерирует адаптеры стримов для конвертации proto ↔ DTO.
func (g *generator) generateStreamAdapters(serviceName string, svc *ServiceOpt) jen.Code {
	group := jen.NewFile("")

	for _, m := range svc.Methods {
		if m.MethodType == Unary {
			continue
		}

		params := m.Func.Params
		var streamParam *gomosaic.VarInfo
		for _, p := range params {
			if isStreamType(p) {
				streamParam = p
				break
			}
		}
		if streamParam == nil {
			continue
		}

		adapterName := serviceName + m.Func.Name + "Adapter"
		group.Type().Id(adapterName).Struct(
			jen.Id(streamParam.Name).Add(g.qualType(streamParam.Type)),
		)

		streamTypeName := streamParam.Type.Name
		var sendType, recvType *gomosaic.TypeInfo
		if streamNT, ok := g.allTypes[streamTypeName]; ok && streamNT.Type.Interface != nil {
			for _, ifaceMethod := range streamNT.Type.Interface.Methods {
				switch ifaceMethod.Name {
				case "Send":
					if len(ifaceMethod.Params) > 0 {
						sendType = ifaceMethod.Params[0].Type
					}
				case "Recv":
					if len(ifaceMethod.Results) > 0 {
						recvType = ifaceMethod.Results[0].Type
					}
				}
			}
		}

		if sendType != nil && (m.MethodType == ServerStream || m.MethodType == BidiStream) {
			sendBaseName := getBaseTypeName(sendType)
			group.Func().Params(jen.Id("a").Op("*").Id(adapterName)).Id("Send").
				Params(jen.Id("msg").Add(jenutils.TypeInfoQual(sendType, g.qual))).
				Error().
				Block(
					jen.Return(
						jen.Id("a").Dot(streamParam.Name).Dot("Send").Call(
							jen.Id("convert" + sendBaseName + "ToProto").Call(jen.Id("msg")),
						),
					),
				)
		}

		if recvType != nil && (m.MethodType == ClientStream || m.MethodType == BidiStream) {
			recvBaseName := getBaseTypeName(recvType)
			group.Func().Params(jen.Id("a").Op("*").Id(adapterName)).Id("Recv").
				Params().
				Params(jenutils.TypeInfoQual(recvType, g.qual), jen.Error()).
				Block(
					jen.List(jen.Id("msg"), jen.Err()).Op(":=").Id("a").Dot(streamParam.Name).Dot("Recv").Call(),
					jen.If(jen.Err().Op("!=").Nil()).Block(
						jen.Return(jen.Nil(), jen.Err()),
					),
					jen.Return(
						jen.Id("convert"+recvBaseName+"FromProto").Call(jen.Id("msg")),
						jen.Nil(),
					),
				)
		}
	}

	return group
}
