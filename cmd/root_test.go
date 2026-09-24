package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmdHelp(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error running --help: %v", err)
	}

	out := buf.String()
	expectedSubstrings := []string{
		"Manuscript Commands:",
		"build",
		"lit",
		"text",
		"Project Commands:",
		"init",
		"clean",
		"Examples:",
		"lumina init paper1",
		"-h, --help",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected root help to contain %q, but got:\n%s", s, out)
		}
	}
}

func TestBuildCmdHelp(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"build", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error running build --help: %v", err)
	}

	out := buf.String()
	expectedSubstrings := []string{
		"Compile manuscript target into one or more output formats",
		"Usage:",
		"lumina build <target> [flags]",
		"--pdf",
		"--docx",
		"--tex",
		"--zip",
		"--pub",
		"--preprocess",
		"-f, --force",
		"--pdf-engine",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected build help to contain %q, but got:\n%s", s, out)
		}
	}
}

func TestLitCmdHelp(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"lit", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error running lit --help: %v", err)
	}

	out := buf.String()
	expectedSubstrings := []string{
		"Manage literature and bibliography for manuscript targets",
		"check",
		"fmt",
		"prune",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected lit help to contain %q, but got:\n%s", s, out)
		}
	}
}

func TestLitPruneHelp(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"lit", "prune", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error running lit prune --help: %v", err)
	}

	out := buf.String()
	expectedSubstrings := []string{
		"--no-dry-run",
		"-y, --yes",
		"lumina lit prune paper1",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected lit prune help to contain %q, but got:\n%s", s, out)
		}
	}
}

func TestTextCmdHelp(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"text", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error running text --help: %v", err)
	}

	out := buf.String()
	expectedSubstrings := []string{
		"Manage text quality and prose standards for manuscript targets",
		"words",
		"fmt",
		"lint",
		"detect",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected text help to contain %q, but got:\n%s", s, out)
		}
	}
}

func TestTextDetectHelp(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"text", "detect", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error running text detect --help: %v", err)
	}

	out := buf.String()
	expectedSubstrings := []string{
		"-t, --threshold",
		"-d, --detail",
		"-j, --json",
		"ai",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected text detect help to contain %q, but got:\n%s", s, out)
		}
	}
}

func TestCleanCmdArgs(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"clean", "unexpected-arg"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when running clean with extra arguments, got nil")
	}
}

func TestLogCmdHelp(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"log", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error running log --help: %v", err)
	}

	out := buf.String()
	expectedSubstrings := []string{
		"lumina log <target> [flags]",
		"-o, --output",
		"--pdf",
		"-t, --terminal",
		"--stat",
		"--since",
		"-n, --max-count",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected log help to contain %q, but got:\n%s", s, out)
		}
	}
}

