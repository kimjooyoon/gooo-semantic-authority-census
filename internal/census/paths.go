package census

import (
	"fmt"
	"path/filepath"
	"strings"
)

func containedPath(base, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("path must be a non-empty relative path")
	}
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve input root: %w", err)
	}
	candidate, err := filepath.Abs(filepath.Join(baseAbs, relative))
	if err != nil {
		return "", fmt.Errorf("resolve relative path: %w", err)
	}
	rel, err := filepath.Rel(baseAbs, candidate)
	if err != nil {
		return "", fmt.Errorf("compare input path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes input root")
	}
	return candidate, nil
}
