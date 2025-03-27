package server

var _ Strategy = &StrategyEcho{}

type StrategyEcho struct{}

func (s *StrategyEcho) UsePtrType() bool {
	return true
}

func (s *StrategyEcho) Type() string {
	return "Echo"
}

func (s *StrategyEcho) Pkg() string {
	return "github.com/labstack/echo/v4"
}

func (s *StrategyEcho) TransportConstruct() string {
	return "NewEchoTransport"
}

func (s *StrategyEcho) TransportPkg() string {
	return "github.com/go-mosaic/runtime/transport/echo"
}

func (*StrategyEcho) PathParamWrap(paramName string) string {
	return "{" + paramName + "}"
}
