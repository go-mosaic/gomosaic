package main

import (
	"github.com/go-mosaic/gomosaic/pkg/cmd"
)

var Version = "dev"

func main() {
	cmd.Run(Version)
}
