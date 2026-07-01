package gomosaic

import "fmt"

// PluginRegistry — реестр плагинов.
type PluginRegistry struct {
	plugins map[string]Generator
}

// NewPluginRegistry создает новый пустой реестр плагинов.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Generator),
	}
}

// Register регистрирует плагин в реестре.
func (r *PluginRegistry) Register(p Generator) error {
	name := p.Name()
	if name == "" {
		return fmt.Errorf("плагин не может иметь пустое имя")
	}

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("плагин с именем %q уже зарегистрирован", name)
	}

	r.plugins[name] = p

	return nil
}

// MustRegister регистрирует плагин или паникует при ошибке.
func (r *PluginRegistry) MustRegister(p Generator) {
	if err := r.Register(p); err != nil {
		panic(err)
	}
}

// MustRegisterAll регистрирует несколько плагинов или паникует при ошибке.
func (r *PluginRegistry) MustRegisterAll(plugins ...Generator) {
	if err := r.RegisterAll(plugins...); err != nil {
		panic(err)
	}
}

// RegisterAll регистрирует несколько плагинов.
func (r *PluginRegistry) RegisterAll(plugins ...Generator) error {
	for _, p := range plugins {
		if err := r.Register(p); err != nil {
			return err
		}
	}

	return nil
}

// Get возвращает плагин по имени.
func (r *PluginRegistry) Get(name string) (Generator, error) {
	p, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("плагин %q не найден", name)
	}

	return p, nil
}

// List возвращает список всех зарегистрированных плагинов.
func (r *PluginRegistry) List() []Generator {
	result := make([]Generator, 0, len(r.plugins))
	for _, p := range r.plugins {
		result = append(result, p)
	}

	return result
}

// Names возвращает имена всех зарегистрированных плагинов.
func (r *PluginRegistry) Names() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}

	return names
}
