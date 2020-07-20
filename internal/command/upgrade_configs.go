package command

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1"
)

func init() {
	upgradeCmd.AddCommand(&newUpgradeConfigsCmd().Command)
}

type upgradeConfigsCmd struct {
	cobra.Command
}

func newUpgradeConfigsCmd() *upgradeConfigsCmd {
	cmd := upgradeConfigsCmd{
		Command: cobra.Command{
			Use:   "configs",
			Short: "upgrade baur configs from config version 4 to 5",
		},
	}

	cmd.Run = cmd.run

	return &cmd
}

func (c *upgradeConfigsCmd) run(cmd *cobra.Command, _ []string) {
	cwd, err := os.Getwd()
	exitOnErr(err)

	err = baur.NewCfgUpgrader(cwd).Upgrade()
	exitOnErr(err)

	stdout.Println("configuration files upgraded successfully")
}
