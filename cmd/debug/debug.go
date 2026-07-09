// Package debug implements the lumina debug subcommand group.
package debug

import (
	"github.com/spf13/cobra"
)

// DebugCmd is the parent command for debug tasks.
var DebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug helpers for checking manuscript parsing and extraction",
}
