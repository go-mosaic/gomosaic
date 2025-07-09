package cmd

import (
	"log"

	"github.com/spf13/cobra"

	basecmd "github.com/go-mosaic/gomosaic/internal/cmd"
)

func Run(version string) {
	log.SetFlags(0)
	cmd := &cobra.Command{Use: "gomosaic"}
	cmd.AddCommand(
		basecmd.CodegenCmd(),
		basecmd.DocgenCmd(),
	)
	cobra.CheckErr(cmd.Execute())
}
