package gomosaic

// Builder предоставляет функционал для сборки генератора кода.
//
// Пример использования:
//
//	builder := gomosaic.NewBuilder("myapp", "1.0.0")
//	builder.WithPlugin(httpPlugin)
//	builder.WithTransformer(myTransform)
//	gen := builder.Build()
type Builder struct {
	pluginReg    *PluginRegistry
	transformReg *TransformRegistry
}

type BuilderOption func(*Builder)

func WithPluginRegistry(r *PluginRegistry) BuilderOption {
	return func(b *Builder) { b.pluginReg = r }
}

func WithTransformRegistry(r *TransformRegistry) BuilderOption {
	return func(b *Builder) { b.transformReg = r }
}

// NewBuilder создает Builder с пустым реестром плагинов и
// стандартным набором трансформеров.
func NewBuilder(opts ...BuilderOption) *Builder {
	b := &Builder{
		pluginReg:    NewPluginRegistry(),
		transformReg: DefaultTransformRegistry(),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

func (b *Builder) WithPlugins(plugins ...Generator) *Builder {
	for _, p := range plugins {
		b.pluginReg.MustRegister(p)
	}

	return b
}

func (b *Builder) WithTransformer(factory func() Transformer) *Builder {
	b.transformReg.AddTransformer(factory)
	return b
}

func (b *Builder) WithParser(factory func() Parser) *Builder {
	b.transformReg.AddParser(factory)
	return b
}

func (b *Builder) WithFormatter(factory func() Formatter) *Builder {
	b.transformReg.AddFormatter(factory)
	return b
}

func (b *Builder) Build(fs FileSystem) *CodeGenerator {
	return NewCodeGenerator(b.pluginReg, b.transformReg, fs)
}

func (b *Builder) PluginRegistry() *PluginRegistry {
	return b.pluginReg
}

func (b *Builder) TransformRegistry() *TransformRegistry {
	return b.transformReg
}
