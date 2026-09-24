package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"lumina/internal/scaffold"
)

func TestLogCmdExecution(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-cmd-log-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Scaffold project and target
	if err := scaffold.Init(tempDir, "paper1"); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("git worktree failed: %v", err)
	}

	sig := &object.Signature{
		Name:  "Author Name",
		Email: "author@example.com",
		When:  time.Now(),
	}

	if _, err := wt.Add("src/paper1/manuscript.md"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if _, err := wt.Commit("feat: initial draft", &git.CommitOptions{Author: sig}); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Change cwd for command execution
	origWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer os.Chdir(origWd)

	t.Run("terminal mode", func(t *testing.T) {
		cmd := RootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"log", "paper1", "--terminal"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error running log --terminal: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "initial draft") {
			t.Errorf("expected terminal output to contain commit subject, got:\n%s", out)
		}
	})

	t.Run("html mode", func(t *testing.T) {
		outHTML := filepath.Join(tempDir, "build", "paper1-changelog.html")
		cmd := RootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"log", "paper1", "-o", outHTML})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error running log: %v", err)
		}

		content, err := os.ReadFile(outHTML)
		if err != nil {
			t.Fatalf("failed to read generated HTML: %v", err)
		}
		htmlStr := string(content)
		if !strings.Contains(htmlStr, "<!DOCTYPE html>") {
			t.Errorf("expected valid HTML output")
		}
		if !strings.Contains(htmlStr, "class=\"pagination\"") {
			t.Errorf("expected pagination in HTML output")
		}
		if strings.Contains(htmlStr, "ago") {
			t.Errorf("expected relative time ('ago') removed from HTML output")
		}
	})

	t.Run("markdown mode file", func(t *testing.T) {
		outMD := filepath.Join(tempDir, "build", "paper1-changelog.md")
		cmd := RootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"log", "paper1", "--markdown", "-o", outMD})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error running log --markdown: %v", err)
		}

		content, err := os.ReadFile(outMD)
		if err != nil {
			t.Fatalf("failed to read generated Markdown: %v", err)
		}
		mdStr := string(content)
		if !strings.Contains(mdStr, "# Manuscript Evolution") {
			t.Errorf("expected markdown header, got:\n%s", mdStr)
		}
		if !strings.Contains(mdStr, "**Revisions:** [1](#commit-") {
			t.Errorf("expected pagination line in markdown, got:\n%s", mdStr)
		}
	})

	t.Run("markdown stdout mode", func(t *testing.T) {
		cmd := RootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"log", "paper1", "--md", "-t"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error running log --md -t: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "# Manuscript Evolution") {
			t.Errorf("expected markdown stdout output, got:\n%s", out)
		}
	})
}
