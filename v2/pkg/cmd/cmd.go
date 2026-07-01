// Package cmd предоставляет CLI для генератора.
//
// Пример использования:
//
//	package main
//
//	import (
//	    "log"
//
//	    "github.com/go-mosaic/gomosaic/v2/pkg/cmd"
//	    "github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
//	    "github.com/go-mosaic/gomosaic/v2/plugins/middleware"
//	)
//
//	func main() {
//	    builder := gomosaic.NewBuilder("myapp", "1.0.0", "./output")
//
//	    builder.WithPlugins(
//	        middleware.NewPlugin(middleware.Config{
//	            Name:           "log-middleware",
//	            MiddlewareName: "Log",
//	            Annotation:     "log",
//	            // ...
//	        }),
//	        middleware.NewPlugin(middleware.Config{
//	            Name:           "metric-middleware",
//	            MiddlewareName: "Metric",
//	            Annotation:     "metric",
//	            // ...
//	        }),
//	    )
//
//	    cmd.Run(builder)
//	}
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func Run(version string, builder *gomosaic.Builder) {
	log.SetFlags(0)

	cmd := &cobra.Command{Use: "gomosaic"}

	cmd.AddCommand(CodegenCmd(version, builder))

	cobra.CheckErr(cmd.Execute())
}

func CodegenCmd(version string, builder *gomosaic.Builder) *cobra.Command {
	var (
		modfile string
		cmd     = &cobra.Command{
			Use:   "codegen [flags] name packages outputDir",
			Short: "Генерация кода",
			Example: examples(
				"gomosaic codegen http-server-chi ./internal/server",
				"",
				"Параметры:",
				"  name:       Имя плагина.",
				"  packages:   Список пакетов для поиска интерфейсов.",
				"  outputDir:  Директория для сохранения сгенерированного кода.",
				"",
				"Флаги:",
				"  --modfile:  Путь к go.mod (опционально).",
			),
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) < 3 {
					return fmt.Errorf("требуется минимум 3 аргумента: name packages outputDir")
				}
				return nil
			},
			Run: func(cmd *cobra.Command, args []string) {
				pluginName := args[0]
				paths := args[1 : len(args)-1]
				outputDir := args[len(args)-1]

				outputDir, err := filepath.Abs(outputDir)
				if err != nil {
					cmd.Println(err)
					return
				}

				if modfile != "" {
					modfile, err = filepath.Abs(modfile)
					if err != nil {
						cmd.Println(err)
						return
					}
				}

				var dir string

				if modfile == "" {
					dir = filepath.Dir(os.Args[0])
				} else {
					dir = filepath.Dir(modfile)
				}

				moduleInfo, err := gomosaic.LoadModuleInfo(modfile)
				if err != nil {
					cmd.Println("Ошибка загрузки модуля:", err)
					return
				}

				nameTypesInfo, err := gomosaic.ParsePackage(dir, paths)
				if err != nil {
					cmd.Println("Ошибка парсинга пакета:", err)
					return
				}

				ctx := context.TODO()
				ctx = gomosaic.ContextWithOutputDir(ctx, outputDir)

				fs := gomosaic.NewFileSystem(version, outputDir)

				gen := builder.Build(fs)

				outputFilenames, err := gen.Generate(ctx, moduleInfo, nameTypesInfo, pluginName)
				if err != nil {
					cmd.Println("Ошибка генерации:", err)
					return
				}

				cmd.Println("Генерация", pluginName, "успешно завершена")

				for _, filename := range outputFilenames {
					cmd.Println("✓", filename)
				}
			},
		}
	)

	cmd.Flags().StringVar(&modfile, "modfile", "", "путь к go.mod")
	cobra.CheckErr(cmd.Flags().MarkHidden("modfile"))

	return cmd
}

func examples(ex ...string) string {
	var result strings.Builder

	for i := range ex {
		if i > 0 {
			result.WriteString("\n")
		}

		result.WriteString("  " + ex[i])
	}

	return result.String()
}
