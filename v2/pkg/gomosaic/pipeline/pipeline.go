// Package pipeline предоставляет расширенные возможности пайплайна генерации:
// контекст генерации, промежуточные обработчики (middleware) и этапы обработки.
package pipeline

import (
	"context"
	"fmt"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

type Stage interface {
	Name() string
	Process(ctx context.Context, data *StageData) (*StageData, error)
}

type StageData struct {
	Module *gomosaic.ModuleInfo
	Types  []*gomosaic.NameTypeInfo
	// Annotations результат этапа загрузки аннотаций.
	Annotations any
	// Files результат этапа генерации кода.
	Files map[string]gomosaic.File
	// Metadata произвольные данные между этапами.
	Metadata map[string]any
}

func NewStageData(module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) *StageData {
	return &StageData{
		Module:   module,
		Types:    types,
		Metadata: make(map[string]any),
	}
}

type Pipeline struct {
	stages []Stage
}

func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

func (p *Pipeline) AddStage(stage Stage) {
	p.stages = append(p.stages, stage)
}

func (p *Pipeline) Run(ctx context.Context, data *StageData) (*StageData, error) {
	var err error

	for _, stage := range p.stages {
		data, err = stage.Process(ctx, data)

		if err != nil {
			return nil, fmt.Errorf("этап %s: %w", stage.Name(), err)
		}
	}

	return data, nil
}

type PluginAdapter struct {
	name   string
	plugin gomosaic.Generator
}

func NewPluginAdapter(p gomosaic.Generator) *PluginAdapter {
	return &PluginAdapter{
		name:   p.Name(),
		plugin: p,
	}
}

func (a *PluginAdapter) Name() string {
	return a.name
}

func (a *PluginAdapter) Process(ctx context.Context, data *StageData) (*StageData, error) {
	files, err := a.plugin.Generate(ctx, data.Module, data.Types)
	if err != nil {
		return nil, err
	}

	data.Files = files

	return data, nil
}
