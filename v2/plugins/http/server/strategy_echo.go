package server

// EchoStrategy реализует Strategy для Echo-роутера.
type EchoStrategy struct{}

// Name реализует Strategy.
func (s *EchoStrategy) Name() string { return "http-server-echo" }

// UsePtrType реализует Strategy.
func (s *EchoStrategy) UsePtrType() bool { return true }

// RouterType реализует Strategy.
func (s *EchoStrategy) RouterType() string { return "Echo" }

// RouterPkg реализует Strategy.
func (s *EchoStrategy) RouterPkg() string { return "github.com/labstack/echo/v4" }

// TransportFactoryType реализует Strategy.
func (s *EchoStrategy) TransportFactoryType() string { return "TransportTypeEcho" }

// PathParamWrap реализует Strategy.
func (s *EchoStrategy) PathParamWrap(paramName string) string { return ":" + paramName }
