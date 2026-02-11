package server

type Strategy interface {
	UsePtrType() bool
	Type() string
	Pkg() string
	TransportFactoryType() string
	PathParamWrap(paramName string) string
}
