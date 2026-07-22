package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd(outputMode *string) *cobra.Command {
	var emoji, text string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Set your online status and custom status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			status := ""
			if len(args) > 0 {
				status = args[0]
			}
			if status != "" {
				return app.api.UpdateStatus(context.Background(), app.userID, status)
			}
			if emoji != "" || text != "" {
				return app.api.UpdateCustomStatus(context.Background(), app.userID, emoji, text)
			}
			return fmt.Errorf("specify status (online|away|dnd|offline) or --emoji/--text")
		},
	}
	cmd.Flags().StringVar(&emoji, "emoji", "", "custom status emoji")
	cmd.Flags().StringVar(&text, "text", "", "custom status text")
	return cmd
}
