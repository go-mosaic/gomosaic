package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"

	_ "github.com/go-mosaic/gomosaic/internal/plugin/http"
	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

const minArgsCount = 3

var (
	yellow = color.New(color.FgYellow).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
)

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
				"",
				"  name: Имя раширения которое будет генерировать код.",
				"  packages: Список пакетов в которых необходимо искать интерфейсы и структуры для генерации кода.",
				"  outputDir: Директория, в которую будет сохранен сгенерированный код.",
				"",
				"Флаги (опционально):",
				"",
				"  --modfile:  Путь к файлу go.mod для генерации кода (при запуске из под корня проекта флаг можно не указывать).",
			),
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) < minArgsCount {
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

				var dir string
				if modfile != "" {
					dir = filepath.Dir(modfile)
				} else {
					dir = filepath.Dir(os.Args[0])
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

func printError(cmd *cobra.Command, err error) {
	var isExitApp bool

	if err != nil {
		var merr *multierror.Error
		if errors.As(err, &merr) {
			merr.ErrorFormat = func(es []error) string {
				errorPoints := make([]string, 0, len(es))
				warningPoints := make([]string, 0, len(es))
				for _, err := range es {
					if gomosaic.IsErrWarning(err) {
						warningPoints = append(warningPoints, fmt.Sprintf("* %s", yellow(err)))
					} else {
						isExitApp = true
						errorPoints = append(errorPoints, fmt.Sprintf("* %s", red(err)))
					}
				}
				var text string
				if len(errorPoints) > 0 {
					text += fmt.Sprintf(
						"\n\n%d ошибки:\n\t%s\n\n",
						len(errorPoints), strings.Join(errorPoints, "\n\t"))
				}
				if len(warningPoints) > 0 {
					text += fmt.Sprintf(
						"\n\n%d предупреждения:\n\t%s\n\n",
						len(warningPoints), strings.Join(warningPoints, "\n\t"))
				}
				return text
			}
		}
		cmd.Println(err)
	}

	if isExitApp {
		os.Exit(1)
	}
}

func examples(ex ...string) string {
	for i := range ex {
		ex[i] = "  " + ex[i]
	}
	return strings.Join(ex, "\n")
}
