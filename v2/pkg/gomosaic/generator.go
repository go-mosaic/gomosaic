package gomosaic

import (
	"context"
	"fmt"
	"sort"
)

type ContextKey string

const (
	outputDirContextKey ContextKey = "output_dir"
)

// ContextWithOutputDir добавляет директорию вывода в контекст.
func ContextWithOutputDir(ctx context.Context, outputDir string) context.Context {
	return context.WithValue(ctx, outputDirContextKey, outputDir)
}

// OutputDirFromContext извлекает директорию вывода из контекста.
func OutputDirFromContext(ctx context.Context) string {
	return ctx.Value(outputDirContextKey).(string)
}

// Generator интерфейс для плагинов генерации кода.
type Generator interface {
	// Name возвращает уникальное имя плагина.
	Name() string

	// Generate генерирует файлы на основе информации о модуле и типах.
	Generate(ctx context.Context, module *ModuleInfo, types []*NameTypeInfo) (map[string]File, error)
}

type CodeGenerator struct {
	pluginRegistry    *PluginRegistry
	transformRegistry *TransformRegistry
	fs                *FileSystem
}

func NewCodeGenerator(pluginReg *PluginRegistry, transformReg *TransformRegistry, fs *FileSystem) *CodeGenerator {
	return &CodeGenerator{
		pluginRegistry:    pluginReg,
		transformRegistry: transformReg,
		fs:                fs,
	}
}

func (cg *CodeGenerator) Generate(ctx context.Context, module *ModuleInfo, types []*NameTypeInfo, pluginName string) (outputFiles []string, err error) {
	p, err := cg.pluginRegistry.Get(pluginName)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить плагин: %w", err)
	}

	files, err := p.Generate(ctx, module, types)
	if err != nil {
		return nil, fmt.Errorf("не удалось сгенерировать код: %w", err)
	}

	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, filename := range keys {
		file := files[filename]

		outputFilename, err := cg.fs.SaveFile(filename, file)
		if err != nil {
			return nil, fmt.Errorf("не удалось сохранить файл: %w", err)
		}

		outputFiles = append(outputFiles, outputFilename)
	}

	return outputFiles, nil
}

// Builder предоставляет гибкое API для сборки генератора кода.
//
// Пример использования:
//
//	builder := gomosaic.NewBuilder("myapp", "1.0.0")
//	builder.WithPlugin(httpPlugin)
//	builder.WithTransformer(myTransform)
//	gen := builder.Build()
type Builder struct {
	version      string
	pluginReg    *PluginRegistry
	transformReg *TransformRegistry
	fs           *FileSystem
}

type BuilderOption func(*Builder)

// WithVersion устанавливает версию генератора.
func WithVersion(v string) BuilderOption {
	return func(b *Builder) { b.version = v }
}

// WithPluginRegistry устанавливает кастомный реестр плагинов.
func WithPluginRegistry(r *PluginRegistry) BuilderOption {
	return func(b *Builder) { b.pluginReg = r }
}

// WithTransformRegistry устанавливает кастомный реестр трансформеров.
func WithTransformRegistry(r *TransformRegistry) BuilderOption {
	return func(b *Builder) { b.transformReg = r }
}

// NewBuilder создает новый Builder для сборки генератора.
func NewBuilder(version string, outputDir string, opts ...BuilderOption) *Builder {
	b := &Builder{
		version:      version,
		pluginReg:    NewPluginRegistry(),
		transformReg: DefaultTransformRegistry(),
		fs:           NewFileSystem(version, outputDir),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// WithPlugins добавляет несколько плагинов.
func (b *Builder) WithPlugins(plugins ...Generator) *Builder {
	for _, p := range plugins {
		b.pluginReg.MustRegister(p)
	}

	return b
}

// WithTransformer добавляет пользовательский трансформер.
func (b *Builder) WithTransformer(factory func() Transformer) *Builder {
	b.transformReg.AddTransformer(factory)
	return b
}

// WithParser добавляет парсер типов.
func (b *Builder) WithParser(factory func() Parser) *Builder {
	b.transformReg.AddParser(factory)
	return b
}

// WithFormatter добавляет форматер типов.
func (b *Builder) WithFormatter(factory func() Formatter) *Builder {
	b.transformReg.AddFormatter(factory)
	return b
}

// Build создает готовый к использованию CodeGenerator.
func (b *Builder) Build() *CodeGenerator {
	return NewCodeGenerator(b.pluginReg, b.transformReg, b.fs)
}

// PluginRegistry возвращает реестр плагинов.
func (b *Builder) PluginRegistry() *PluginRegistry {
	return b.pluginReg
}

// TransformRegistry возвращает реестр трансформеров.
func (b *Builder) TransformRegistry() *TransformRegistry {
	return b.transformReg
}
