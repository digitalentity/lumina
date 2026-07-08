package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnv loads environment variables from a .env file in the specified root directory.
// Variables that are already defined in the environment are not overwritten.
func LoadEnv(root string) error {
	path := filepath.Join(root, ".env")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No .env file, nothing to load
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := parseValue(line[idx+1:])

		if key != "" && os.Getenv(key) == "" {
			if err := os.Setenv(key, val); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

// parseValue extracts the value from a .env line value part, handling quotes and comments.
func parseValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}

	if s[0] == '"' {
		var builder strings.Builder
		for i := 1; i < len(s); i++ {
			if s[i] == '"' {
				return builder.String()
			}
			if s[i] == '\\' && i+1 < len(s) {
				i++
				switch s[i] {
				case 'n':
					builder.WriteByte('\n')
				case 'r':
					builder.WriteByte('\r')
				case 't':
					builder.WriteByte('\t')
				case '\\':
					builder.WriteByte('\\')
				case '"':
					builder.WriteByte('"')
				case '\'':
					builder.WriteByte('\'')
				default:
					builder.WriteByte('\\')
					builder.WriteByte(s[i])
				}
			} else {
				builder.WriteByte(s[i])
			}
		}
		return builder.String()
	}

	if s[0] == '\'' {
		var builder strings.Builder
		for i := 1; i < len(s); i++ {
			if s[i] == '\'' {
				return builder.String()
			}
			builder.WriteByte(s[i])
		}
		return builder.String()
	}

	// Unquoted value: strip inline comments
	if idx := strings.Index(s, "#"); idx != -1 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}
