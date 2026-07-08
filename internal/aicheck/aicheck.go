package aicheck

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"lumina/internal/aicheck/bm25"
	"lumina/internal/aicheck/cache"
	"lumina/internal/aicheck/chunk"
	"lumina/internal/aicheck/llm"
	"lumina/internal/aicheck/pdf"
	"lumina/internal/bibtex"
	"lumina/internal/citations"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

// ManuscriptParagraph represents a prose paragraph extracted from the manuscript.
type ManuscriptParagraph struct {
	Text      string   `json:"text"`
	Citations []string `json:"citations"`
}

// CheckResult holds all findings from the AI cross-check run.
type CheckResult struct {
	VerifyResults []VerifyResult
	UncitedClaims []UncitedResult
}

type VerifyResult struct {
	Paragraph   string
	CitationKey string
	Status      string   // "supported" | "contradicted" | "unsupported" | "neutral"
	Reasoning   string   // Explains why
	Passages    []string // direct supporting quotes
}

type UncitedResult struct {
	Paragraph string
	Assertion string
	Reasoning string
}

// FormatEntry formats a parsed bibtex.Entry back into standard BibTeX string.
func FormatEntry(e bibtex.Entry) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("@%s{%s,\n", e.Type, e.Key))
	var keys []string
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("  %s = {%s},\n", k, e.Fields[k]))
	}
	sb.WriteString("}")
	return sb.String()
}

// IndexLiterature scans the literature directory, extracts text from each PDF,
// and populates the literature cache. It is called by RunCrossCheck and by the
// dedicated "lumina ai index" subcommand.
//
// When force is true the existing cache is wiped before indexing.
func IndexLiterature(ctx context.Context, ms *manuscript.Manuscript, force bool) error {
	if force {
		logx.Info("Forcing fresh index. Clearing lit cache...")
		if err := cache.ClearCache(ms.Root); err != nil {
			logx.Warn("Failed to clear cache directory: %v", err)
		}
	}

	bibPath := filepath.Join(ms.Root, "references.bib")
	entries, err := bibtex.Parse(bibPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read references.bib: %w", err)
	}

	bibMap := make(map[string]bibtex.Entry)
	for _, entry := range entries {
		bibMap[entry.Key] = entry
	}

	pdfMap, err := buildPDFMap(ms.Root, bibMap)
	if err != nil {
		return fmt.Errorf("failed to scan literature directory: %w", err)
	}

	pe := pdf.NewPDFExtractor(ms.Runner, ms.Root)
	for key, pdfPath := range pdfMap {
		bibStr := ""
		if e, ok := bibMap[key]; ok {
			bibStr = FormatEntry(e)
		}

		hash, err := cache.GetFileHash(pdfPath)
		if err != nil {
			logx.Warn("Failed to hash %s: %v", filepath.Base(pdfPath), err)
			continue
		}

		// Skip if cache is fresh.
		if lCache, err := cache.GetLitCache(ms.Root, hash); err == nil && lCache.BibtexEntry == bibStr {
			logx.Info("Cache hit for %s, skipping.", filepath.Base(pdfPath))
			continue
		}

		logx.Info("Indexing %s...", filepath.Base(pdfPath))
		text, err := pe.ExtractText(pdfPath)
		if err != nil {
			logx.Warn("Failed to extract text from %s: %v", filepath.Base(pdfPath), err)
			continue
		}

		chunks := SplitIntoChunks(text)
		if err := cache.SaveLitCache(ms.Root, hash, &cache.LitCacheEntry{
			BibtexKey:   key,
			BibtexEntry: bibStr,
			FullText:    text,
			Chunks:      chunks,
		}); err != nil {
			logx.Warn("Failed to save cache for %s: %v", filepath.Base(pdfPath), err)
		}
	}

	return nil
}

// RunCrossCheck coordinates the full AI cross-checking pipeline.
func RunCrossCheck(ctx context.Context, ms *manuscript.Manuscript, force bool) (*CheckResult, error) {
	if force {
		logx.Info("Forcing fresh run. Clearing caches...")
		if err := cache.ClearCache(ms.Root); err != nil {
			logx.Warn("Failed to clear cache directory: %v", err)
		}
	}

	// 1. Initialize LLM Client with transparent caching
	baseClient, err := llm.NewClient(ms.Config.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	client, err := llm.NewCachingClient(baseClient, ms.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM cache: %w", err)
	}

	// 2. Load BibTeX entries
	bibPath := filepath.Join(ms.Root, "references.bib")
	entries, err := bibtex.Parse(bibPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read references.bib: %w", err)
	}

	bibMap := make(map[string]bibtex.Entry)
	for _, entry := range entries {
		bibMap[entry.Key] = entry
	}

	pdfMap, err := buildPDFMap(ms.Root, bibMap)
	if err != nil {
		return nil, fmt.Errorf("failed to scan literature directory: %w", err)
	}

	// 3. Read manuscript paragraphs
	mdContent, err := os.ReadFile(ms.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to read manuscript.md: %w", err)
	}

	paras := ExtractManuscriptParagraphs(string(mdContent))
	pe := pdf.NewPDFExtractor(ms.Runner, ms.Root)
	res := &CheckResult{
		VerifyResults: make([]VerifyResult, 0),
		UncitedClaims: make([]UncitedResult, 0),
	}

	for _, para := range paras {
		if len(para.Citations) > 0 {
			// Verification mode
			for _, key := range para.Citations {
				bibEntry, hasBib := bibMap[key]
				bibStr := ""
				if hasBib {
					bibStr = FormatEntry(bibEntry)
				}

				// Find PDF file
				pdfPath, hasPDF := pdfMap[key]
				if !hasPDF {
					fallbackPath := filepath.Join(ms.Root, "literature", key+".pdf")
					if _, err := os.Stat(fallbackPath); err == nil {
						pdfPath = fallbackPath
						hasPDF = true
					}
				}

				var chunks []string
				pdfHash := "missing-pdf"

				if hasPDF {
					hash, err := cache.GetFileHash(pdfPath)
					if err == nil {
						pdfHash = hash
						// Check cache; fall back to on-demand extraction if stale.
						lCache, err := cache.GetLitCache(ms.Root, pdfHash)
						if err == nil && lCache.BibtexEntry == bibStr {
							chunks = lCache.Chunks
							if len(chunks) == 0 && lCache.FullText != "" {
								chunks = SplitIntoChunks(lCache.FullText)
							}
						} else {
							logx.Info("Processing cited literature: %s...", filepath.Base(pdfPath))
							text, err := pe.ExtractText(pdfPath)
							if err == nil {
								chunks = SplitIntoChunks(text)
								_ = cache.SaveLitCache(
									ms.Root,
									pdfHash,
									&cache.LitCacheEntry{
										BibtexKey:   key,
										BibtexEntry: bibStr,
										FullText:    text,
										Chunks:      chunks,
									},
								)
							} else {
								logx.Warn("Failed to extract text from %s: %v", filepath.Base(pdfPath), err)
							}
						}
					}
				} else {
					logx.Warn("Literature PDF missing for citation @%s", key)
				}

				// Search top chunks using BM25
				var passages []string
				if len(chunks) > 0 {
					index := bm25.NewIndex(chunks)
					passages = index.Search(para.Text, 5)
				}

				// Render prompt once; key on (prompt, model) so template or model changes bust the cache.
				prompt, err := llm.RenderVerifyPrompt(llm.VerifyPromptData{
					Paragraph:   para.Text,
					CitationKey: key,
					Bibtex:      bibStr,
					Passages:    passages,
				})
				if err != nil {
					logx.Error("Failed to render verify prompt for @%s: %v", key, err)
					continue
				}

				if !client.IsCached(prompt) {
					logx.Info("Verifying claim for @%s...", key)
				}
				rawJSON, err := client.Call(ctx, prompt)
				if err != nil {
					logx.Error("LLM claim verification failed for @%s: %v", key, err)
					continue
				}
				vr, err := llm.ParseVerificationResult(rawJSON)
				if err != nil {
					logx.Error("Failed to parse verification result for @%s: %v", key, err)
					continue
				}

				res.VerifyResults = append(res.VerifyResults, VerifyResult{
					Paragraph:   para.Text,
					CitationKey: key,
					Status:      vr.Status,
					Reasoning:   vr.Reasoning,
					Passages:    vr.Passages,
				})
			}
		}
	}

	// 4. Uncited claim detection mode (manuscript-wide)
	prompt, err := llm.RenderUncitedPrompt(llm.UncitedPromptData{
		Manuscript: string(mdContent),
	})
	if err != nil {
		logx.Error("Failed to render uncited prompt: %v", err)
	} else {
		if !client.IsCached(prompt) {
			logx.Info("Analyzing manuscript for uncited claims...")
		}
		rawJSON, err := client.Call(ctx, prompt)
		if err != nil {
			logx.Error("LLM uncited claim detection failed: %v", err)
		} else {
			claims, err := llm.ParseUncitedClaims(rawJSON)
			if err != nil {
				logx.Error("Failed to parse uncited claims: %v", err)
			} else {
				for _, c := range claims {
					res.UncitedClaims = append(res.UncitedClaims, UncitedResult{
						Paragraph: c.Paragraph,
						Assertion: c.Assertion,
						Reasoning: c.Reasoning,
					})
				}
			}
		}
	}

	return res, nil
}

// ExtractManuscriptParagraphs parses the Markdown content and extracts paragraphs/list items.
func ExtractManuscriptParagraphs(mdContent string) []ManuscriptParagraph {
	chunks := chunk.Split(mdContent, 5)
	paragraphs := make([]ManuscriptParagraph, len(chunks))
	for i, c := range chunks {
		paragraphs[i] = ManuscriptParagraph{
			Text:      c,
			Citations: citations.ExtractCitationsFromText(c),
		}
	}
	return paragraphs
}

// SplitIntoChunks splits text into paragraph-sized chunks for BM25 indexing.
func SplitIntoChunks(text string) []string {
	return chunk.Split(text, 10)
}

// buildPDFMap scans the literature directory to match citation keys to PDFs and updates the bibMap.
func buildPDFMap(root string, bibMap map[string]bibtex.Entry) (map[string]string, error) {
	pdfMap := make(map[string]string)
	litDir := filepath.Join(root, "literature")
	files, err := os.ReadDir(litDir)
	if err != nil {
		if os.IsNotExist(err) {
			return pdfMap, nil
		}
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".bib") {
			bibFilePath := filepath.Join(litDir, file.Name())
			if subEntries, err := bibtex.Parse(bibFilePath); err == nil {
				pdfFilePath := strings.TrimSuffix(bibFilePath, ".bib") + ".pdf"
				if _, err := os.Stat(pdfFilePath); err == nil {
					for _, entry := range subEntries {
						pdfMap[entry.Key] = pdfFilePath
						if _, exists := bibMap[entry.Key]; !exists {
							bibMap[entry.Key] = entry
						}
					}
				}
			}
		}
	}
	return pdfMap, nil
}
