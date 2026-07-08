package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DockerRunner executes commands inside a Docker container.
type DockerRunner struct {
	Image string
	Root  string
}

// Run executes the tool inside the Docker container.
func (d *DockerRunner) Run(tool string, args []string, cwd string) error {
	cmd := d.dockerCmd(tool, args, cwd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Capture executes the tool inside the Docker container and returns its stdout.
func (d *DockerRunner) Capture(tool string, args []string, cwd string) ([]byte, error) {
	cmd := d.dockerCmd(tool, args, cwd)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func (d *DockerRunner) dockerCmd(tool string, args []string, cwd string) *exec.Cmd {
	uid := os.Getuid()
	gid := os.Getgid()
	cleanRoot := filepath.Clean(d.Root)

	containerWd := "/workspace"
	if cwd != "" {
		absCwd, err := filepath.Abs(cwd)
		if err == nil {
			cleanCwd := filepath.Clean(absCwd)
			if strings.HasPrefix(cleanCwd, cleanRoot) {
				rel, err := filepath.Rel(cleanRoot, cleanCwd)
				if err == nil {
					containerWd = filepath.Join("/workspace", rel)
				}
			}
		}
	}

	rewrittenArgs := make([]string, len(args))
	for i, arg := range args {
		rewrittenArgs[i] = rewriteArg(arg, cleanRoot)
	}

	dockerArgs := []string{
		"run",
		"--rm",
		"-u", fmt.Sprintf("%d:%d", uid, gid),
		"-v", fmt.Sprintf("%s:/workspace", cleanRoot),
		"-w", containerWd,
		"-e", "HOME=/tmp",
		d.Image,
		tool,
	}
	dockerArgs = append(dockerArgs, rewrittenArgs...)

	return exec.Command("docker", dockerArgs...)
}

// rewriteArg rewrites arg to its /workspace-relative equivalent if it is an
// absolute path rooted under cleanRoot. Unlike a blind substring replace,
// this leaves untouched any argument that merely happens to contain
// cleanRoot as a substring without actually being a path under it (e.g. a
// flag value or filename that coincidentally embeds the root directory's
// name).
func rewriteArg(arg, cleanRoot string) string {
	cleanArg := filepath.Clean(arg)
	if cleanArg == cleanRoot {
		return "/workspace"
	}
	if rel, ok := strings.CutPrefix(cleanArg, cleanRoot+string(filepath.Separator)); ok {
		return filepath.Join("/workspace", rel)
	}
	return arg
}

// CheckPresent checks that docker is on PATH and the tools image exists.
func (d *DockerRunner) CheckPresent(tool string) error {
	// First check if docker is available on the host
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found on PATH: %w", err)
	}

	// Then check if the image is available
	cmd := exec.Command("docker", "image", "inspect", d.Image)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker image %q not found (run 'make image' to build it): %w", d.Image, err)
	}

	return nil
}
