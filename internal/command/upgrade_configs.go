package command

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/pkg/baur"
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
			Use:               "configs",
			Short:             "migrate baur configs to the newest version",
			ValidArgsFunction: cobra.NoFileCompletions,
		},
	}

	cmd.Run = cmd.run

	return &cmd
}

func (c *upgradeConfigsCmd) run(_ *cobra.Command, _ []string) {
	cwd, err := os.Getwd()
	exitOnErr(err)

	err = baur.NewCfgUpgrader(cwd).Upgrade()
	exitOnErr(err)

	stdout.Println("configuration files upgraded successfully")

	repo, err := findRepository()
	exitOnErr(err, "validation failed: loading repository config failed")

	_, err = argToApps(repo, []string{"*"})
	exitOnErr(err, "validation failed")
}
