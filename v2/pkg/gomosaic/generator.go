package gomosaic

import (
	"context"
	"fmt"
	"sort"
)

type ContextKey string

const (
	outputDirContextKey         ContextKey = "output_dir"
	transformRegistryContextKey ContextKey = "transform_registry"
)

func TransformRegistryFromContext(ctx context.Context) *TransformRegistry {
	reg, _ := ctx.Value(transformRegistryContextKey).(*TransformRegistry)
	return reg
}

func ContextWithOutputDir(ctx context.Context, outputDir string) context.Context {
	return context.WithValue(ctx, outputDirContextKey, outputDir)
}

func OutputDirFromContext(ctx context.Context) string {
	return ctx.Value(outputDirContextKey).(string)
}

type Generator interface {
	Name() string
	Generate(ctx context.Context, module *ModuleInfo, types []*NameTypeInfo) (map[string]File, error)
}

type CodeGenerator struct {
	pluginRegistry    *PluginRegistry
	transformRegistry *TransformRegistry
	fs                FileSystem
}

func NewCodeGenerator(pluginReg *PluginRegistry, transformReg *TransformRegistry, fs FileSystem) *CodeGenerator {
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
