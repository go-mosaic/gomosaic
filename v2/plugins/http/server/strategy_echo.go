package server

// EchoStrategy — стратегия для Echo-роутера.
type EchoStrategy struct{}

func (s *EchoStrategy) Name() string                          { return "http-server-echo" }
func (s *EchoStrategy) UsePtrType() bool                      { return true }
func (s *EchoStrategy) RouterType() string                    { return "Echo" }
func (s *EchoStrategy) RouterPkg() string                     { return "github.com/labstack/echo/v4" }
func (s *EchoStrategy) TransportFactoryType() string          { return "TransportTypeEcho" }
func (s *EchoStrategy) PathParamWrap(paramName string) string { return ":" + paramName }
