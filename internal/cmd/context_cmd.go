package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/isdmx/mmrun/internal/output"
	"github.com/isdmx/mmrun/internal/session"
)

func newContextCmd(outputMode *string) *cobra.Command {
	ctxCmd := &cobra.Command{
		Use: "context", Short: "Manage session contexts",
		Example: "  mmrun context list\n  mmrun context use work",
	}

	ctxCmd.AddCommand(&cobra.Command{
		Use:   "current",
		Short: "Show the active context name",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			st, err := session.LoadAll()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), st.Current)
			return nil
		},
	})

	ctxCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all contexts, active marked with *",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runContextList(app, cmd.OutOrStdout())
		},
	})

	ctxCmd.AddCommand(&cobra.Command{
		Use:   "use <name>",
		Short: "Switch the active context",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runContextUse(args[0])
		},
	})

	var serverURL string
	addCmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new context (prompts for authentication)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runContextAdd(args[0], serverURL)
		},
	}
	addCmd.Flags().StringVar(&serverURL, "server", "", "server URL")
	ctxCmd.AddCommand(addCmd)

	var yes bool
	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a context (requires --yes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("remove requires --yes to confirm")
			}
			return runContextRemove(args[0])
		},
	}
	removeCmd.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	ctxCmd.AddCommand(removeCmd)

	return ctxCmd
}

func runContextList(app *appContext, w io.Writer) error {
	st, err := session.LoadAll()
	if err != nil {
		return err
	}
	res := output.Result{Title: "Contexts", Columns: []string{"name", "server", "active"}}
	for name, s := range st.Contexts {
		active := ""
		if name == st.Current {
			active = "*"
		}
		res.Rows = append(res.Rows, output.Row{"name": name, "server": s.ServerURL, "active": active})
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}

func runContextUse(name string) error {
	st, err := session.LoadAll()
	if err != nil {
		return err
	}
	if _, ok := st.Contexts[name]; !ok {
		return fmt.Errorf("context %q not found", name)
	}
	st.Current = name
	return session.SaveStore(st)
}

func runContextAdd(name, serverURL string) error {
	st, err := session.LoadAll()
	if err != nil && !errors.Is(err, session.ErrNoSession) {
		return err
	}
	if st == nil {
		st = &session.Store{Current: "default", Contexts: map[string]*session.Session{}}
	}
	if serverURL == "" {
		var rerr error
		serverURL, rerr = promptLine("Server URL: ")
		if rerr != nil {
			return rerr
		}
	}
	c := client.New(serverURL)
	if err := passwordLogin(context.Background(), c); err != nil {
		return err
	}
	u, err := c.Me(context.Background())
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}
	st.Contexts[name] = &session.Session{
		ServerURL:   serverURL,
		Token:       c.Token(),
		UserID:      u.Id,
		Username:    u.Username,
		ContextName: name,
	}
	return session.SaveStore(st)
}

func runContextRemove(name string) error {
	st, err := session.LoadAll()
	if err != nil {
		return err
	}
	if st.Current == name {
		return fmt.Errorf("cannot remove the active context %q; switch with 'context use' first", name)
	}
	delete(st.Contexts, name)
	return session.SaveStore(st)
}
