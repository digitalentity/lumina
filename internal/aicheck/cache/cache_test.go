package cache

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileHash(t *testing.T) {
	tmp, err := os.CreateTemp("", "hash-test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	_, _ = tmp.WriteString("Lumina Academic Writing CLI")
	tmp.Close()

	hash, err := GetFileHash(tmp.Name())
	if err != nil {
		t.Fatalf("GetFileHash error: %v", err)
	}

	expected := "65888874ff0e1aab49b638ec5764540082e46d1511a21327d60838eef9949944"
	if hash != expected {
		t.Errorf("expected hash %q, got %q", expected, hash)
	}
}

func TestLitCacheSaveLoad(t *testing.T) {
	root, err := os.MkdirTemp("", "lumina-cache-root")
	if err != nil {
		t.Fatalf("failed to create temp root: %v", err)
	}
	defer os.RemoveAll(root)

	pdfHash := "abcdef1234567890"
	entry := &LitCacheEntry{
		BibtexKey:   "smith2024",
		BibtexEntry: "@article{smith2024, title={Gravity}}",
		Chunks: []string{
			"gravity drive distortions",
			"warp propulsion inside space",
		},
	}

	// 1. Get from empty cache -> expect error
	_, err = GetLitCache(root, pdfHash)
	if err == nil {
		t.Fatal("expected error reading from empty cache, got nil")
	}

	// 2. Save entry
	if err := SaveLitCache(root, pdfHash, entry); err != nil {
		t.Fatalf("SaveLitCache error: %v", err)
	}

	// 3. Load entry and verify
	loaded, err := GetLitCache(root, pdfHash)
	if err != nil {
		t.Fatalf("GetLitCache error: %v", err)
	}

	if !reflect.DeepEqual(loaded, entry) {
		t.Errorf("loaded entry did not match saved. got %+v, want %+v", loaded, entry)
	}

	// 4. Clear cache
	if err := ClearCache(root); err != nil {
		t.Fatalf("ClearCache error: %v", err)
	}

	// 5. Verify gone
	_, err = GetLitCache(root, pdfHash)
	if err == nil {
		t.Fatal("expected error reading cleared cache, got nil")
	}
	
	// Ensure folder is cleaned up
	_, err = os.Stat(filepath.Join(root, ".lumina", "literature_cache"))
	if !os.IsNotExist(err) {
		t.Errorf("expected literature_cache directory to be deleted")
	}
}

func TestLLMCache(t *testing.T) {
	root, err := os.MkdirTemp("", "lumina-llm-cache-root")
	if err != nil {
		t.Fatalf("failed to create temp root: %v", err)
	}
	defer os.RemoveAll(root)

	para := "Manuscript claim context."
	model := "gemini-2.5-flash"

	key := ComputeLLMKey(para, model)
	if key == "" {
		t.Fatal("expected computed key to be non-empty")
	}

	// 1. Load empty cache
	cache, err := LoadLLMCache(root)
	if err != nil {
		t.Fatalf("LoadLLMCache error: %v", err)
	}
	if len(cache) != 0 {
		t.Errorf("expected empty cache, got %d entries", len(cache))
	}

	// 2. Add entry and save
	entry := LLMCacheEntry{
		Response: `{"status": "supported", "reasoning": "It is supported.", "passages": ["supporting passage"]}`,
	}
	cache[key] = entry

	if err := SaveLLMCache(root, cache); err != nil {
		t.Fatalf("SaveLLMCache error: %v", err)
	}

	// 3. Load and verify
	loadedCache, err := LoadLLMCache(root)
	if err != nil {
		t.Fatalf("LoadLLMCache error: %v", err)
	}

	loadedEntry, exists := loadedCache[key]
	if !exists {
		t.Fatalf("expected key to exist in loaded cache")
	}
	if !reflect.DeepEqual(loadedEntry, entry) {
		t.Errorf("loaded entry did not match saved. got %+v, want %+v", loadedEntry, entry)
	}
}
