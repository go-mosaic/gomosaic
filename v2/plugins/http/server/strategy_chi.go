package server

// ChiStrategy реализует Strategy для Chi-роутера.
type ChiStrategy struct{}

// Name реализует Strategy.
func (s *ChiStrategy) Name() string { return "http-server-chi" }

// UsePtrType реализует Strategy.
func (s *ChiStrategy) UsePtrType() bool { return false }

// RouterType реализует Strategy.
func (s *ChiStrategy) RouterType() string { return "Router" }

// RouterPkg реализует Strategy.
func (s *ChiStrategy) RouterPkg() string { return "github.com/go-chi/chi/v5" }

// TransportFactoryType реализует Strategy.
func (s *ChiStrategy) TransportFactoryType() string { return "TransportTypeChi" }

// PathParamWrap реализует Strategy.
func (s *ChiStrategy) PathParamWrap(paramName string) string { return "{" + paramName + "}" }
