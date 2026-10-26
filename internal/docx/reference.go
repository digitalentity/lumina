package docx

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	documentPart = "word/document.xml"
	stylesPart   = "word/styles.xml"

	// Word's defaults when a reference document omits w:pgSz or w:pgMar.
	defaultPageW, defaultPageH = 12240, 15840
	defaultMargin              = 1440
	defaultHeaderFooter        = 720
)

// sectPrFollowers lists w:sectPr children that the schema places after
// w:pgMar, in order. w:pgSz additionally precedes w:pgMar.
var sectPrFollowers = []string{
	"paperSrc", "pgBorders", "lnNumType", "pgNumType", "cols", "formProt",
	"vAlign", "noEndnote", "titlePg", "textDirection", "bidi", "rtlGutter",
	"docGrid", "printerSettings", "sectPrChange",
}

var (
	sectPrRe   = regexp.MustCompile(`(?s)<w:sectPr\b[^>]*?/>|<w:sectPr\b[^>]*>.*?</w:sectPr>`)
	tocStyleRe = regexp.MustCompile(`(?s)<w:style\b[^>]*\bw:styleId="TOC[1-9]"[^>]*>.*?</w:style>`)
	tabRe      = regexp.MustCompile(`<w:tab\b[^>]*/>`)
)

// PatchReference copies the reference document src to dst, applying p to
// the document's final section properties. Right-aligned tab stops in the
// TOC1-TOC9 styles that sit on the old right margin are moved to the new
// one. All other parts are copied unchanged.
func PatchReference(src, dst string, p PageSetup) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open reference document: %w", err)
	}
	defer zr.Close()

	parts := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		data, err := readZipFile(f)
		if err != nil {
			return fmt.Errorf("failed to read %s from reference document: %w", f.Name, err)
		}
		parts[f.Name] = data
	}

	doc, ok := parts[documentPart]
	if !ok {
		return fmt.Errorf("reference document has no %s", documentPart)
	}
	newDoc, oldTextW, newTextW, err := patchDocument(string(doc), p)
	if err != nil {
		return err
	}
	parts[documentPart] = []byte(newDoc)

	if styles, ok := parts[stylesPart]; ok && oldTextW != newTextW {
		parts[stylesPart] = []byte(patchTOCTabs(string(styles), oldTextW, newTextW))
	}

	return writeZip(dst, zr.File, parts)
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// writeZip writes parts to dst in the order and with the headers of files.
func writeZip(dst string, files []*zip.File, parts map[string][]byte) (err error) {
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dst, err)
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()

	zw := zip.NewWriter(out)
	for _, f := range files {
		hdr := f.FileHeader
		w, err := zw.CreateHeader(&hdr)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", f.Name, err)
		}
		if _, err := w.Write(parts[f.Name]); err != nil {
			return fmt.Errorf("failed to write %s: %w", f.Name, err)
		}
	}
	return zw.Close()
}

// patchDocument rewrites the last w:sectPr of document.xml. It returns the
// patched XML and the text width (page width minus side margins) before and
// after the change, in twips.
func patchDocument(doc string, p PageSetup) (string, int, int, error) {
	locs := sectPrRe.FindAllStringIndex(doc, -1)
	if len(locs) == 0 {
		return "", 0, 0, errors.New("reference document has no section properties (w:sectPr)")
	}
	start, end := locs[len(locs)-1][0], locs[len(locs)-1][1]
	openTag, inner := splitSectPr(doc[start:end])

	pgSz, hasPgSz := findElement(inner, "pgSz")
	pgMar, hasPgMar := findElement(inner, "pgMar")

	oldW := attrInt(pgSz, "w:w", defaultPageW)
	oldH := attrInt(pgSz, "w:h", defaultPageH)
	oldL := attrInt(pgMar, "w:left", defaultMargin)
	oldR := attrInt(pgMar, "w:right", defaultMargin)

	w, h := deref(p.Width, oldW), deref(p.Height, oldH)
	if p.Landscape != nil && *p.Landscape != (w > h) {
		w, h = h, w
	}
	l, r := deref(p.Left, oldL), deref(p.Right, oldR)

	if p.Width != nil || p.Height != nil || p.Landscape != nil {
		tag := pgSz
		if !hasPgSz {
			tag = "<w:pgSz/>"
		}
		tag = setAttr(tag, "w:w", w)
		tag = setAttr(tag, "w:h", h)
		if p.Landscape != nil {
			tag = setAttrString(tag, "w:orient", orientation(*p.Landscape))
		}
		inner = upsertElement(inner, "pgSz", tag, append([]string{"pgMar"}, sectPrFollowers...))
	}

	if p.hasMargins() {
		tag := pgMar
		if !hasPgMar {
			tag = fmt.Sprintf(`<w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="%d" w:footer="%d" w:gutter="0"/>`,
				defaultMargin, defaultMargin, defaultMargin, defaultMargin, defaultHeaderFooter, defaultHeaderFooter)
		}
		for _, m := range []struct {
			name string
			v    *int
		}{{"w:top", p.Top}, {"w:right", p.Right}, {"w:bottom", p.Bottom}, {"w:left", p.Left}} {
			if m.v != nil {
				tag = setAttr(tag, m.name, *m.v)
			}
		}
		inner = upsertElement(inner, "pgMar", tag, sectPrFollowers)
	}

	patched := doc[:start] + openTag + inner + "</w:sectPr>" + doc[end:]
	return patched, oldW - oldL - oldR, w - l - r, nil
}

// splitSectPr returns the opening tag and the inner XML of a w:sectPr
// element, expanding a self-closing element.
func splitSectPr(s string) (string, string) {
	if strings.HasSuffix(s, "/>") && !strings.Contains(s, "</w:sectPr>") {
		return strings.TrimSuffix(strings.TrimSuffix(s, "/>"), " ") + ">", ""
	}
	open := s[:strings.Index(s, ">")+1]
	return open, strings.TrimSuffix(s[len(open):], "</w:sectPr>")
}

func elementRe(name string) *regexp.Regexp {
	return regexp.MustCompile(`<w:` + name + `\b[^>]*/>`)
}

// findElement returns the first empty element <w:name .../> in s.
func findElement(s, name string) (string, bool) {
	tag := elementRe(name).FindString(s)
	return tag, tag != ""
}

// upsertElement replaces <w:name .../> in s with tag, or inserts tag before
// the first of the given following siblings (or at the end).
func upsertElement(s, name, tag string, followers []string) string {
	re := elementRe(name)
	if loc := re.FindStringIndex(s); loc != nil {
		return s[:loc[0]] + tag + s[loc[1]:]
	}
	at := len(s)
	for _, f := range followers {
		loc := regexp.MustCompile(`<w:` + f + `[\s/>]`).FindStringIndex(s)
		if loc != nil && loc[0] < at {
			at = loc[0]
		}
	}
	return s[:at] + tag + s[at:]
}

func attrRe(name string) *regexp.Regexp {
	return regexp.MustCompile(`\s` + regexp.QuoteMeta(name) + `="([^"]*)"`)
}

// attrInt reads an integer attribute from tag, or returns def.
func attrInt(tag, name string, def int) int {
	m := attrRe(name).FindStringSubmatch(tag)
	if m == nil {
		return def
	}
	v, err := strconv.Atoi(m[1])
	if err != nil {
		return def
	}
	return v
}

func setAttr(tag, name string, v int) string {
	return setAttrString(tag, name, strconv.Itoa(v))
}

// setAttrString sets name="v" on an empty element tag, replacing any
// existing value.
func setAttrString(tag, name, v string) string {
	re := attrRe(name)
	attr := fmt.Sprintf(` %s="%s"`, name, v)
	if re.MatchString(tag) {
		return re.ReplaceAllLiteralString(tag, attr)
	}
	body := strings.TrimRight(strings.TrimSuffix(tag, "/>"), " ")
	return body + attr + "/>"
}

// patchTOCTabs moves right-aligned tab stops at oldPos in TOC1-TOC9 styles
// to newPos.
func patchTOCTabs(styles string, oldPos, newPos int) string {
	return tocStyleRe.ReplaceAllStringFunc(styles, func(style string) string {
		return tabRe.ReplaceAllStringFunc(style, func(tab string) string {
			if !strings.Contains(tab, `w:val="right"`) || attrInt(tab, "w:pos", -1) != oldPos {
				return tab
			}
			return setAttr(tab, "w:pos", newPos)
		})
	})
}

func orientation(landscape bool) string {
	if landscape {
		return "landscape"
	}
	return "portrait"
}

func deref(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}
