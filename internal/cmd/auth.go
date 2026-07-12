package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/dmitriev/mmrun/internal/client"
	"github.com/dmitriev/mmrun/internal/config"
	"github.com/dmitriev/mmrun/internal/session"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCmd(outputMode *string) *cobra.Command {
	auth := &cobra.Command{Use: "auth", Short: "Manage authentication"}
	auth.AddCommand(newLoginCmd())
	auth.AddCommand(newLogoutCmd())
	auth.AddCommand(newAuthStatusCmd())
	return auth
}

// needsMFA reports whether a login error message indicates MFA is required.
func needsMFA(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "mfa") || strings.Contains(m, "multi-factor")
}

func promptLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	s, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(s), err
}

func promptSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	return strings.TrimSpace(string(b)), err
}

func newLoginCmd() *cobra.Command {
	var serverURL, token, sso string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in and store a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if serverURL == "" {
				var err error
				if serverURL, err = promptLine("Server URL: "); err != nil {
					return err
				}
			}
			c := client.New(serverURL)

			switch {
			case token != "":
				c.SetToken(token)
			case sso != "":
				t, err := ssoLogin(ctx, serverURL, sso)
				if err != nil {
					return err
				}
				c.SetToken(t)
			default:
				if err := passwordLogin(ctx, c); err != nil {
					return err
				}
			}

			u, err := c.Me(ctx)
			if err != nil {
				return fmt.Errorf("token validation failed: %w", err)
			}
			if err := session.Save(&session.Session{
				ServerURL: serverURL,
				Token:     c.Token(),
				UserID:    u.Id,
			}); err != nil {
				return err
			}
			cfg, _ := config.Load()
			cfg.ServerURL = serverURL
			_ = config.Save(cfg)

			fmt.Fprintf(os.Stderr, "Logged in as %s\n", u.Username)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "", "server URL")
	cmd.Flags().StringVar(&token, "token", "", "personal access token")
	cmd.Flags().StringVar(&sso, "sso", "", "SSO provider (gitlab|google|office365|saml)")
	return cmd
}

func passwordLogin(ctx context.Context, c *client.Client) error {
	loginID, err := promptLine("Username or email: ")
	if err != nil {
		return err
	}
	password, err := promptSecret("Password: ")
	if err != nil {
		return err
	}
	_, err = c.Login(ctx, loginID, password)
	if err == nil {
		return nil
	}
	if !needsMFA(err.Error()) {
		return err
	}
	mfa, err := promptLine("MFA token: ")
	if err != nil {
		return err
	}
	_, err = c.LoginWithMFA(ctx, loginID, password, mfa)
	return err
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke the session and remove local credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := session.Load()
			if err != nil {
				return err
			}
			c := client.NewWithToken(sess.ServerURL, sess.Token)
			if sess.SessionID != "" {
				_ = c.RevokeSession(context.Background(), sess.UserID, sess.SessionID)
			}
			return session.Clear()
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current session status",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := session.Load()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "server: %s\nuser_id: %s\n", sess.ServerURL, sess.UserID)
			return nil
		},
	}
}
