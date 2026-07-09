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
	VerifyResults       []VerifyResult
	UncitedClaims       []UncitedResult
	CitationSuggestions []SuggestionResult
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

// CitationSuggestion is a single candidate paper judged to support an uncited claim.
type CitationSuggestion struct {
	CitationKey string
	Reasoning   string
	Passages    []string
}

// SuggestionResult holds the outcome of searching the literature for support of an uncited claim.
type SuggestionResult struct {
	Paragraph   string
	Assertion   string
	Suggestions []CitationSuggestion // empty if no candidate supported the claim well enough
	Reasoning   string               // explains a rejection, or notes on the picks
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
				pdfPath, hasPDF := resolvePDFPath(ms.Root, key, pdfMap)

				var chunks []string
				if hasPDF {
					chunks = getOrExtractChunks(ms, pe, key, pdfPath, bibStr)
				} else {
					logx.Warn("Literature PDF missing for citation @%s", key)
				}

				// Search top chunks using selected search method
				var passages []string
				if len(chunks) > 0 {
					if ms.Config.AI.SearchMethod == "embeddings" {
						matches, err := SearchEmbeddings(ctx, client, ms.Config.AI.EmbeddingModel, para.Text, chunks, 5, ms.Config.AI.SearchThreshold)
						if err != nil {
							logx.Error("Semantic search failed: %v. Falling back to BM25.", err)
							index := bm25.NewIndex(chunks)
							passages = index.Search(para.Text, 5)
						} else {
							for _, m := range matches {
								passages = append(passages, m.Text)
							}
						}
					} else {
						index := bm25.NewIndex(chunks)
						passages = index.Search(para.Text, 5)
					}
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

	// 5. Suggest supporting literature for uncited claims, searching across the
	// whole library since there is no citation key to scope the search to.
	if len(res.UncitedClaims) > 0 {
		var candidatesFinder func(query string) []llm.SuggestionCandidate

		if ms.Config.AI.SearchMethod == "embeddings" {
			chunks, chunkKeys := buildLiteratureChunks(ms, pe, bibMap, pdfMap)
			candidatesFinder = func(query string) []llm.SuggestionCandidate {
				return findSuggestionCandidatesEmbeddings(ctx, client, ms.Config.AI.EmbeddingModel, chunks, chunkKeys, bibMap, query, ms.Config.AI.SearchThreshold)
			}
		} else {
			litIndex, chunkKeys := buildLiteratureIndex(ms, pe, bibMap, pdfMap)
			candidatesFinder = func(query string) []llm.SuggestionCandidate {
				return findSuggestionCandidates(litIndex, chunkKeys, bibMap, query)
			}
		}

		for _, uc := range res.UncitedClaims {
			candidates := candidatesFinder(uc.Assertion)
			if len(candidates) == 0 {
				continue
			}

			prompt, err := llm.RenderSuggestPrompt(llm.SuggestPromptData{
				Assertion:  uc.Assertion,
				Paragraph:  uc.Paragraph,
				Candidates: candidates,
			})
			if err != nil {
				logx.Error("Failed to render suggest prompt: %v", err)
				continue
			}

			if !client.IsCached(prompt) {
				logx.Info("Searching literature for support of: %q...", uc.Assertion)
			}
			rawJSON, err := client.Call(ctx, prompt)
			if err != nil {
				logx.Error("LLM citation suggestion failed: %v", err)
				continue
			}
			sr, err := llm.ParseSuggestionResult(rawJSON)
			if err != nil {
				logx.Error("Failed to parse suggestion result: %v", err)
				continue
			}

			suggestions := make([]CitationSuggestion, len(sr.Suggestions))
			for i, s := range sr.Suggestions {
				suggestions[i] = CitationSuggestion{
					CitationKey: s.CitationKey,
					Reasoning:   s.Reasoning,
					Passages:    s.Passages,
				}
			}
			res.CitationSuggestions = append(res.CitationSuggestions, SuggestionResult{
				Paragraph:   uc.Paragraph,
				Assertion:   uc.Assertion,
				Suggestions: suggestions,
				Reasoning:   sr.Reasoning,
			})
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

// suggestCandidatePapers and suggestCandidatePassages bound how much literature
// context is fed into a single citation-suggestion LLM call.
const (
	suggestSearchPoolSize    = 20 // number of chunks pulled from the combined index before grouping
	suggestCandidatePapers   = 4  // distinct papers offered to the LLM per uncited claim
	suggestCandidatePassages = 3  // passages kept per candidate paper
)

// resolvePDFPath finds the literature PDF for a citation key, falling back to
// the conventional "literature/<key>.pdf" path if it isn't in pdfMap.
func resolvePDFPath(root, key string, pdfMap map[string]string) (string, bool) {
	if p, ok := pdfMap[key]; ok {
		return p, true
	}
	fallback := filepath.Join(root, "literature", key+".pdf")
	if _, err := os.Stat(fallback); err == nil {
		return fallback, true
	}
	return "", false
}

// getOrExtractChunks resolves the chunked text of a literature PDF, reusing the
// on-disk cache when the bib entry it was extracted alongside hasn't changed.
func getOrExtractChunks(ms *manuscript.Manuscript, pe *pdf.PDFExtractor, key, pdfPath, bibStr string) []string {
	hash, err := cache.GetFileHash(pdfPath)
	if err != nil {
		logx.Warn("Failed to hash %s: %v", filepath.Base(pdfPath), err)
		return nil
	}

	if lCache, err := cache.GetLitCache(ms.Root, hash); err == nil && lCache.BibtexEntry == bibStr {
		chunks := lCache.Chunks
		if len(chunks) == 0 && lCache.FullText != "" {
			chunks = SplitIntoChunks(lCache.FullText)
		}
		return chunks
	}

	logx.Info("Processing literature: %s...", filepath.Base(pdfPath))
	text, err := pe.ExtractText(pdfPath)
	if err != nil {
		logx.Warn("Failed to extract text from %s: %v", filepath.Base(pdfPath), err)
		return nil
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
	return chunks
}

// buildLiteratureChunks builds a flat list of literature chunks and their citation keys.
func buildLiteratureChunks(ms *manuscript.Manuscript, pe *pdf.PDFExtractor, bibMap map[string]bibtex.Entry, pdfMap map[string]string) ([]string, []string) {
	keys := make([]string, 0, len(pdfMap))
	for key := range pdfMap {
		keys = append(keys, key)
	}
	sort.Strings(keys) // deterministic ordering, since results feed an LLM cache key

	var chunks []string
	var chunkKeys []string
	for _, key := range keys {
		pdfPath, hasPDF := resolvePDFPath(ms.Root, key, pdfMap)
		if !hasPDF {
			continue
		}

		bibStr := ""
		if e, ok := bibMap[key]; ok {
			bibStr = FormatEntry(e)
		}

		for _, c := range getOrExtractChunks(ms, pe, key, pdfPath, bibStr) {
			chunks = append(chunks, c)
			chunkKeys = append(chunkKeys, key)
		}
	}

	return chunks, chunkKeys
}

// buildLiteratureIndex builds a single BM25 index across every resolvable
// literature PDF, not just cited ones, so uncited claims can be matched against
// the full library rather than a single already-known paper. It returns the
// index alongside a parallel slice mapping each indexed chunk back to the
// citation key of the paper it came from.
func buildLiteratureIndex(ms *manuscript.Manuscript, pe *pdf.PDFExtractor, bibMap map[string]bibtex.Entry, pdfMap map[string]string) (*bm25.Index, []string) {
	chunks, chunkKeys := buildLiteratureChunks(ms, pe, bibMap, pdfMap)
	return bm25.NewIndex(chunks), chunkKeys
}

// findSuggestionCandidates searches the combined literature index for passages
// relevant to a claim, groups the results by source paper, and returns up to
// suggestCandidatePapers candidates each carrying up to suggestCandidatePassages
// supporting passages, ranked by BM25 relevance.
func findSuggestionCandidates(index *bm25.Index, chunkKeys []string, bibMap map[string]bibtex.Entry, query string) []llm.SuggestionCandidate {
	results := index.SearchScored(query, suggestSearchPoolSize)

	var order []string
	selected := make(map[string]bool)
	passagesByKey := make(map[string][]string)
	for _, r := range results {
		key := chunkKeys[r.ID]
		if !selected[key] {
			if len(order) >= suggestCandidatePapers {
				continue
			}
			selected[key] = true
			order = append(order, key)
		}
		if len(passagesByKey[key]) < suggestCandidatePassages {
			passagesByKey[key] = append(passagesByKey[key], r.Text)
		}
	}

	candidates := make([]llm.SuggestionCandidate, 0, len(order))
	for _, key := range order {
		bibStr := ""
		if e, ok := bibMap[key]; ok {
			bibStr = FormatEntry(e)
		}
		candidates = append(candidates, llm.SuggestionCandidate{
			CitationKey: key,
			Bibtex:      bibStr,
			Passages:    passagesByKey[key],
		})
	}
	return candidates
}

// findSuggestionCandidatesEmbeddings searches the combined literature chunks using semantic embeddings
// for passages relevant to a claim, groups the results by source paper, and returns up to
// suggestCandidatePapers candidates each carrying up to suggestCandidatePassages
// supporting passages, ranked by cosine similarity and filtered by threshold.
func findSuggestionCandidatesEmbeddings(ctx context.Context, client llm.Client, model string, chunks []string, chunkKeys []string, bibMap map[string]bibtex.Entry, query string, threshold float64) []llm.SuggestionCandidate {
	matches, err := SearchEmbeddings(ctx, client, model, query, chunks, suggestSearchPoolSize, threshold)
	if err != nil {
		logx.Error("Semantic suggestion search failed: %v", err)
		return nil
	}

	var order []string
	selected := make(map[string]bool)
	passagesByKey := make(map[string][]string)
	for _, m := range matches {
		key := chunkKeys[m.ID]
		if !selected[key] {
			if len(order) >= suggestCandidatePapers {
				continue
			}
			selected[key] = true
			order = append(order, key)
		}
		if len(passagesByKey[key]) < suggestCandidatePassages {
			passagesByKey[key] = append(passagesByKey[key], m.Text)
		}
	}

	candidates := make([]llm.SuggestionCandidate, 0, len(order))
	for _, key := range order {
		bibStr := ""
		if e, ok := bibMap[key]; ok {
			bibStr = FormatEntry(e)
		}
		candidates = append(candidates, llm.SuggestionCandidate{
			CitationKey: key,
			Bibtex:      bibStr,
			Passages:    passagesByKey[key],
		})
	}
	return candidates
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
