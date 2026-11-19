package gomosaic

import (
	"context"
	"fmt"
	"sort"
)

// ContextKey задаёт тип ключей контекста генератора.
type ContextKey string

const (
	outputDirContextKey         ContextKey = "output_dir"
	transformRegistryContextKey ContextKey = "transform_registry"
)

// TransformRegistryFromContext достаёт реестр трансформеров из контекста.
func TransformRegistryFromContext(ctx context.Context) *TransformRegistry {
	reg, _ := ctx.Value(transformRegistryContextKey).(*TransformRegistry)
	return reg
}

// ContextWithOutputDir кладёт директорию вывода в контекст.
func ContextWithOutputDir(ctx context.Context, outputDir string) context.Context {
	return context.WithValue(ctx, outputDirContextKey, outputDir)
}

// OutputDirFromContext возвращает директорию вывода из контекста.
func OutputDirFromContext(ctx context.Context) string {
	return ctx.Value(outputDirContextKey).(string)
}

// Generator описывает интерфейс плагина-генератора кода.
type Generator interface {
	Name() string
	Generate(ctx context.Context, module *ModuleInfo, types []*NameTypeInfo) (map[string]File, error)
}

// CodeGenerator запускает плагин и сохраняет результат в файловую систему.
type CodeGenerator struct {
	pluginRegistry    *PluginRegistry
	transformRegistry *TransformRegistry
	fs                FileSystem
}

// NewCodeGenerator создаёт CodeGenerator на готовых реестрах и файловой системе.
func NewCodeGenerator(pluginReg *PluginRegistry, transformReg *TransformRegistry, fs FileSystem) *CodeGenerator {
	return &CodeGenerator{
		pluginRegistry:    pluginReg,
		transformRegistry: transformReg,
		fs:                fs,
	}
}

// Generate запускает плагин pluginName и сохраняет сгенерированные файлы.
func (cg *CodeGenerator) Generate(ctx context.Context, module *ModuleInfo, types []*NameTypeInfo, pluginName string) (outputFiles []string, err error) {
	p, err := cg.pluginRegistry.Get(pluginName)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить плагин: %w", err)
	}

	ctx = context.WithValue(ctx, transformRegistryContextKey, cg.transformRegistry)

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
