// Package runner executes commands on host or inside docker containers.
package runner

import "lumina/internal/config"

// Runner defines the interface for running external CLI tools.
type Runner interface {
	// Run executes the tool with given arguments, using the specified working directory.
	// Output is streamed to stdout/stderr.
	Run(tool string, args []string, cwd string) error

	// Capture executes the tool and returns its stdout.
	Capture(tool string, args []string, cwd string) ([]byte, error)

	// CheckPresent checks if the tool is available.
	CheckPresent(tool string) error
}

// New creates a new Runner based on the configuration.
func New(cfg config.Config, root string) Runner {
	if cfg.Runner == "docker" {
		return &DockerRunner{
			Image: cfg.ToolsImage,
			Root:  root,
		}
	}
	return &HostRunner{}
}
