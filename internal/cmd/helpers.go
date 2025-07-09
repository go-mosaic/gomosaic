package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

var (
	yellow = color.New(color.FgYellow).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
)

func examples(ex ...string) string {
	for i := range ex {
		ex[i] = "  " + ex[i]
	}
	return strings.Join(ex, "\n")
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
