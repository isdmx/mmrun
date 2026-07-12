package cmd

import (
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
	"github.com/isdmx/mmrun/internal/version"
)

func newVersionCmd(outputMode *string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and build information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Get()
			res := output.Result{
				Title:   "Version",
				Columns: []string{"field", "value"},
				Rows: []output.Row{
					{"field": "version", "value": info.Version},
					{"field": "commit", "value": info.Commit},
					{"field": "date", "value": info.Date},
					{"field": "go", "value": info.GoVersion},
					{"field": "platform", "value": info.OS + "/" + info.Arch},
				},
			}
			return output.New(*outputMode, stdoutFile(cmd.OutOrStdout())).Render(cmd.OutOrStdout(), res)
		},
	}
}
