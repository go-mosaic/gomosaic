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

func ContextWithOutputDir(ctx context.Context, outputDir string) context.Context {
	return context.WithValue(ctx, outputDirContextKey, outputDir)
}

func OutputDirFromContext(ctx context.Context) string {
	return ctx.Value(outputDirContextKey).(string)
}

// Generator интерфейс для плагинов
type Generator interface {
	// Generate Генерация файлов на основе информации о модуле и типах
	Generate(ctx context.Context, module *ModuleInfo, types []*NameTypeInfo) (map[string]File, error)
	// Name Имя плагина
	Name() string
}

// CodeGenerator основной генератор кода
type CodeGenerator struct {
	pluginManager *PluginManager
	fs            *FileSystem
}

// NewCodeGenerator создает новый экземпляр CodeGenerator
func NewCodeGenerator(pluginManager *PluginManager, fs *FileSystem) *CodeGenerator {
	return &CodeGenerator{
		pluginManager: pluginManager,
		fs:            fs,
	}
}

// Generate использует плагин для генерации кода и сохраняет файлы
func (cg *CodeGenerator) Generate(ctx context.Context, module *ModuleInfo, types []*NameTypeInfo, pluginName string) (outputFiles []string, err error) {
	plugin, err := cg.pluginManager.GetPlugin(pluginName)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить плагин: %w", err)
	}

	files, err := plugin.Generate(ctx, module, types)
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
