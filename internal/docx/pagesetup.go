package docx

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// PageSetup holds page geometry in twips (1/1440 in). Nil fields keep the
// reference document's value.
type PageSetup struct {
	Width, Height            *int
	Top, Right, Bottom, Left *int
	Landscape                *bool
}

// IsZero reports whether p requests no change at all.
func (p PageSetup) IsZero() bool {
	return p.Width == nil && p.Height == nil &&
		p.Top == nil && p.Right == nil && p.Bottom == nil && p.Left == nil &&
		p.Landscape == nil
}

// hasMargins reports whether p sets at least one margin.
func (p PageSetup) hasMargins() bool {
	return p.Top != nil || p.Right != nil || p.Bottom != nil || p.Left != nil
}

// unitTwips maps TeX length units to twips per unit.
var unitTwips = map[string]float64{
	"pt": 1440 / 72.27, // TeX point
	"bp": 20,           // PostScript point
	"pc": 12 * 1440 / 72.27,
	"in": 1440,
	"cm": 1440 / 2.54,
	"mm": 144 / 2.54,
}

var lengthRe = regexp.MustCompile(`^([0-9]*\.?[0-9]+)\s*([a-z]+)$`)

// ParseLength converts a TeX length ("2cm", "1in", "72.27pt") to twips.
// Negative lengths and font-relative units (em, ex) are rejected.
func ParseLength(s string) (int, error) {
	m := lengthRe.FindStringSubmatch(strings.ToLower(strings.TrimSpace(s)))
	if m == nil {
		return 0, fmt.Errorf("invalid length %q", s)
	}
	unit, ok := unitTwips[m[2]]
	if !ok {
		return 0, fmt.Errorf("unsupported unit %q in length %q", m[2], s)
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid length %q: %w", s, err)
	}
	return int(math.Round(v * unit)), nil
}

type paperSize struct{ w, h int }

// papers lists portrait paper sizes in twips, keyed by LaTeX name.
var papers = map[string]paperSize{
	"a4":        {11906, 16838},
	"a5":        {8391, 11906},
	"b5":        {9979, 14173},
	"letter":    {12240, 15840},
	"legal":     {12240, 20160},
	"executive": {10440, 15120},
}

// lookupPaper resolves "a4", "A4", "a4paper" and the like.
func lookupPaper(name string) (paperSize, bool) {
	n := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), "paper")
	p, ok := papers[n]
	return p, ok
}

// PageSetupFromMeta reads papersize and geometry from pandoc metadata.
// geometry overrides papersize, and later geometry options override earlier
// ones. Returned warnings name geometry options that were skipped because
// they have no DOCX equivalent.
func PageSetupFromMeta(meta map[string]any) (PageSetup, []string, error) {
	var p PageSetup
	if v, ok := meta["papersize"]; ok {
		name, ok := v.(string)
		if !ok {
			return p, nil, fmt.Errorf("papersize: expected a string, got %T", v)
		}
		size, ok := lookupPaper(name)
		if !ok {
			return p, nil, fmt.Errorf("papersize: unsupported paper size %q", name)
		}
		p.setPaper(size)
	}

	opts, err := geometryOptions(meta["geometry"])
	if err != nil {
		return p, nil, err
	}
	var warnings []string
	for _, opt := range opts {
		warning, err := p.applyGeometry(opt)
		if err != nil {
			return p, nil, fmt.Errorf("geometry: %w", err)
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}
	return p, warnings, nil
}

func (p *PageSetup) setPaper(size paperSize) {
	p.Width, p.Height = ptr(size.w), ptr(size.h)
}

// geometryOptions flattens the geometry value (a string or a list of
// strings, as pandoc accepts) into individual key[=value] options.
func geometryOptions(v any) ([]string, error) {
	switch g := v.(type) {
	case nil:
		return nil, nil
	case string:
		return splitOptions(g), nil
	case []any:
		var opts []string
		for _, item := range g {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("geometry: expected strings, got %T", item)
			}
			opts = append(opts, splitOptions(s)...)
		}
		return opts, nil
	default:
		return nil, fmt.Errorf("geometry: expected a string or list, got %T", v)
	}
}

// splitOptions splits on commas outside braces and drops empty items.
func splitOptions(s string) []string {
	var opts []string
	depth, start := 0, 0
	flush := func(end int) {
		if opt := strings.TrimSpace(s[start:end]); opt != "" {
			opts = append(opts, opt)
		}
	}
	for i, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				flush(i)
				start = i + 1
			}
		}
	}
	flush(len(s))
	return opts
}

// applyGeometry applies one geometry option. It returns a warning for
// options that do not affect the page box.
func (p *PageSetup) applyGeometry(opt string) (string, error) {
	key, val, hasVal := strings.Cut(opt, "=")
	key = strings.ToLower(strings.TrimSpace(key))
	val = strings.TrimSpace(val)

	if !hasVal {
		switch key {
		case "landscape":
			p.Landscape = ptr(true)
			return "", nil
		case "portrait":
			p.Landscape = ptr(false)
			return "", nil
		}
		if size, ok := lookupPaper(key); ok && strings.HasSuffix(key, "paper") {
			p.setPaper(size)
			return "", nil
		}
		return skipped(opt), nil
	}

	switch key {
	case "margin":
		v, err := lengths(key, val, 1, 2)
		if err != nil {
			return "", err
		}
		h, vert := v[0], v[len(v)-1]
		p.Left, p.Right, p.Top, p.Bottom = ptr(h), ptr(h), ptr(vert), ptr(vert)
	case "hmargin":
		v, err := lengths(key, val, 1, 2)
		if err != nil {
			return "", err
		}
		p.Left, p.Right = ptr(v[0]), ptr(v[len(v)-1])
	case "vmargin":
		v, err := lengths(key, val, 1, 2)
		if err != nil {
			return "", err
		}
		p.Top, p.Bottom = ptr(v[0]), ptr(v[len(v)-1])
	case "left", "lmargin", "inner":
		return "", p.setLength(&p.Left, key, val)
	case "right", "rmargin", "outer":
		return "", p.setLength(&p.Right, key, val)
	case "top", "tmargin":
		return "", p.setLength(&p.Top, key, val)
	case "bottom", "bmargin":
		return "", p.setLength(&p.Bottom, key, val)
	case "paper", "papername":
		size, ok := lookupPaper(val)
		if !ok {
			return "", fmt.Errorf("unsupported paper size %q", val)
		}
		p.setPaper(size)
	case "paperwidth":
		return "", p.setLength(&p.Width, key, val)
	case "paperheight":
		return "", p.setLength(&p.Height, key, val)
	case "papersize":
		v, err := lengths(key, val, 2, 2)
		if err != nil {
			return "", err
		}
		p.Width, p.Height = ptr(v[0]), ptr(v[1])
	default:
		return skipped(opt), nil
	}
	return "", nil
}

func (p *PageSetup) setLength(field **int, key, val string) error {
	v, err := lengths(key, val, 1, 1)
	if err != nil {
		return err
	}
	*field = ptr(v[0])
	return nil
}

// lengths parses "len" or "{len,len}" into between minN and maxN twips values.
func lengths(key, val string, minN, maxN int) ([]int, error) {
	inner := strings.TrimSuffix(strings.TrimPrefix(val, "{"), "}")
	parts := splitOptions(inner)
	if len(parts) < minN || len(parts) > maxN {
		return nil, fmt.Errorf("%s: expected %d to %d lengths, got %q", key, minN, maxN, val)
	}
	out := make([]int, len(parts))
	for i, part := range parts {
		v, err := ParseLength(part)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		out[i] = v
	}
	return out, nil
}

func skipped(opt string) string {
	return fmt.Sprintf("geometry option %q has no DOCX equivalent; skipped", opt)
}

func ptr[T any](v T) *T { return &v }
