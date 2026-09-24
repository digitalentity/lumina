// Package changelog provides Git history extraction for manuscript files.
package changelog

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"lumina/internal/manuscript"
)

// ExtractChangelog extracts the revision history of the target manuscript.
func ExtractChangelog(ms *manuscript.Manuscript, opts Options) (*Changelog, error) {
	repo, err := git.PlainOpenWithOptions(ms.Root, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, fmt.Errorf("not a git repository: %s (run 'git init' first)", ms.Root)
		}
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to access git worktree: %w", err)
	}
	gitRoot := wt.Filesystem.Root()

	relPath, err := filepath.Rel(gitRoot, ms.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to compute relative manuscript path: %w", err)
	}
	relPath = filepath.ToSlash(relPath)

	relBibPath := ""
	if ms.BibPath != "" {
		if rb, err := filepath.Rel(gitRoot, ms.BibPath); err == nil {
			relBibPath = filepath.ToSlash(rb)
		}
	}

	headRef, err := repo.Head()
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, fmt.Errorf("git repository has no commits yet")
		}
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	cIter, err := repo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read commit log: %w", err)
	}

	sinceTime := parseSinceFilter(opts.Since)

	var revisions []Revision
	totalAdded := 0
	totalDeleted := 0

	err = cIter.ForEach(func(c *object.Commit) error {
		if !sinceTime.IsZero() && c.Author.When.Before(sinceTime) {
			return nil
		}

		currHash, currContent, currExists := getFileSnapshot(repo, c, relPath)
		var currBibHash plumbing.Hash
		var currBibContent string
		var currBibExists bool
		if relBibPath != "" {
			currBibHash, currBibContent, currBibExists = getFileSnapshot(repo, c, relBibPath)
		}

		// Check parent(s)
		var parentHash plumbing.Hash
		var parentContent string
		var parentExists bool

		var parentBibHash plumbing.Hash
		var parentBibContent string
		var parentBibExists bool

		if c.NumParents() > 0 {
			parent, err := c.Parent(0)
			if err == nil {
				parentHash, parentContent, parentExists = getFileSnapshot(repo, parent, relPath)
				if relBibPath != "" {
					parentBibHash, parentBibContent, parentBibExists = getFileSnapshot(repo, parent, relBibPath)
				}
			}
		}

		manuscriptChanged := (currHash != parentHash && (currExists || parentExists))
		bibChanged := (relBibPath != "" && currBibHash != parentBibHash && (currBibExists || parentBibExists))

		// Skip commits that altered neither manuscript nor bibliography
		if !manuscriptChanged && !bibChanged {
			return nil
		}

		var diffs []ParagraphDiff
		wordsAdded := 0
		wordsDeleted := 0

		if manuscriptChanged {
			diffs = DiffParagraphs(parentContent, currContent)
			for _, d := range diffs {
				switch d.Type {
				case DiffAdded:
					wordsAdded += CountWords(d.CurrentText)
				case DiffDeleted:
					wordsDeleted += CountWords(d.OriginalText)
				case DiffModified:
					for _, s := range d.Spans {
						switch s.Type {
						case DiffAdded:
							wordsAdded += CountWords(s.Text)
						case DiffDeleted:
							wordsDeleted += CountWords(s.Text)
						}
					}
				}
			}
		}

		var citations []CitationChange
		if bibChanged {
			citations = DiffBib(parentBibContent, currBibContent)
		}

		totalWords := CountWords(currContent)
		shortHash := c.Hash.String()
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}

		subject := strings.TrimSpace(c.Message)
		body := ""
		if idx := strings.Index(subject, "\n"); idx != -1 {
			body = strings.TrimSpace(subject[idx+1:])
			subject = strings.TrimSpace(subject[:idx])
		}

		rev := Revision{
			Hash:         c.Hash.String(),
			ShortHash:    shortHash,
			Author:       c.Author.Name,
			Email:        c.Author.Email,
			Date:         c.Author.When,
			Subject:      subject,
			Body:         body,
			WordsAdded:   wordsAdded,
			WordsDeleted: wordsDeleted,
			NetWords:     wordsAdded - wordsDeleted,
			TotalWords:   totalWords,
			Diffs:        diffs,
			Citations:    citations,
		}

		revisions = append(revisions, rev)
		totalAdded += wordsAdded
		totalDeleted += wordsDeleted

		if opts.MaxCount > 0 && len(revisions) >= opts.MaxCount {
			return io.EOF // Stop iteration early
		}

		return nil
	})

	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("error reading commits: %w", err)
	}

	title := ms.Stem
	if rawTitle, ok := ms.RawMeta["title"].(string); ok && strings.TrimSpace(rawTitle) != "" {
		title = strings.TrimSpace(rawTitle)
	}

	// Order chronologically (oldest revision first, newest revision last)
	slices.Reverse(revisions)

	return &Changelog{
		Target:            ms.Target,
		Title:             title,
		Stem:              ms.Stem,
		ManuscriptPath:    relPath,
		BibPath:           relBibPath,
		Revisions:         revisions,
		TotalCommits:      len(revisions),
		TotalWordsAdded:   totalAdded,
		TotalWordsDeleted: totalDeleted,
	}, nil
}

func getFileSnapshot(repo *git.Repository, commit *object.Commit, relPath string) (plumbing.Hash, string, bool) {
	tree, err := commit.Tree()
	if err != nil {
		return plumbing.ZeroHash, "", false
	}

	entry, err := tree.FindEntry(relPath)
	if err != nil {
		return plumbing.ZeroHash, "", false
	}

	blob, err := repo.BlobObject(entry.Hash)
	if err != nil {
		return entry.Hash, "", true
	}

	reader, err := blob.Reader()
	if err != nil {
		return entry.Hash, "", true
	}
	defer reader.Close()

	contentBytes, err := io.ReadAll(reader)
	if err != nil {
		return entry.Hash, "", true
	}

	return entry.Hash, string(contentBytes), true
}

func parseSinceFilter(since string) time.Time {
	if since == "" {
		return time.Time{}
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
	}
	for _, fmtStr := range formats {
		if t, err := time.Parse(fmtStr, since); err == nil {
			return t
		}
	}

	return time.Time{}
}
