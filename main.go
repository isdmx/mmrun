package main

import (
	"os"

	"github.com/dmitriev/mmrun/internal/cmd"
)

func main() {
	os.Exit(cmd.Run())
}
