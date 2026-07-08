package preprocess

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gtext "github.com/yuin/goldmark/text"
	"lumina/internal/runner"
)

type mermaidConfig struct {
	Theme          string            `json:"theme"`
	ThemeVariables map[string]string `json:"themeVariables"`
}

func mermaidConfigJSON() ([]byte, error) {
	return json.Marshal(mermaidConfig{
		Theme: "neutral",
		ThemeVariables: map[string]string{
			"background":       "#ffffff",
			"primaryColor":     "#ffffff",
			"primaryTextColor": "#000000",
			"lineColor":        "#000000",
			"mainBkg":          "#ffffff",
			"nodeBorder":       "#000000",
			"actorBkg":         "#ffffff",
			"actorBorder":      "#000000",
			"noteBkgColor":     "#ffffff",
			"noteBorderColor":  "#000000",
		},
	})
}

type mmdToRender struct {
	code []byte
	path string
}

// FindMermaidBlocks parses the Goldmark AST and extracts Mermaid blocks
// along with their code, cache path, and target replacement range.
func FindMermaidBlocks(content []byte, figuresDir string) ([]replacement, []mmdToRender) {
	doc := goldmark.DefaultParser().Parse(gtext.NewReader(content))
	var replacements []replacement
	var mmdsToRender []mmdToRender

	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		if fcb, ok := n.(*gast.FencedCodeBlock); ok {
			if string(fcb.Language(content)) == "mermaid" {
				lines := fcb.Lines()
				if lines.Len() > 0 {
					contentStart := lines.At(0).Start
					contentEnd := lines.At(lines.Len() - 1).Stop
					blkStart := lineStart(content, contentStart)
					blkEnd := extendThroughClosingFence(content, contentEnd)
					code := bytes.TrimSpace(lines.Value(content))

					sum := sha256.Sum256(code)
					sha := hex.EncodeToString(sum[:])[:16]
					pngFilename := fmt.Sprintf("mermaid-%s.png", sha)
					pngPath := filepath.Join(figuresDir, pngFilename)

					replacements = append(replacements, replacement{
						start: blkStart,
						end:   blkEnd,
						text:  fmt.Sprintf("![Mermaid Diagram](figures/%s)", pngFilename),
					})

					mmdsToRender = append(mmdsToRender, mmdToRender{
						code: code,
						path: pngPath,
					})
				}
				return gast.WalkSkipChildren, nil
			}
		}
		return gast.WalkContinue, nil
	})

	return replacements, mmdsToRender
}

// RenderMermaid invokes mmdc via the runner using temp files stored in the manuscript's .lumina directory
// to ensure they are accessible inside Docker containers.
func RenderMermaid(run runner.Runner, code []byte, pngPath, luminaDir string) error {
	cfgJSON, err := mermaidConfigJSON()
	if err != nil {
		return err
	}

	// Create temp files in .lumina directory so DockerRunner can rewrite their paths
	mmdFile, err := os.CreateTemp(luminaDir, "mermaid-*.mmd")
	if err != nil {
		return err
	}
	defer os.Remove(mmdFile.Name())
	if _, err := mmdFile.Write(code); err != nil {
		_ = mmdFile.Close()
		return err
	}
	_ = mmdFile.Close()

	cfgFile, err := os.CreateTemp(luminaDir, "mermaid-config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(cfgFile.Name())
	if _, err := cfgFile.Write(cfgJSON); err != nil {
		_ = cfgFile.Close()
		return err
	}
	_ = cfgFile.Close()

	pupFile, err := os.CreateTemp(luminaDir, "mermaid-puppeteer-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(pupFile.Name())
	if _, err := pupFile.Write([]byte(`{"args":["--no-sandbox","--disable-setuid-sandbox"]}`)); err != nil {
		_ = pupFile.Close()
		return err
	}
	_ = pupFile.Close()

	args := []string{
		"-i", mmdFile.Name(),
		"-o", pngPath,
		"-s", "3",
		"-b", "white",
		"-c", cfgFile.Name(),
		"-p", pupFile.Name(),
	}

	return run.Run("mmdc", args, ".")
}

func lineStart(source []byte, pos int) int {
	if pos == 0 {
		return 0
	}
	if idx := bytes.LastIndexByte(source[:pos-1], '\n'); idx != -1 {
		return idx + 1
	}
	return 0
}

func extendThroughClosingFence(source []byte, pos int) int {
	if pos >= len(source) {
		return pos
	}
	end := pos + bytes.IndexByte(source[pos:], '\n')
	if end < pos {
		end = len(source)
	}
	trimmed := bytes.TrimSpace(source[pos:end])
	if len(trimmed) < 3 {
		return pos
	}
	for _, c := range trimmed {
		if c != '`' && c != '~' {
			return pos
		}
	}
	return end
}
