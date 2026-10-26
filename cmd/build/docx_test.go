package build

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lumina/internal/manuscript"
)

const testDocument = `<w:document xmlns:w="w"><w:body><w:p/><w:sectPr/></w:body></w:document>`

// fakeRunner records pandoc calls and writes a minimal reference document
// for --print-default-data-file.
type fakeRunner struct {
	calls [][]string
}

func (f *fakeRunner) Run(tool string, args []string, cwd string) error {
	f.calls = append(f.calls, append([]string{tool}, args...))
	return writeTestDocx(filepath.Join(cwd, args[1]))
}

func (f *fakeRunner) Capture(string, []string, string) ([]byte, error) { return nil, nil }
func (f *fakeRunner) CheckPresent(string) error                        { return nil }

func writeTestDocx(path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, testDocument); err != nil {
		return err
	}
	return zw.Close()
}

func readDocument(t *testing.T, path string) string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	t.Fatalf("%s has no word/document.xml", path)
	return ""
}

func newDocxTestManuscript(t *testing.T, meta map[string]any) (*manuscript.Manuscript, *fakeRunner) {
	t.Helper()
	root := t.TempDir()
	fr := &fakeRunner{}
	ms := &manuscript.Manuscript{
		Root:      root,
		LuminaDir: filepath.Join(root, ".lumina"),
		RawMeta:   meta,
		Runner:    fr,
	}
	if err := os.MkdirAll(ms.LuminaBuildDir(), 0755); err != nil {
		t.Fatal(err)
	}
	return ms, fr
}

func TestPageSetupReferenceNoMetadata(t *testing.T) {
	ms, fr := newDocxTestManuscript(t, map[string]any{"title": "x"})
	got, err := pageSetupReference(ms, "/templates/default/reference.docx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/templates/default/reference.docx" {
		t.Errorf("expected the template reference unchanged, got %q", got)
	}
	if len(fr.calls) != 0 {
		t.Errorf("expected no pandoc calls, got %v", fr.calls)
	}
}

func TestPageSetupReferenceTemplate(t *testing.T) {
	ms, fr := newDocxTestManuscript(t, map[string]any{"papersize": "letter", "geometry": "margin=2cm"})
	src := filepath.Join(ms.Root, "reference.docx")
	if err := writeTestDocx(src); err != nil {
		t.Fatal(err)
	}

	got, err := pageSetupReference(ms, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != filepath.Join(ms.LuminaBuildDir(), "reference.docx") {
		t.Errorf("expected the staged copy, got %q", got)
	}
	if len(fr.calls) != 0 {
		t.Errorf("expected no pandoc calls, got %v", fr.calls)
	}
	doc := readDocument(t, got)
	for _, want := range []string{`<w:pgSz w:w="12240" w:h="15840"/>`, `w:left="1134"`} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in %s", want, doc)
		}
	}
	if readDocument(t, src) != testDocument {
		t.Error("template reference document must not be modified")
	}
}

func TestPageSetupReferenceDefault(t *testing.T) {
	ms, fr := newDocxTestManuscript(t, map[string]any{"papersize": "a4"})
	got, err := pageSetupReference(ms, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fr.calls) != 1 || !strings.Contains(strings.Join(fr.calls[0], " "), "--print-default-data-file reference.docx") {
		t.Errorf("expected one default-reference extraction, got %v", fr.calls)
	}
	if !strings.Contains(readDocument(t, got), `<w:pgSz w:w="11906" w:h="16838"/>`) {
		t.Error("default reference was not patched")
	}
}

func TestPageSetupReferenceInvalid(t *testing.T) {
	ms, _ := newDocxTestManuscript(t, map[string]any{"papersize": "tabloid"})
	if _, err := pageSetupReference(ms, ""); err == nil {
		t.Error("expected error for an unsupported paper size")
	}
}
