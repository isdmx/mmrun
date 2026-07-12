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
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer ln.Close()

	tokenCh := make(chan string, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := r.URL.Query().Get("token")
		if tok == "" {
			tok = r.Header.Get("Token")
		}
		fmt.Fprintln(w, "Login complete. You may close this window.")
		select {
		case tokenCh <- tok:
		default:
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	redirect := fmt.Sprintf("http://%s/", ln.Addr().String())
	authURL := fmt.Sprintf("%s/oauth/%s/login?redirect_to=%s", serverURL, provider, redirect)
	if err := openBrowser(authURL); err != nil {
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

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
