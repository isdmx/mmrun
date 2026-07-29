package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

//nolint:gocognit // stdin batch pattern inlined per command per task spec
func newCopyCmd(outputMode *string) *cobra.Command {
	var noStdin bool
	cmd := &cobra.Command{
		Use:               "copy <post-id>",
		Short:             "Copy post permalink to clipboard",
		Example:           "  mmrun copy <post-id>",
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: completePostIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !noStdin && isStdinPipe() {
				targets, serr := readStdinTargets()
				if serr != nil {
					return serr
				}
				if len(targets) == 0 {
					return fmt.Errorf("no targets on stdin")
				}
				app, err := requireSession(*outputMode)
				if err != nil {
					return err
				}
				success, failed := 0, 0
				for _, id := range targets {
					url, uerr := resolveOpenURL(context.Background(), app, id)
					if uerr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: copy %s: %v\n", id, uerr)
						failed++
						continue
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), url)
					if cerr := copyToClipboard(url); cerr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: copy %s: %v\n", id, cerr)
						failed++
						continue
					}
					success++
				}
				if success == 0 {
					return fmt.Errorf("all %d targets failed", len(targets))
				}
				if failed > 0 {
					return ErrPartialSuccess
				}
				return nil
			}
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
	cmd.Flags().BoolVar(&noStdin, "no-stdin", false, "read post ID from positional arg even when piped")
	return cmd
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
