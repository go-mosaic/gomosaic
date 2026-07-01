// Package plugin содержит документацию по созданию плагинов для gomosaic.
//
// В gomosaic плагины регистрируются через PluginRegistry.
//
// Пример создания плагина:
//
//	type MyPlugin struct{}
//
//	func (p *MyPlugin) Name() string { return "my-plugin" }
//
//	func (p *MyPlugin) Generate(ctx context.Context, module *gomosaic.ModuleInfo, types []*gomosaic.NameTypeInfo) (map[string]gomosaic.File, error) {
//	    // генерация кода
//	}
//
//	// Регистрация:
//	builder := gomosaic.NewBuilder("myapp", "1.0.0", "./output")
//	builder.WithPlugin(&MyPlugin{})
package plugin
