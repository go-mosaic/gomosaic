package flatten

import (
	"github.com/dave/jennifer/jen"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type Paths []PathName

func (p Paths) String() string {
	var path string
	for i, v := range p {
		if i > 0 {
			path += "."
		}
		path += v.Name
	}
	return path
}

type FlattenPath struct {
	Var      *gomosaic.VarInfo
	Path     *jen.Statement
	Paths    Paths
	Children []FlattenPath
	IsArray  bool
}

type PathName struct {
	Name     string
	JSONName string
}

func (n PathName) Value() string {
	return n.Name
}

func (n PathName) JSON() string {
	if n.JSONName == "" {
		return n.Name
	}
	return n.JSONName
}

type FlattenProcessor struct {
	allPaths    []FlattenPath
	currentPath []PathName
}

func (p *FlattenProcessor) varsByType(typeInfo *gomosaic.TypeInfo) (vars []*gomosaic.VarInfo) {
	if typeInfo.IsNamed {
		typeInfo = typeInfo.ElemType
	}
	if typeInfo.Struct != nil {
		vars = append(vars, typeInfo.Struct.Fields...)
	}
	return vars
}

func (p *FlattenProcessor) flattenVar(v *gomosaic.VarInfo) {
	var jsonName string
	if tag, err := v.Tags.Get("json"); err == nil {
		jsonName = tag.Name
	}
	p.currentPath = append(p.currentPath, PathName{Name: v.Name, JSONName: jsonName})

	vars := p.varsByType(v.Type)

	hasChildren := len(vars) == 0

	if v.Type.IsPtr && !hasChildren {
		currentPath := make([]PathName, len(p.currentPath))
		copy(currentPath, p.currentPath)

		path := jen.Id(currentPath[0].Value()).Do(func(s *jen.Statement) {
			for i := 1; i < len(currentPath); i++ {
				s.Dot(currentPath[i].Value())
			}
		})

		p.allPaths = append(p.allPaths, FlattenPath{
			Var:   v,
			Paths: currentPath,
			Path:  path,
		})
	}

	if hasChildren {
		var (
			children []FlattenPath
			isArray  bool
		)

		if v.Type.IsSlice {
			isArray = true

			for _, v := range p.varsByType(v.Type.ElemType) {
				children = append(children, new(FlattenProcessor).Flatten(v)...)
			}
		}

		currentPath := make([]PathName, len(p.currentPath))
		copy(currentPath, p.currentPath)

		path := jen.Id(currentPath[0].Value()).Do(func(s *jen.Statement) {
			for i := 1; i < len(currentPath); i++ {
				s.Dot(currentPath[i].Value())
			}
		})

		p.allPaths = append(p.allPaths, FlattenPath{
			Var:      v,
			Paths:    currentPath,
			Children: children,
			Path:     path,
			IsArray:  isArray,
		})
	} else {
		for _, v := range vars {
			p.flattenVar(v)
		}
	}

	p.currentPath = p.currentPath[:len(p.currentPath)-1]
}

func (p *FlattenProcessor) Flatten(v *gomosaic.VarInfo) []FlattenPath {
	p.flattenVar(v)
	return p.allPaths
}

func Flatten(v *gomosaic.VarInfo) []FlattenPath {
	return (&FlattenProcessor{}).Flatten(v)
}
