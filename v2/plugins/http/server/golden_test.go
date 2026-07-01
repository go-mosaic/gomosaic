package server

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestHTTPServerPlugin_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/service.go")

	plugin := NewPlugin(&ChiStrategy{})
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/controller")

	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}

	golden.AssertFile(t, files["server_chi_gen.go"].(*gomosaic.GoFile), "http_server_chi")
}

func TestHTTPServerFull_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/full/service.go")

	plugin := NewPlugin(&ChiStrategy{})
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/controller")

	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}

	golden.AssertFile(t, files["server_chi_gen.go"].(*gomosaic.GoFile), "http_server_chi_full")
}
