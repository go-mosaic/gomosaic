// Package pipeline предоставляет расширенные возможности пайплайна генерации:
// контекст генерации, промежуточные обработчики (middleware) и этапы обработки.
package pipeline

import (
	"context"
	"fmt"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

// Stage описывает этап обработки в пайплайне.
type Stage interface {
	Name() string
	Process(ctx context.Context, data *StageData) (*StageData, error)
}

// StageData хранит данные, передаваемые между этапами.
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

// NewStageData создаёт данные этапа для модуля и типов.
func NewStageData(module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) *StageData {
	return &StageData{
		Module:   module,
		Types:    types,
		Metadata: make(map[string]any),
	}
}

// Pipeline последовательно выполняет этапы.
type Pipeline struct {
	stages []Stage
}

// NewPipeline создаёт пайплайн из этапов.
func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// AddStage добавляет этап в конец пайплайна.
func (p *Pipeline) AddStage(stage Stage) {
	p.stages = append(p.stages, stage)
}

// Run выполняет этапы по порядку и возвращает итоговые данные.
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

// PluginAdapter адаптирует плагин-генератор к Stage.
type PluginAdapter struct {
	name   string
	plugin gomosaic.Generator
}

// NewPluginAdapter создаёт адаптер для плагина.
func NewPluginAdapter(p gomosaic.Generator) *PluginAdapter {
	return &PluginAdapter{
		name:   p.Name(),
		plugin: p,
	}
}

// Name реализует Stage.
func (a *PluginAdapter) Name() string {
	return a.name
}

// Process реализует Stage.
func (a *PluginAdapter) Process(ctx context.Context, data *StageData) (*StageData, error) {
	files, err := a.plugin.Generate(ctx, data.Module, data.Types)
	if err != nil {
		return nil, err
	}

	data.Files = files

	return data, nil
}
