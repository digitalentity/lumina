package runner

import (
	"os"
	"os/exec"
)

// HostRunner executes commands directly on the host machine.
type HostRunner struct{}

// Run runs the tool on the host.
func (h HostRunner) Run(tool string, args []string, cwd string) error {
	cmd := exec.Command(tool, args...)
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Capture runs the tool on the host and returns its stdout.
func (h HostRunner) Capture(tool string, args []string, cwd string) ([]byte, error) {
	cmd := exec.Command(tool, args...)
	cmd.Dir = cwd
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

// CheckPresent checks if the tool exists on the host's PATH.
func (h HostRunner) CheckPresent(tool string) error {
	_, err := exec.LookPath(tool)
	return err
}
