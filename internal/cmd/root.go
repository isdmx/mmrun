package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type globalOpts struct {
	outputMode string
}

var validOutputModes = map[string]bool{"auto": true, "human": true, "ai": true, "json": true}

func newRootCmd(opts *globalOpts) *cobra.Command {
	root := &cobra.Command{
		Use:           "mmrun",
		Short:         "Scriptable Mattermost CLI client",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !validOutputModes[opts.outputMode] {
				return fmt.Errorf("invalid --output %q: use auto, human, ai, or json", opts.outputMode)
			}
			return nil
		},
	}
	root.PersistentFlags().StringVarP(&opts.outputMode, "output", "o", "auto", "output mode: auto|human|ai|json")
	root.AddCommand(newMeCmd(&opts.outputMode))
	root.AddCommand(newAuthCmd(&opts.outputMode))
	root.AddCommand(newTeamCmd(&opts.outputMode))
	root.AddCommand(newChannelCmd(&opts.outputMode))
	root.AddCommand(newPostCmd(&opts.outputMode))
	root.AddCommand(newReadCmd(&opts.outputMode))
	root.AddCommand(newSearchCmd(&opts.outputMode))
	root.AddCommand(newFileCmd(&opts.outputMode))
	root.AddCommand(newTailCmd(&opts.outputMode))
	return root
}

// Run executes the CLI, prints any error in the active output format, and
// returns the process exit code.
func Run() int {
	opts := &globalOpts{}
	err := newRootCmd(opts).Execute()
	if err == nil {
		return 0
	}
	printError(err, opts.outputMode)
	return ExitCode(err)
}

// printError writes err to stderr, as a JSON object when the output mode is
// "json", otherwise as a plain prefixed line.
func printError(err error, outputMode string) {
	if outputMode == "json" {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return
	}
	fmt.Fprintln(os.Stderr, "mmrun:", err)
}
