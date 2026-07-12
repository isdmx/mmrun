package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// ssoLogin opens the browser to the provider's OAuth flow and captures the
// token via a localhost redirect listener.
func ssoLogin(ctx context.Context, serverURL, provider string) (string, error) {
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer func() { _ = ln.Close() }()

	tokenCh := make(chan string, 1)
	srv := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := r.URL.Query().Get("token")
			if tok == "" {
				tok = r.Header.Get("Token")
			}
			_, _ = fmt.Fprintln(w, "Login complete. You may close this window.")
			select {
			case tokenCh <- tok:
			default:
			}
		}),
	}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	redirect := fmt.Sprintf("http://%s/", ln.Addr().String())
	authURL := fmt.Sprintf("%s/oauth/%s/login?redirect_to=%s", serverURL, provider, redirect)
	if err := openBrowser(ctx, authURL); err != nil {
		fmt.Printf("Open this URL to continue login:\n%s\n", authURL)
	}

	select {
	case tok := <-tokenCh:
		if tok == "" {
			return "", fmt.Errorf("no token received from SSO redirect")
		}
		return tok, nil
	case <-time.After(3 * time.Minute):
		return "", fmt.Errorf("SSO login timed out")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func openBrowser(ctx context.Context, url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{url}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		name, args = "xdg-open", []string{url}
	}
	//nolint:gosec // G204: launches the OS URL handler for an interactive SSO login
	return exec.CommandContext(ctx, name, args...).Start()
}
