// Command fetchcorpus downloads a modern academic-prose training corpus for
// train-ngram: paper titles and abstracts from the arXiv API, restricted to
// papers submitted on or before 2023-12-31 to avoid corpus contamination by
// widespread LLM-assisted writing (ChatGPT launched November 2022). Run by
// hand, from tools/train-ngram/, whenever the corpus needs (re)fetching:
//
//	go run ./fetchcorpus -out corpus
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// categories span distinct academic registers/vocabularies for a broader
// modern-prose sample than any single field would give.
var categories = []string{
	"cs.CL", "cs.LG", "cs.AI",
	"math.CO", "math.ST",
	"physics.gen-ph", "astro-ph.GA",
	"q-bio.GN", "q-bio.NC",
	"stat.ME",
	"econ.GN",
}

const (
	apiBase    = "https://export.arxiv.org/api/query"
	eprintBase = "https://export.arxiv.org/e-print/"

	pageSize    = 200
	pagesPerCat = 3 // 3 * 200 = up to 600 abstracts per category

	// fullTextPerCategory bounds how many papers per category get their
	// full LaTeX source fetched and stripped to prose (in addition to,
	// not instead of, the abstract). Abstracts are cheap and plentiful;
	// full papers are one e-print download each, so this stays far below
	// pagesPerCat*pageSize to keep a full corpus fetch to tens of minutes.
	fullTextPerCategory = 40

	// maxDecompressedBytes is a zip-bomb sanity backstop on the whole
	// decompressed archive, well above any real paper's source tree
	// (e-prints commonly bundle tens of MB of embedded figures).
	maxDecompressedBytes = 200 << 20 // 200 MiB

	// maxTexFileBytes caps a single .tex file's content; tar.Reader
	// already delimits each entry by its header size, so this only guards
	// against a pathological single oversized file.
	maxTexFileBytes = 4 << 20 // 4 MiB

	// requestPause honors arXiv's API etiquette guidance of no more than
	// one request every 3 seconds. Applied between both metadata-API
	// pages and individual e-print downloads.
	requestPause = 3 * time.Second

	// cutoffDateFrom/cutoffDateTo bound the query to arXiv's founding
	// through the end of 2023: papers submitted on or before this date
	// predate widespread LLM-assisted academic writing.
	cutoffDateFrom = "19910101"
	cutoffDateTo   = "20231231"
)

type feed struct {
	Entries []entry `xml:"entry"`
}

type entry struct {
	ID      string `xml:"id"`
	Title   string `xml:"title"`
	Summary string `xml:"summary"`
}

// arxivID extracts the bare id (e.g. "2302.00129") from an Atom entry's
// <id> URL (e.g. "http://arxiv.org/abs/2302.00129v2"), dropping any
// version suffix.
func arxivID(entryID string) string {
	id := entryID[strings.LastIndex(entryID, "/")+1:]
	if v := strings.LastIndex(id, "v"); v > 0 {
		if _, err := fmt.Sscanf(id[v+1:], "%d", new(int)); err == nil {
			id = id[:v]
		}
	}
	return id
}

func main() {
	outDir := flag.String("out", "corpus", "directory to write per-category corpus files")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("creating output dir: %v", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	for _, cat := range categories {
		// A transient network hiccup on one category (arXiv API timeout,
		// a dropped connection) must not lose the categories already
		// written to disk. fetchCategory skips any category whose output
		// file already exists, so simply re-running this command resumes
		// from the first category that never finished.
		if err := fetchCategory(client, cat, *outDir); err != nil {
			log.Printf("fetching %s: %v (re-run this command to retry)", cat, err)
		}
	}
}

func fetchCategory(client *http.Client, category, outDir string) error {
	path := filepath.Join(outDir, fmt.Sprintf("arxiv-%s.txt", sanitizeCategory(category)))
	if _, err := os.Stat(path); err == nil {
		log.Printf("skip %s (already present)", path)
		return nil
	}

	var sb strings.Builder
	count := 0
	fullTextFetched := 0
	for page := range pagesPerCat {
		entries, err := fetchPage(client, category, page*pageSize)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		if len(entries) == 0 {
			break
		}
		for _, e := range entries {
			sb.WriteString(clean(e.Title))
			sb.WriteString("\n\n")
			sb.WriteString(clean(e.Summary))
			sb.WriteString("\n\n")
			count++

			if fullTextFetched >= fullTextPerCategory || e.ID == "" {
				continue
			}
			time.Sleep(requestPause)
			prose, err := fetchFullText(client, arxivID(e.ID))
			fullTextFetched++
			if err != nil {
				log.Printf("skip full text for %s: %v", e.ID, err)
				continue
			}
			if prose != "" {
				sb.WriteString(prose)
				sb.WriteString("\n\n")
			}
		}
		time.Sleep(requestPause)
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	log.Printf("wrote %s (%d abstracts, %d full-text papers)", path, count, fullTextFetched)
	return nil
}

// fetchFullText downloads a paper's e-print source, extracts its .tex
// file(s), and strips them to plain prose. Returns "", nil (not an error)
// when the paper has no LaTeX source (PDF-only submissions, scanned older
// papers) — those are simply skipped, not fatal to the corpus fetch.
func fetchFullText(client *http.Client, id string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, eprintBase+id, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "lumina-train-ngram-corpus-fetcher/1.0 (offline research tool)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDecompressedBytes))
	if err != nil {
		return "", err
	}

	texFiles, err := extractTexFiles(body)
	if err != nil {
		return "", nil //nolint:nilerr // no usable LaTeX source; not fatal
	}

	var out strings.Builder
	for _, tex := range texFiles {
		out.WriteString(StripLatex(tex))
		out.WriteString("\n\n")
	}
	return strings.TrimSpace(out.String()), nil
}

// extractTexFiles gzip-decompresses an e-print body and returns the
// contents of every .tex file inside. Handles both the common case (a tar
// archive of the paper's source tree, commonly tens of MB once figures are
// included) and the single-file case (some older submissions are a bare
// gzipped .tex file, not a tar).
func extractTexFiles(body []byte) ([]string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("not gzip (likely a PDF-only submission): %w", err)
	}
	defer gz.Close()

	decompressed, err := io.ReadAll(io.LimitReader(gz, maxDecompressedBytes))
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(bytes.NewReader(decompressed))
	var texFiles []string
	// firstEntry distinguishes "this was never a tar archive" (fall back
	// to treating the whole payload as one bare .tex file, the older
	// single-source-file format) from "this is a tar archive that we
	// stopped reading partway through" (e.g. a later entry got skipped or
	// something odd happened) — the latter must never fall back to
	// dumping the entire archive, tex and binary figures alike, as one
	// giant "prose" blob.
	firstEntry := true
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if firstEntry {
				return []string{string(decompressed)}, nil
			}
			break
		}
		firstEntry = false
		if hdr.Typeflag != tar.TypeReg || !strings.HasSuffix(hdr.Name, ".tex") {
			continue
		}
		content, err := io.ReadAll(io.LimitReader(tr, maxTexFileBytes))
		if err != nil {
			return nil, err
		}
		texFiles = append(texFiles, string(content))
	}
	if len(texFiles) == 0 {
		return nil, fmt.Errorf("no .tex files found in archive")
	}
	return texFiles, nil
}

func fetchPage(client *http.Client, category string, start int) ([]entry, error) {
	q := url.Values{}
	q.Set("search_query", fmt.Sprintf("cat:%s AND submittedDate:[%s0000 TO %s2359]", category, cutoffDateFrom, cutoffDateTo))
	q.Set("start", fmt.Sprintf("%d", start))
	q.Set("max_results", fmt.Sprintf("%d", pageSize))
	q.Set("sortBy", "submittedDate")
	q.Set("sortOrder", "descending")

	req, err := http.NewRequest(http.MethodGet, apiBase+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "lumina-train-ngram-corpus-fetcher/1.0 (offline research tool)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("parsing atom feed: %w", err)
	}
	return f.Entries, nil
}

// clean collapses whitespace, including the newlines arXiv wraps abstract
// text with, into single spaces.
func clean(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func sanitizeCategory(cat string) string {
	return strings.ReplaceAll(cat, ".", "-")
}
