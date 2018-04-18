package cmd

import (
	"fmt"
	"os"

	"github.com/simplesurance/sisubuild/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "sb",
	Short:   "sisubuild manages builds and artifacts in mono repositories.",
	Version: version.FullVerNr(),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
