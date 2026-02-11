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

func (s *StrategyEcho) TransportFactoryType() string {
	return "TransportTypeEcho"
}

func (*StrategyEcho) PathParamWrap(paramName string) string {
	return "{" + paramName + "}"
}
