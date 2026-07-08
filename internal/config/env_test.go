package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-env-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set an environment variable ahead of time to test the overwrite protection.
	os.Setenv("PRE_EXISTING", "original_value")
	defer os.Unsetenv("PRE_EXISTING")

	content := `
# A comment line
KEY_SIMPLE=val1
export KEY_EXPORT=val2

# Inline comments
KEY_COMMENT=val3 # inline comment here

# Quoted values
KEY_DOUBLE="val4\nwith\tescapes" # comment after quote
KEY_SINGLE='val5 # not a comment' # comment after single quote

# Overwrite check
PRE_EXISTING=new_value
`
	err = os.WriteFile(filepath.Join(tempDir, ".env"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write .env file: %v", err)
	}

	err = LoadEnv(tempDir)
	if err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}

	defer func() {
		os.Unsetenv("KEY_SIMPLE")
		os.Unsetenv("KEY_EXPORT")
		os.Unsetenv("KEY_COMMENT")
		os.Unsetenv("KEY_DOUBLE")
		os.Unsetenv("KEY_SINGLE")
	}()

	tests := []struct {
		key      string
		expected string
	}{
		{"KEY_SIMPLE", "val1"},
		{"KEY_EXPORT", "val2"},
		{"KEY_COMMENT", "val3"},
		{"KEY_DOUBLE", "val4\nwith\tescapes"},
		{"KEY_SINGLE", "val5 # not a comment"},
		{"PRE_EXISTING", "original_value"}, // should NOT be overwritten
	}

	for _, tc := range tests {
		got := os.Getenv(tc.key)
		if got != tc.expected {
			t.Errorf("os.Getenv(%q) = %q, expected %q", tc.key, got, tc.expected)
		}
	}
}

func TestLoadEnvAbsent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-env-absent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// No error should be returned if .env is absent.
	err = LoadEnv(tempDir)
	if err != nil {
		t.Errorf("expected no error for absent file, got %v", err)
	}
}
