package command

import (
	"github.com/spf13/cobra"
)

const inputsLongHelp = `
The inputs groups provides command to show Build Inputs of applications.
`

var inputsCmd = &cobra.Command{
	Use:   "inputs",
	Short: "show build inputs",
	Long:  inputsLongHelp[1:],
}

func init() {
	rootCmd.AddCommand(inputsCmd)
}
