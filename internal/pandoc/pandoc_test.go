package pandoc

import (
	"path/filepath"
	"reflect"
	"testing"

	"lumina/internal/manuscript"
)

type MockRunner struct {
	LastTool string
	LastArgs []string
}

func (m *MockRunner) Run(tool string, args []string, cwd string) error {
	m.LastTool = tool
	m.LastArgs = args
	return nil
}

func (m *MockRunner) Capture(tool string, args []string, cwd string) ([]byte, error) {
	m.LastTool = tool
	m.LastArgs = args
	return []byte("mocked output"), nil
}

func (m *MockRunner) CheckPresent(tool string) error {
	return nil
}

func TestInvocationRunHost(t *testing.T) {
	mr := &MockRunner{}
	ms := &manuscript.Manuscript{
		Root:   "/home/user/paper",
		Runner: mr,
	}

	inv := &Invocation{
		Input:        "/home/user/paper/.lumina/manuscript.md",
		MetadataFile: "/home/user/paper/.lumina/metadata.yaml",
		Output:       "/home/user/paper/_build/manuscript.pdf",
		Filters:      []string{"pandoc-crossref"},
		ExtraFlags:   []string{"-s"},
		Template:     "/home/user/paper/publish/template.tex",
	}

	err := inv.Run(ms)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := []string{
		"--metadata-file", "/home/user/paper/.lumina/metadata.yaml",
		"--filter", "pandoc-crossref",
		"--template", "/home/user/paper/publish/template.tex",
		"-s",
		"-o", "/home/user/paper/_build/manuscript.pdf",
		"/home/user/paper/.lumina/manuscript.md",
	}

	if !reflect.DeepEqual(mr.LastArgs, expectedArgs) {
		t.Errorf("got args %+v, expected %+v", mr.LastArgs, expectedArgs)
	}
}

func TestRewriteToWorkspace(t *testing.T) {
	root := "/home/user/paper"
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/paper/.lumina/manuscript.md", "/workspace/.lumina/manuscript.md"},
		{"/home/user/paper/publish/template.tex", "/workspace/publish/template.tex"},
		{"/outside/path", "/outside/path"},
		{"", ""},
	}

	for _, tc := range tests {
		got := rewriteToWorkspace(tc.path, root)
		expectedClean := tc.expected
		if tc.expected != "" {
			expectedClean = filepath.Clean(tc.expected)
		}
		if got != expectedClean {
			t.Errorf("for %q, expected %q, got %q", tc.path, expectedClean, got)
		}
	}
}
