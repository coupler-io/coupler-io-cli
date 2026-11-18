package main

import (
	"os"

	"github.com/coupler-io/coupler-io-cli/cmd"
)

// Set by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, date)
	os.Exit(cmd.Execute())
}
