package structure

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

var knownImports = map[string]bool{
	"time.Time":                   true,
	"github.com/google/uuid.UUID": true,
	"encoding/json.Number":        true,
	"encoding/json.RawMessage":    true,
}

type Generator struct {
	f         *jen.File
	processed map[string]bool
}

func NewGenerator(f *jen.File) *Generator {
	return &Generator{
		f:         f,
		processed: make(map[string]bool),
	}
}

func (g *Generator) Generate(t *gomosaic.TypeInfo) {
	g.generateType(t)
}

func (g *Generator) generateType(t *gomosaic.TypeInfo) {
	if t.IsPtr || t.IsMap || t.IsSlice {
		t = t.ElemType
	}

	if t.IsPtr {
		t = t.ElemType
	}

	if !t.IsNamed {
		return
	}

	processedKey := t.Package + "." + t.Name

	if g.processed[processedKey] || knownImports[processedKey] {
		return
	}

	g.processed[processedKey] = true

	switch {
	case t.ElemType.Struct != nil:
		g.generateStruct(t.Name, t.ElemType)
	case t.ElemType.IsSlice:
		g.generateSliceAlias(t.Name, t.ElemType)
	case t.ElemType.IsBasic:
		g.generateBasicAlias(t.Name, t.ElemType)
	default:
		return
	}
}

func (g *Generator) generateBasicAlias(name string, t *gomosaic.TypeInfo) {
	g.f.Type().Id(name).Id(t.Name)
}

func (g *Generator) generateSliceAlias(name string, t *gomosaic.TypeInfo) {
	elemType := g.getTypeExpr(t.ElemType)

	g.f.Type().Id(name).Index().Add(elemType)

	// Генерируем тип элемента если он именованный
	// (проверка на именованный тип есть внутри метода generateType)
	g.generateType(t.ElemType)
}

func (g *Generator) generateStruct(name string, t *gomosaic.TypeInfo) {
	g.f.Type().Id(name).StructFunc(func(s *jen.Group) {
		for _, field := range t.Struct.Fields {
			if field.Name == "" {
				continue
			}

			fieldType := g.getTypeExpr(field.Type)

			if tag, err := field.Tags.Get("json"); err == nil {
				if tag.Name == "-" {
					continue
				}

				s.Id(field.Name).Add(fieldType).Tag(map[string]string{"json": tag.Name})
			} else {
				s.Id(field.Name).Add(fieldType)
			}
		}
	})

	for _, field := range t.Struct.Fields {
		if knownImports[field.Type.Package+"."+field.Type.Name] {
			continue
		}

		g.generateType(field.Type)
	}
}

func (g *Generator) Processed() map[string]bool {
	return g.processed
}

func (g *Generator) getTypeExpr(t *gomosaic.TypeInfo) jen.Code {
	if t.IsBasic {
		return jen.Id(t.Name)
	}

	switch {
	case t.IsPtr:
		return jen.Op("*").Add(g.getTypeExpr(t.ElemType))
	case t.IsSlice:
		return jen.Index().Add(g.getTypeExpr(t.ElemType))
	case t.IsArray:
		return jen.Index(jen.Lit(t.ArrayLen)).Add(g.getTypeExpr(t.ElemType))
	case t.IsMap:
		return jen.Map(g.getTypeExpr(t.KeyType)).Add(g.getTypeExpr(t.ElemType))
	case t.IsNamed:
		if knownImports[t.Package+"."+t.Name] {
			return jen.Qual(t.Package, t.Name)
		}

		return jen.Id(t.Name)
	default:
		return jen.Interface()
	}
}
