package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"lumina/internal/config"
	"lumina/internal/manuscript"
)

func TestExtractChangelog(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-git-changelog-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	paper1Dir := filepath.Join(tempDir, "src", "paper1")
	paper2Dir := filepath.Join(tempDir, "src", "paper2")
	_ = os.MkdirAll(paper1Dir, 0755)
	_ = os.MkdirAll(paper2Dir, 0755)

	m1Path := filepath.Join(paper1Dir, "manuscript.md")
	meta1Path := filepath.Join(paper1Dir, "metadata.yaml")
	_ = os.WriteFile(meta1Path, []byte("title: \"Distributed Consensus Study\"\n"), 0644)

	m2Path := filepath.Join(paper2Dir, "manuscript.md")

	sig := &object.Signature{
		Name:  "Test Author",
		Email: "author@example.com",
		When:  time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
	}

	// Commit 1: Initial paper1
	content1 := "# Introduction\n\nThis is the initial draft of our study on consensus protocols."
	if err := os.WriteFile(m1Path, []byte(content1), 0644); err != nil {
		t.Fatalf("write m1: %v", err)
	}
	if _, err := wt.Add("src/paper1/manuscript.md"); err != nil {
		t.Fatalf("wt add: %v", err)
	}
	if _, err := wt.Commit("feat: initial manuscript draft", &git.CommitOptions{Author: sig}); err != nil {
		t.Fatalf("wt commit 1: %v", err)
	}

	// Commit 2: Modify paper1
	sig2 := &object.Signature{
		Name:  "Test Author",
		Email: "author@example.com",
		When:  time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC),
	}
	content2 := "# Introduction\n\nThis is the revised draft of our extensive study on consensus protocols.\n\n## Methodology\n\nWe benchmark Raft and Paxos."
	if err := os.WriteFile(m1Path, []byte(content2), 0644); err != nil {
		t.Fatalf("write m1 v2: %v", err)
	}
	if _, err := wt.Add("src/paper1/manuscript.md"); err != nil {
		t.Fatalf("wt add v2: %v", err)
	}
	if _, err := wt.Commit("docs: expand methodology and revise intro", &git.CommitOptions{Author: sig2}); err != nil {
		t.Fatalf("wt commit 2: %v", err)
	}

	// Commit 3: Unrelated commit to paper2 (must NOT appear in paper1's changelog)
	sig3 := &object.Signature{
		Name:  "Other Author",
		Email: "other@example.com",
		When:  time.Date(2026, 1, 10, 15, 0, 0, 0, time.UTC),
	}
	if err := os.WriteFile(m2Path, []byte("# Paper Two\n\nUnrelated content."), 0644); err != nil {
		t.Fatalf("write m2: %v", err)
	}
	if _, err := wt.Add("src/paper2/manuscript.md"); err != nil {
		t.Fatalf("wt add m2: %v", err)
	}
	if _, err := wt.Commit("feat: start paper two", &git.CommitOptions{Author: sig3}); err != nil {
		t.Fatalf("wt commit 3: %v", err)
	}

	// Commit 4: Add bibliography entries to paper1
	bib1Path := filepath.Join(paper1Dir, "references.bib")
	bibContent := `@article{lamport1978,
  author = {Leslie Lamport},
  title = {Time, Clocks, and the Ordering of Events in a Distributed System},
  year = {1978},
  journal = {Communications of the ACM}
}`
	if err := os.WriteFile(bib1Path, []byte(bibContent), 0644); err != nil {
		t.Fatalf("write bib: %v", err)
	}
	if _, err := wt.Add("src/paper1/references.bib"); err != nil {
		t.Fatalf("wt add bib: %v", err)
	}
	sig4 := &object.Signature{
		Name:  "Test Author",
		Email: "author@example.com",
		When:  time.Date(2026, 1, 12, 16, 0, 0, 0, time.UTC),
	}
	if _, err := wt.Commit("chore: add Lamport citation", &git.CommitOptions{Author: sig4}); err != nil {
		t.Fatalf("wt commit 4: %v", err)
	}

	ms := &manuscript.Manuscript{
		Root:        tempDir,
		ProjectRoot: tempDir,
		Target:      "paper1",
		TargetDir:   paper1Dir,
		Source:      m1Path,
		BibPath:     bib1Path,
		BuildDir:    filepath.Join(tempDir, "build"),
		Stem:        "paper1",
		Config:      config.Config{},
		RawMeta:     map[string]any{"title": "Distributed Consensus Study"},
	}

	cl, err := ExtractChangelog(ms, Options{})
	if err != nil {
		t.Fatalf("ExtractChangelog failed: %v", err)
	}

	// Must have 3 commits (excluding paper2 commit)
	if cl.TotalCommits != 3 {
		t.Fatalf("expected 3 revisions for paper1, got %d", cl.TotalCommits)
	}

	if cl.Title != "Distributed Consensus Study" {
		t.Errorf("expected title 'Distributed Consensus Study', got %q", cl.Title)
	}

	// Initial commit should be first
	initial := cl.Revisions[0]
	if initial.Subject != "feat: initial manuscript draft" {
		t.Errorf("expected initial commit subject 'feat: initial manuscript draft', got %q", initial.Subject)
	}

	// Second commit
	second := cl.Revisions[1]
	if second.Subject != "docs: expand methodology and revise intro" {
		t.Errorf("expected second commit subject 'docs: expand methodology and revise intro', got %q", second.Subject)
	}

	// Third commit (bibliography only)
	third := cl.Revisions[2]
	if third.Subject != "chore: add Lamport citation" {
		t.Errorf("expected third commit subject 'chore: add Lamport citation', got %q", third.Subject)
	}
	if len(third.Citations) != 1 {
		t.Fatalf("expected 1 citation change in third commit, got %d", len(third.Citations))
	}
	if third.Citations[0].Key != "lamport1978" || !strings.Contains(third.Citations[0].Formatted, "Leslie Lamport") {
		t.Errorf("unexpected citation change: %#v", third.Citations[0])
	}

	// Test max-count
	clLimited, err := ExtractChangelog(ms, Options{MaxCount: 1})
	if err != nil {
		t.Fatalf("ExtractChangelog with MaxCount failed: %v", err)
	}
	if clLimited.TotalCommits != 1 {
		t.Errorf("expected 1 revision with MaxCount=1, got %d", clLimited.TotalCommits)
	}

	// Test since filter
	clSince, err := ExtractChangelog(ms, Options{Since: "2026-01-11"})
	if err != nil {
		t.Fatalf("ExtractChangelog with Since failed: %v", err)
	}
	if clSince.TotalCommits != 1 {
		t.Errorf("expected 1 revision after 2026-01-11, got %d", clSince.TotalCommits)
	}
}
