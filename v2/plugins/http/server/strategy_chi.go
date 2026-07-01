package server

// ChiStrategy — стратегия для Chi-роутера.
type ChiStrategy struct{}

func (s *ChiStrategy) Name() string                          { return "http-server-chi" }
func (s *ChiStrategy) UsePtrType() bool                      { return false }
func (s *ChiStrategy) RouterType() string                    { return "Router" }
func (s *ChiStrategy) RouterPkg() string                     { return "github.com/go-chi/chi/v5" }
func (s *ChiStrategy) TransportFactoryType() string          { return "TransportTypeChi" }
func (s *ChiStrategy) PathParamWrap(paramName string) string { return "{" + paramName + "}" }
