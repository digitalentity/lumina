package runner

import (
	"lumina/internal/config"
	"testing"
)

func TestNew(t *testing.T) {
	cfg := config.Config{
		Runner:     "docker",
		ToolsImage: "custom:v1",
	}
	r := New(cfg, "/dummy")
	dr, ok := r.(*DockerRunner)
	if !ok {
		t.Fatalf("expected DockerRunner, got %T", r)
	}
	if dr.Image != "custom:v1" {
		t.Errorf("expected custom:v1, got %s", dr.Image)
	}

	cfg.Runner = "host"
	r = New(cfg, "/dummy")
	_, ok = r.(*HostRunner)
	if !ok {
		t.Fatalf("expected HostRunner, got %T", r)
	}
}

func TestHostRunner(t *testing.T) {
	r := &HostRunner{}
	if err := r.CheckPresent("go"); err != nil {
		t.Errorf("expected go to be present, got %v", err)
	}

	err := r.Run("go", []string{"version"}, ".")
	if err != nil {
		t.Errorf("expected go version to run successfully, got %v", err)
	}
}

func TestRewriteArg(t *testing.T) {
	root := "/home/user/manuscript"

	for _, tc := range []struct {
		name string
		arg  string
		want string
	}{
		{"root itself", "/home/user/manuscript", "/workspace"},
		{"path under root", "/home/user/manuscript/figures/diagram.png", "/workspace/figures/diagram.png"},
		{"unrelated absolute path", "/home/user/manuscript-notes/foo", "/home/user/manuscript-notes/foo"},
		{"non-path flag value", "--pdf-engine=xelatex", "--pdf-engine=xelatex"},
		{"relative path", "manuscript.md", "manuscript.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := rewriteArg(tc.arg, root); got != tc.want {
				t.Errorf("rewriteArg(%q, %q) = %q, expected %q", tc.arg, root, got, tc.want)
			}
		})
	}
}

func TestHostRunnerCapture(t *testing.T) {
	r := &HostRunner{}
	out, err := r.Capture("go", []string{"version"}, ".")
	if err != nil {
		t.Fatalf("expected Capture to run, got %v", err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty output from Capture")
	}
}
