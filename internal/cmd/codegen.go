package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	_ "github.com/go-mosaic/gomosaic/internal/plugin/http"
	_ "github.com/go-mosaic/gomosaic/internal/plugin/logmiddleware"
	_ "github.com/go-mosaic/gomosaic/internal/plugin/metricmiddleware"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

const codegenMinArgsCount = 3

func CodegenCmd(postRun ...func()) *cobra.Command {
	var (
		modfile string
		cmd     = &cobra.Command{
			Use:   "codegen [flags] name packages outputDir",
			Short: "Команда codegen используется для автоматической генерации различного кода на языке Go (Golang) на основе переданных параметров.",
			Example: examples(
				"gomosaic codegen http-server ./internal/server",
				"",
				"Параметры:",
				"  name: Имя раширения которое будет генерировать код.",
				"  packages: Список пакетов в которых необходимо искать интерфейсы и структуры для генерации кода.",
				"  outputDir: Директория, в которую будет сохранен сгенерированный код.",
				"",
				"Флаги (опционально):",
				"  --modfile:  Путь к файлу go.mod для генерации кода (при запуске из под корня проекта флаг можно не указывать).",
			),
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) < codegenMinArgsCount {
					return fmt.Errorf("не верные аргументы")
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
					printError(cmd, err)
					return
				}

				nameTypesInfo, err := gomosaic.ParsePackage(dir, paths)
				if err != nil {
					printError(cmd, err)
					return
				}

				ctx := context.TODO()
				ctx = gomosaic.ContextWithOutputDir(ctx, outputDir)

				fs := gomosaic.NewFileSystem("dev", outputDir)
				cg := gomosaic.NewCodeGenerator(gomosaic.DefaultPluginManager, fs)

				outputFilenames, err := cg.Generate(ctx, moduleInfo, nameTypesInfo, pluginName)
				if err != nil {
					printError(cmd, err)
					return
				}

				cmd.Println("Генерация " + pluginName + " успешно завершена")
				for _, filename := range outputFilenames {
					cmd.Println(green("✓"), filename)
				}

				for _, fn := range postRun {
					fn()
				}
			},
		}
	)

	cmd.Flags().StringVar(&modfile, "modfile", "", "")

	cobra.CheckErr(cmd.Flags().MarkHidden("modfile"))
	return cmd
}
