package docx

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const docHead = `<?xml version="1.0"?><w:document xmlns:w="w" xmlns:r="r"><w:body><w:p/>`
const docTail = `</w:body></w:document>`

const fullSectPr = `<w:sectPr><w:footerReference w:type="default" r:id="rId9"/>` +
	`<w:pgSz w:w="11906" w:h="16838"/>` +
	`<w:pgMar w:top="851" w:right="1134" w:bottom="851" w:left="1134" w:header="425" w:footer="227" w:gutter="0"/>` +
	`<w:cols w:space="708"/></w:sectPr>`

const stylesXML = `<w:styles>` +
	`<w:style w:type="paragraph" w:styleId="TOC2"><w:pPr><w:tabs>` +
	`<w:tab w:val="right" w:leader="dot" w:pos="9638"/><w:tab w:val="left" w:pos="9638"/>` +
	`</w:tabs></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Normal"><w:pPr><w:tabs>` +
	`<w:tab w:val="right" w:pos="9638"/></w:tabs></w:pPr></w:style></w:styles>`

// writeDocx writes a zip with the given parts (in order) to a temp file.
func writeDocx(t *testing.T, parts [][2]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range parts {
		w, err := zw.Create(p[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(p[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "in.docx")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// readDocx returns the parts of a docx in archive order.
func readDocx(t *testing.T, path string) ([]string, map[string]string) {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	parts := map[string]string{}
	for _, f := range zr.File {
		data, err := readZipFile(f)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, f.Name)
		parts[f.Name] = string(data)
	}
	return names, parts
}

func patch(t *testing.T, doc, styles string, p PageSetup) map[string]string {
	t.Helper()
	src := writeDocx(t, [][2]string{
		{"[Content_Types].xml", "<Types/>"},
		{documentPart, doc},
		{stylesPart, styles},
	})
	dst := filepath.Join(t.TempDir(), "out.docx")
	if err := PatchReference(src, dst, p); err != nil {
		t.Fatalf("PatchReference: %v", err)
	}
	names, parts := readDocx(t, dst)
	if strings.Join(names, ",") != "[Content_Types].xml,word/document.xml,word/styles.xml" {
		t.Errorf("parts or order changed: %v", names)
	}
	if parts["[Content_Types].xml"] != "<Types/>" {
		t.Errorf("untouched part changed: %q", parts["[Content_Types].xml"])
	}
	return parts
}

func assertContains(t *testing.T, s string, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			t.Errorf("missing %q in:\n%s", sub, s)
		}
	}
}

func assertOrder(t *testing.T, s string, subs ...string) {
	t.Helper()
	last := -1
	for _, sub := range subs {
		i := strings.Index(s, sub)
		if i <= last {
			t.Errorf("%q out of order in:\n%s", sub, s)
		}
		last = i
	}
}

func TestPatchReferenceExistingSectPr(t *testing.T) {
	parts := patch(t, docHead+fullSectPr+docTail, stylesXML, PageSetup{
		Width: ptr(12240), Height: ptr(15840),
		Left: ptr(1440), Right: ptr(1440),
	})
	doc := parts[documentPart]
	assertContains(t, doc,
		`<w:pgSz w:w="12240" w:h="15840"/>`,
		`<w:pgMar w:top="851" w:right="1440" w:bottom="851" w:left="1440" w:header="425" w:footer="227" w:gutter="0"/>`,
		`<w:footerReference w:type="default" r:id="rId9"/>`,
	)
	assertOrder(t, doc, "<w:footerReference", "<w:pgSz", "<w:pgMar", "<w:cols")

	// Old text width 11906-2268=9638, new 12240-2880=9360.
	styles := parts[stylesPart]
	assertContains(t, styles,
		`<w:tab w:val="right" w:leader="dot" w:pos="9360"/>`,
		`<w:tab w:val="left" w:pos="9638"/>`,
		`w:styleId="Normal"><w:pPr><w:tabs><w:tab w:val="right" w:pos="9638"/>`,
	)
}

func TestPatchReferenceEmptySectPr(t *testing.T) {
	parts := patch(t, docHead+`<w:sectPr />`+docTail, stylesXML, PageSetup{
		Width: ptr(11906), Height: ptr(16838), Top: ptr(851),
	})
	doc := parts[documentPart]
	assertContains(t, doc,
		`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/>`,
		`<w:pgMar w:top="851" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr>`,
	)
}

func TestPatchReferenceMarginsOnlyKeepsPaper(t *testing.T) {
	parts := patch(t, docHead+fullSectPr+docTail, stylesXML, PageSetup{Bottom: ptr(1000)})
	doc := parts[documentPart]
	assertContains(t, doc, `<w:pgSz w:w="11906" w:h="16838"/>`, `w:bottom="1000"`)
	if parts[stylesPart] != stylesXML {
		t.Error("styles should be untouched when the text width is unchanged")
	}
}

func TestPatchReferenceLandscape(t *testing.T) {
	parts := patch(t, docHead+fullSectPr+docTail, stylesXML, PageSetup{Landscape: ptr(true)})
	assertContains(t, parts[documentPart], `<w:pgSz w:w="16838" w:h="11906" w:orient="landscape"/>`)

	parts = patch(t, docHead+fullSectPr+docTail, stylesXML, PageSetup{
		Width: ptr(12240), Height: ptr(15840), Landscape: ptr(false),
	})
	assertContains(t, parts[documentPart], `<w:pgSz w:w="12240" w:h="15840" w:orient="portrait"/>`)
}

func TestPatchReferenceLastSectionOnly(t *testing.T) {
	inner := `<w:p><w:pPr><w:sectPr><w:pgSz w:w="1" w:h="2"/></w:sectPr></w:pPr></w:p>`
	parts := patch(t, docHead+inner+fullSectPr+docTail, stylesXML, PageSetup{Width: ptr(100)})
	doc := parts[documentPart]
	assertContains(t, doc, `<w:pgSz w:w="1" w:h="2"/>`, `<w:pgSz w:w="100" w:h="16838"/>`)
}

func TestPatchReferenceErrors(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "out.docx")

	noSect := writeDocx(t, [][2]string{{documentPart, docHead + docTail}})
	if err := PatchReference(noSect, dst, PageSetup{Width: ptr(1)}); err == nil {
		t.Error("expected error for a document without w:sectPr")
	}

	noDoc := writeDocx(t, [][2]string{{stylesPart, stylesXML}})
	if err := PatchReference(noDoc, dst, PageSetup{Width: ptr(1)}); err == nil {
		t.Error("expected error for a docx without document.xml")
	}

	if err := PatchReference(filepath.Join(t.TempDir(), "missing.docx"), dst, PageSetup{}); err == nil {
		t.Error("expected error for a missing source")
	}
}
