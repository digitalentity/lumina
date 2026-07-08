package aicheck

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gtext "github.com/yuin/goldmark/text"

	"lumina/internal/aicheck/bm25"
	"lumina/internal/aicheck/cache"
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

// RunCrossCheck coordinates the full AI cross-checking pipeline.
func RunCrossCheck(ctx context.Context, ms *manuscript.Manuscript, force bool) (*CheckResult, error) {
	if force {
		logx.Info("Forcing fresh run. Clearing caches...")
		if err := cache.ClearCache(ms.Root); err != nil {
			logx.Warn("Failed to clear cache directory: %v", err)
		}
	}

	// 1. Initialize LLM Client
	client, err := llm.NewClient(ms.Config.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	// 2. Load LLM Cache
	llmCache, err := cache.LoadLLMCache(ms.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to load LLM cache: %w", err)
	}

	// 3. Load BibTeX entries
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

	// 4. Read manuscript paragraphs
	mdContent, err := os.ReadFile(ms.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to read manuscript.md: %w", err)
	}

	paras := ExtractManuscriptParagraphs(string(mdContent))
	pe := pdf.NewPDFExtractor(ms.Runner, ms.Root)
	res := &CheckResult{
		VerifyResults: []VerifyResult{},
		UncitedClaims: []UncitedResult{},
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
						// Check cache
						lCache, err := cache.GetLitCache(ms.Root, pdfHash)
						if err == nil && lCache.BibtexEntry == bibStr {
							chunks = lCache.Chunks
						} else {
							// Cache miss or stale metadata -> extract and chunk
							logx.Info("Processing cited literature: %s...", filepath.Base(pdfPath))
							text, err := pe.ExtractText(pdfPath)
							if err == nil {
								chunks = SplitIntoChunks(text)
								_ = cache.SaveLitCache(ms.Root, pdfHash, &cache.LitCacheEntry{
									BibtexKey:   key,
									BibtexEntry: bibStr,
									Chunks:      chunks,
								})
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

				// Check LLM Cache
				cacheKey := cache.ComputeLLMKey(para.Text, pdfHash, bibStr)
				if val, cached := llmCache[cacheKey]; cached {
					res.VerifyResults = append(res.VerifyResults, VerifyResult{
						Paragraph:   para.Text,
						CitationKey: key,
						Status:      val.Status,
						Reasoning:   val.Reasoning,
						Passages:    val.Passages,
					})
				} else {
					// Query LLM
					logx.Info("Verifying claim for @%s...", key)
					vr, err := client.VerifyClaim(ctx, para.Text, key, passages, bibStr)
					if err != nil {
						logx.Error("LLM claim verification failed for @%s: %v", key, err)
						continue
					}

					// Save cache
					llmCache[cacheKey] = cache.LLMCacheEntry{
						Status:    vr.Status,
						Reasoning: vr.Reasoning,
						Passages:  vr.Passages,
					}
					_ = cache.SaveLLMCache(ms.Root, llmCache)

					res.VerifyResults = append(res.VerifyResults, VerifyResult{
						Paragraph:   para.Text,
						CitationKey: key,
						Status:      vr.Status,
						Reasoning:   vr.Reasoning,
						Passages:    vr.Passages,
					})
				}
			}
		} else {
			// Uncited claim detection mode
			cacheKey := cache.ComputeLLMKey(para.Text, "uncited", "")
			if val, cached := llmCache[cacheKey]; cached {
				if val.Status == "uncited-claims" {
					for _, passage := range val.Passages {
						res.UncitedClaims = append(res.UncitedClaims, UncitedResult{
							Paragraph: para.Text,
							Assertion: passage,
							Reasoning: val.Reasoning,
						})
					}
				}
			} else {
				logx.Info("Analyzing paragraph for uncited claims...")
				claims, err := client.DetectUncitedClaims(ctx, para.Text)
				if err != nil {
					logx.Error("LLM uncited claim detection failed: %v", err)
					continue
				}

				// Save cache
				var passages []string
				reasoning := ""
				status := "no-uncited-claims"
				if len(claims) > 0 {
					status = "uncited-claims"
					reasoning = claims[0].Reasoning // store first or joined
					for _, c := range claims {
						passages = append(passages, c.Assertion)
						res.UncitedClaims = append(res.UncitedClaims, UncitedResult{
							Paragraph: para.Text,
							Assertion: c.Assertion,
							Reasoning: c.Reasoning,
						})
					}
				}

				llmCache[cacheKey] = cache.LLMCacheEntry{
					Status:    status,
					Reasoning: reasoning,
					Passages:  passages,
				}
				_ = cache.SaveLLMCache(ms.Root, llmCache)
			}
		}
	}

	return res, nil
}

// extractBlockText compiles raw text from Goldmark AST lines.
func extractBlockText(n gast.Node, source []byte) string {
	var sb strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		sb.Write(line.Value(source))
	}
	return strings.TrimSpace(sb.String())
}

// ExtractManuscriptParagraphs parses the Markdown content and extracts paragraphs/list items.
func ExtractManuscriptParagraphs(mdContent string) []ManuscriptParagraph {
	source := []byte(mdContent)
	doc := goldmark.DefaultParser().Parse(gtext.NewReader(source))
	var paragraphs []ManuscriptParagraph

	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		if n.Kind() == gast.KindParagraph || n.Kind() == gast.KindTextBlock {
			text := extractBlockText(n, source)
			if text != "" {
				paragraphs = append(paragraphs, ManuscriptParagraph{
					Text:      text,
					Citations: citations.ExtractCitationsFromText(text),
				})
			}
			return gast.WalkSkipChildren, nil
		}

		return gast.WalkContinue, nil
	})

	return paragraphs
}

// SplitIntoChunks splits text by double-newlines into paragraph-sized chunks.
func SplitIntoChunks(text string) []string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	parts := strings.Split(normalized, "\n\n")
	var chunks []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			cleaned := strings.Join(strings.Fields(trimmed), " ")
			chunks = append(chunks, cleaned)
		}
	}
	return chunks
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
