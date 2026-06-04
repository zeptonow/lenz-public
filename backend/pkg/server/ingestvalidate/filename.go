package ingestvalidate

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

func ValidateBasename(name string, maxLen int) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty filename")
	}
	if maxLen > 0 && len(name) > maxLen {
		return "", fmt.Errorf("filename too long")
	}
	if strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("invalid filename")
	}
	if name == "." || name == ".." {
		return "", fmt.Errorf("invalid filename")
	}
	if filepath.Base(name) != name {
		return "", fmt.Errorf("invalid filename")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", fmt.Errorf("invalid filename")
	}
	if strings.IndexFunc(name, func(r rune) bool { return unicode.IsControl(r) }) != -1 {
		return "", fmt.Errorf("invalid filename")
	}
	return name, nil
}

func LowerExt(name string) string {
	return strings.ToLower(filepath.Ext(name))
}
