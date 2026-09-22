// Command fetchcorpus downloads a modern academic-prose training corpus for
// train-ngram: paper titles and abstracts from the arXiv API, restricted to
// papers submitted on or before 2023-12-31 to avoid corpus contamination by
// widespread LLM-assisted writing (ChatGPT launched November 2022). Run by
// hand, from tools/train-ngram/, whenever the corpus needs (re)fetching:
//
//	go run ./fetchcorpus -out corpus
package main

import (
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
	apiBase = "https://export.arxiv.org/api/query"

	pageSize    = 200
	pagesPerCat = 3 // 3 * 200 = up to 600 abstracts per category

	// requestPause honors arXiv's API etiquette guidance of no more than
	// one request every 3 seconds.
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
	Title   string `xml:"title"`
	Summary string `xml:"summary"`
}

func main() {
	outDir := flag.String("out", "corpus", "directory to write per-category corpus files")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("creating output dir: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	for _, cat := range categories {
		if err := fetchCategory(client, cat, *outDir); err != nil {
			log.Fatalf("fetching %s: %v", cat, err)
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
		}
		time.Sleep(requestPause)
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	log.Printf("wrote %s (%d abstracts)", path, count)
	return nil
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
