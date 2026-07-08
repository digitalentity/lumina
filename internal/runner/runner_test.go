package runner

import (
	"testing"
	"lumina/internal/config"
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
