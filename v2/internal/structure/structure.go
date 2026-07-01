// Package structure генерирует объявления типов для их копирования в клиентский пакет.
package structure

import (
	"errors"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

var errNoTag = errors.New("no json tag")

// Generator накапливает скопированные типы и запоминает их для переопределения квалификации.
type Generator struct {
	f          *jen.File
	modulePath string
	processed  map[string]bool
}

// NewGenerator создаёт генератор, который пишет объявления типов в f.
func NewGenerator(f *jen.File, modulePath string) *Generator {
	return &Generator{
		f:          f,
		modulePath: modulePath,
		processed:  make(map[string]bool),
	}
}

// Generate рекурсивно генерирует объявление типа t.
func (g *Generator) Generate(t *gomosaic.TypeInfo) {
	g.generateType(t)
}

// Processed возвращает множество ключей "pkgPath.TypeName", которые были скопированы.
func (g *Generator) Processed() map[string]bool {
	return g.processed
}

func (g *Generator) generateType(t *gomosaic.TypeInfo) {
	if t == nil {
		return
	}

	if t.IsPtr || t.IsMap || t.IsSlice {
		t = t.ElemType
	}

	if t == nil || t.IsPtr {
		if t == nil {
			return
		}

		t = t.ElemType
	}

	if t == nil || !t.IsNamed {
		return
	}

	if !strings.HasPrefix(t.Package, g.modulePath) {
		return
	}

	processedKey := t.Package + "." + t.Name
	if g.processed[processedKey] {
		return
	}

	if t.ElemType == nil {
		return
	}

	switch {
	case t.ElemType.Struct != nil:
		g.processed[processedKey] = true
		g.generateStruct(t.Name, t.ElemType)
	case t.ElemType.IsSlice:
		g.processed[processedKey] = true
		g.generateSliceAlias(t.Name, t.ElemType)
	case t.ElemType.IsBasic:
		g.processed[processedKey] = true
		g.generateBasicAlias(t.Name, t.ElemType)
	}
}

func (g *Generator) generateBasicAlias(name string, t *gomosaic.TypeInfo) {
	g.f.Type().Id(name).Id(t.Name)
}

func (g *Generator) generateSliceAlias(name string, t *gomosaic.TypeInfo) {
	elemType := g.getTypeExpr(t.ElemType)

	g.f.Type().Id(name).Index().Add(elemType)

	g.generateType(t.ElemType)
}

func (g *Generator) generateStruct(name string, t *gomosaic.TypeInfo) {
	g.f.Type().Id(name).StructFunc(func(s *jen.Group) {
		for _, field := range t.Struct.Fields {
			if field.Name == "" {
				continue
			}

			fieldType := g.getTypeExpr(field.Type)

			tag, err := jsonTag(field)
			if err != nil {
				s.Id(field.Name).Add(fieldType)

				continue
			}

			if tag == "-" {
				continue
			}

			s.Id(field.Name).Add(fieldType).Tag(map[string]string{"json": tag})
		}
	})

	for _, field := range t.Struct.Fields {
		g.generateType(field.Type)
	}
}

func jsonTag(field *gomosaic.VarInfo) (string, error) {
	if field.Tags == nil {
		return "", errNoTag
	}

	tag, err := field.Tags.Get("json")
	if err != nil {
		return "", err
	}

	return tag.Value(), nil
}

func (g *Generator) getTypeExpr(t *gomosaic.TypeInfo) jen.Code {
	if t == nil {
		return jen.Interface()
	}

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
		if !strings.HasPrefix(t.Package, g.modulePath) {
			s := jen.Qual(t.Package, t.Name)
			if t.IsInstantiated {
				s.IndexFunc(func(group *jen.Group) {
					for _, typeParam := range t.TypeParams {
						group.Add(g.getTypeExpr(typeParam))
					}
				})
			}

			return s
		}

		return jen.Id(t.Name)
	default:
		return jen.Interface()
	}
}
