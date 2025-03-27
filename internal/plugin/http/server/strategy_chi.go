package server

var _ Strategy = &StrategyChi{}

type StrategyChi struct{}

func (s *StrategyChi) UsePtrType() bool {
	return false
}

func (s *StrategyChi) Type() string {
	return "Router"
}

func (s *StrategyChi) Pkg() string {
	return "github.com/go-chi/chi/v5"
}

func (s *StrategyChi) TransportConstruct() string {
	return "NewChiTransport"
}

func (s *StrategyChi) TransportPkg() string {
	return "github.com/go-mosaic/runtime/transport/chi"
}

func (*StrategyChi) PathParamWrap(paramName string) string {
	return "{" + paramName + "}"
}
