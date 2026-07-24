package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func newCopyCmd(outputMode *string) *cobra.Command {
	return &cobra.Command{
		Use:               "copy <post-id>",
		Short:             "Copy post permalink to clipboard",
		Example:           "  mmrun copy <post-id>",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePostIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			url, err := resolveOpenURL(context.Background(), app, args[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), url)
			return copyToClipboard(url)
		},
	}
}

func copyToClipboard(text string) error {
	ctx := context.Background()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.CommandContext(ctx, "wl-copy")
		} else {
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		}
	default:
		cmd = exec.CommandContext(ctx, "clip")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
