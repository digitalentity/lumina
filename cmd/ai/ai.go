// Package ai implements the lumina ai subcommand group.
package ai

import (
	"github.com/spf13/cobra"
)

// AICmd is the parent command for AI-assisted writing tasks.
var AICmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-assisted writing helpers (check claims and citations)",
}
