// Package cmd implements the mmrun command-line interface (cobra commands).
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/isdmx/mmrun/internal/version"
)

type globalOpts struct {
	outputMode string
}

var validOutputModes = map[string]bool{"auto": true, "human": true, "ai": true, "json": true}

func newRootCmd(opts *globalOpts) *cobra.Command {
	root := &cobra.Command{
		Use:           "mmrun",
		Short:         "Scriptable Mattermost CLI client",
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !validOutputModes[opts.outputMode] {
				return fmt.Errorf("invalid --output %q: use auto, human, ai, or json", opts.outputMode)
			}
			return nil
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.PersistentFlags().StringVarP(&opts.outputMode, "output", "o", "auto", "output mode: auto|human|ai|json")
	root.AddCommand(newMeCmd(&opts.outputMode))
	root.AddCommand(newAuthCmd(&opts.outputMode))
	root.AddCommand(newTeamCmd(&opts.outputMode))
	root.AddCommand(newChannelCmd(&opts.outputMode))
	root.AddCommand(newUserCmd(&opts.outputMode))
	root.AddCommand(newPostCmd(&opts.outputMode))
	root.AddCommand(newReplyCmd(&opts.outputMode))
	root.AddCommand(newReadCmd(&opts.outputMode))
	root.AddCommand(newThreadCmd(&opts.outputMode))
	root.AddCommand(newMarkReadCmd(&opts.outputMode))
	root.AddCommand(newSearchCmd(&opts.outputMode))
	root.AddCommand(newFileCmd(&opts.outputMode))
	root.AddCommand(newTailCmd(&opts.outputMode))
	root.AddCommand(newReactCmd(&opts.outputMode))
	root.AddCommand(newEditCmd(&opts.outputMode))
	root.AddCommand(newMentionsCmd(&opts.outputMode))
	root.AddCommand(newPinCmd(&opts.outputMode))
	root.AddCommand(newPinnedCmd(&opts.outputMode))
	root.AddCommand(newStatsCmd(&opts.outputMode))
	root.AddCommand(newFlaggedCmd(&opts.outputMode))
	root.AddCommand(newFlagCmd(&opts.outputMode))
	root.AddCommand(newVersionCmd(&opts.outputMode))
	root.AddCommand(newConfigCmd(&opts.outputMode))
	root.AddCommand(newOpenCmd(&opts.outputMode))
	root.AddCommand(newCopyCmd(&opts.outputMode))
	root.AddCommand(newContextCmd(&opts.outputMode))
	return root
}

// Run executes the CLI, prints any error in the active output format, and
// returns the process exit code.
func Run() int {
	opts := &globalOpts{}
	err := newRootCmd(opts).Execute()
	if err != nil {
		if client.StatusCode(err) == 401 {
			if _, envErr := envAuth(); envErr != nil {
				if _, rerr := reLogin(); rerr == nil {
					err = newRootCmd(opts).Execute()
				}
			}
		}
		if err != nil {
			printError(err, opts.outputMode)
			return ExitCode(err)
		}
	}
	return 0
}

// printError writes err to stderr, as a JSON object when the output mode is
// "json", otherwise as a plain prefixed line.
func printError(err error, outputMode string) {
	if msg := friendlyMsg(err); msg != "" {
		fmt.Fprintln(os.Stderr, "mmrun:", msg)
	}
	if outputMode == "json" {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return
	}
	fmt.Fprintln(os.Stderr, "mmrun:", err)
}
