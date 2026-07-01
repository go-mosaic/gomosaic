package client

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestHTTPClientPlugin_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/service.go")

	plugin := NewPlugin()
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/client")

	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}

	golden.AssertFile(t, files["client_gen.go"].(*gomosaic.GoFile), "http_client")
}

func TestHTTPClientFull_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/full/service.go")

	plugin := NewPlugin()
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/client")

	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}

	golden.AssertFile(t, files["client_gen.go"].(*gomosaic.GoFile), "http_client_full")
}
