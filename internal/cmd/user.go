package cmd

import (
	"context"
	"io"
	"strings"

	"github.com/isdmx/mmrun/internal/output"
	"github.com/spf13/cobra"
)

func newUserCmd(outputMode *string) *cobra.Command {
	user := &cobra.Command{Use: "user", Short: "User operations"}

	var teamName string
	search := &cobra.Command{
		Use:   "search <term>",
		Short: "Search users by name or username (e.g. to find someone to message)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			teamID := ""
			if teamName != "" {
				id, _, terr := app.resolveTeam(cmd.Context(), teamName)
				if terr != nil {
					return terr
				}
				teamID = id
			}
			return runUserSearch(app, strings.Join(args, " "), teamID, cmd.OutOrStdout())
		},
	}
	search.Flags().StringVar(&teamName, "team", "", "restrict the search to members of this team")
	user.AddCommand(search)
	return user
}

func runUserSearch(app *appContext, term, teamID string, w io.Writer) error {
	ctx := context.Background()
	users, err := app.api.SearchUsers(ctx, term, teamID, 50)
	if err != nil {
		return err
	}
	res := output.Result{Title: "Users", Columns: []string{"username", "name", "email", "id"}}
	for _, u := range users {
		res.Rows = append(res.Rows, output.Row{
			"username": "@" + u.Username,
			"name":     u.GetFullName(),
			"email":    u.Email,
			"id":       u.Id,
		})
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}
