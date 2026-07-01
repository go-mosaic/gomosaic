package validation

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestValidationPlugin_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/user.go")

	plugin := NewPlugin()
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/model")

	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}

	golden.AssertFile(t, files["validation_gen.go"].(*gomosaic.GoFile), "validation")
}

func TestValidationFull_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/full/full_user.go")

	plugin := NewPlugin()
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/model")

	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}

	golden.AssertFile(t, files["validation_gen.go"].(*gomosaic.GoFile), "validation_full")
}
