package cmd

import (
	"github.com/spf13/cobra"
)

type globalOpts struct {
	outputMode string
	configPath string
	verbose    bool
}

func newRootCmd() *cobra.Command {
	opts := &globalOpts{}
	root := &cobra.Command{
		Use:           "mmrun",
		Short:         "Scriptable Mattermost CLI client",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVarP(&opts.outputMode, "output", "o", "auto", "output mode: auto|human|ai|json")
	root.PersistentFlags().StringVar(&opts.configPath, "config", "", "path to config file")
	root.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose logging")
	return root
}

// Execute is the entrypoint used by main.
func Execute() error {
	return newRootCmd().Execute()
}
