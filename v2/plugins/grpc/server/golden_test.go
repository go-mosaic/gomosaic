package server

import (
	"context"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/internal/golden"
	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestGRPCServerPlugin_Golden(t *testing.T) {
	module, types := golden.ParseAndGenerate(t, ".", "fixtures/service.go")

	plugin := NewPlugin()
	ctx := gomosaic.ContextWithOutputDir(context.Background(), module.Dir+"/internal/grpc")

	files, err := plugin.Generate(ctx, module, types)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if files == nil || len(files) == 0 {
		t.Fatal("пустой результат генерации")
	}

	golden.AssertFile(t, files["grpc_server_gen.go"].(*gomosaic.GoFile), "grpc_server")
}
