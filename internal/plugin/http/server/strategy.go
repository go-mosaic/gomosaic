package server

type Strategy interface {
	UsePtrType() bool
	Type() string
	Pkg() string
	TransportConstruct() string
	TransportPkg() string
	PathParamWrap(paramName string) string
}
