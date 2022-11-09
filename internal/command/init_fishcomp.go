package command

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/fs"
)

type initFishCompCmd struct {
	cobra.Command
	stdout bool
}

func init() {
	initCmd.AddCommand(&newInitFishCompCmd().Command)
}

var initFishCompLongHelp = fmt.Sprintf(`
Generate and install a baur completion script for the fish shell.

If --stdout is passed, the completion script is written to STDOUT.
Otherwise it is written to the completion directory determined by the following
environment variables.

Environment Variables:
 %s			Script is written to %s
`,
	envVarXDGDataHome,
	filepath.Join("$"+envVarXDGDataHome, fishCompRelDataHomeDirFilepath()),
)

func newInitFishCompCmd() *initFishCompCmd {
	cmd := initFishCompCmd{
		Command: cobra.Command{
			Use:               "fishcomp",
			Short:             "generate and install a fish completion script",
			Long:              initFishCompLongHelp,
			GroupID:           initShellCompletionGroupID,
			ValidArgsFunction: cobra.NoFileCompletions,
		},
	}

	cmd.Flags().BoolVar(&cmd.stdout, "stdout", false, "write completion script to stdout")

	cmd.Run = cmd.run

	return &cmd
}

func fishCompletionFile() (string, error) {
	/*
		https://fishshell.com/docs/current/completions.html#where-to-put-completions

		By default, Fish searches the following for completions, using
		the first available file that it finds:

		A directory for end-users to keep their own completions,
		usually ~/.config/fish/completions (controlled by the
		XDG_CONFIG_HOME environment variable);

		A directory for systems administrators to install completions
		for all users on the system, usually /etc/fish/completions;

		A user-specified directory for third-party vendor completions,
		usually ~/.local/share/fish/vendor_completions.d (controlled by
		the XDG_DATA_HOME environment variable);

		A directory for third-party software vendors to ship their own
		completions for their software, usually
		/usr/share/fish/vendor_completions.d;

		The completions shipped with fish, usually installed in
		/usr/share/fish/completions; and

		Completions automatically generated from the operating system’s
		manual, usually stored in
		~/.local/share/fish/generated_completions. os.UserConfigDir
	*/

	dataHome, err := xdgDataHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(dataHome, fishCompRelDataHomeDirFilepath()), nil
}

func fishCompRelDataHomeDirFilepath() string {
	return filepath.Join("fish", "vendor_completions.d", "baur.fish")
}

func (c *initFishCompCmd) run(_ *cobra.Command, _ []string) {
	if c.stdout {
		err := rootCmd.GenFishCompletion(stdout, false)
		exitOnErr(err)
		return
	}

	complFile, err := fishCompletionFile()
	exitOnErr(err,
		"could not find fish completion directory,",
		"try rerunning the command with '--stdout'",
	)

	complDir := filepath.Dir(complFile)
	err = fs.Mkdir(complDir)
	exitOnErrf(err, "could not create directory %q", complDir)

	err = rootCmd.GenFishCompletionFile(complFile, true)
	exitOnErr(err, "generating completion script failed")

	stdout.Printf("fish completion script written to %s\n", term.Highlight(complFile))
}
