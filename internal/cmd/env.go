package cmd

import (
	"fmt"
	"os"
)

type envAuthData struct {
	url   string
	token string
}

// envAuth returns the server URL and token from MMRUN_URL and MMRUN_TOKEN when
// both are set, or an error when either is missing.
func envAuth() (envAuthData, error) {
	url := os.Getenv("MMRUN_URL")
	token := os.Getenv("MMRUN_TOKEN")
	if url == "" || token == "" {
		return envAuthData{}, fmt.Errorf("MMRUN_URL and MMRUN_TOKEN must both be set for env-based auth")
	}
	return envAuthData{url: url, token: token}, nil
}
