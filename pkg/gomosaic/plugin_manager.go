package gomosaic

import (
	"fmt"
	"plugin"
)

var DefaultPluginManager = NewPluginManager()

// PluginManager управляет загрузкой и использованием плагинов
type PluginManager struct {
	plugins map[string]Generator
}

// NewPluginManager создает новый менеджер плагинов
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Generator),
	}
}

// LoadPlugin загружает плагин из файла
func (pm *PluginManager) LoadPlugin(path string) error {
	plug, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("не удалось загрузить плагин: %w", err)
	}

	sym, err := plug.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("символ плагина не найден: %w", err)
	}

	generator, ok := sym.(Generator)
	if !ok {
		return fmt.Errorf("недопустимый тип плагина")
	}

	pm.plugins[generator.Name()] = generator
	return nil
}

func (pm *PluginManager) RegisterPlugin(plugin Generator) {
	pm.plugins[plugin.Name()] = plugin
}

// GetPlugin возвращает плагин по имени
func (pm *PluginManager) GetPlugin(name string) (Generator, error) {
	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("плагин %s не найден", name)
	}
	return plugin, nil
}

func RegisterPlugin(plugin Generator) {
	DefaultPluginManager.RegisterPlugin(plugin)
}
