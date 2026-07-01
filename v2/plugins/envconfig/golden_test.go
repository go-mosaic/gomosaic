package envconfig

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestParseConfig(t *testing.T) {
	_, types := golden.ParseAndGenerate(t, ".", "fixtures/config.go")

	if len(types) == 0 {
		t.Fatal("типы не найдены")
	}
	for _, nt := range types {
		t.Logf("Тип: %s (пакет: %s)", nt.Name, nt.Package.Path)
		if nt.Type.Struct != nil {
			for _, f := range nt.Type.Struct.Fields {
				t.Logf("  Поле: %s %s", f.Name, f.Type.Name)
				for _, ann := range f.Annotations {
					t.Logf("    @%s %v", ann.Key, ann.Params)
				}
			}
		}
	}
}

func TestEnvConfigPlugin_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/config.go")

	t.Logf("модуль: %s, типов: %d", module.Path, len(types))

	plugin := NewPlugin()
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/config")

	t.Log("вызов Generate...")
	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}
	t.Log("Generate завершён")

	golden.AssertFile(t, files["env_config_gen.go"].(*gomosaic.GoFile), "env_config")
	_ = fmt.Sprintf
}

func TestEnvConfigNested_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/full/nested_config.go")

	plugin := NewPlugin()
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/config")

	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}

	golden.AssertFile(t, files["env_config_gen.go"].(*gomosaic.GoFile), "env_config_nested")
}
