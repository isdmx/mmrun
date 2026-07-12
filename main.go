package main

import (
	"fmt"
	"os"

	"github.com/dmitriev/mmrun/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "mmrun:", err)
		os.Exit(cmd.ExitCode(err))
	}
}
