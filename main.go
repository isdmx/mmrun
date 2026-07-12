// Command mmrun is a scriptable command-line client for Mattermost.
package main

import (
	"os"

	"github.com/isdmx/mmrun/internal/cmd"
)

func main() {
	os.Exit(cmd.Run())
}
