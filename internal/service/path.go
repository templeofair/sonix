package service

import (
	"path/filepath"
	"strings"
)

// PathInside reports whether candidate is root itself or a path under root.
// Uses filepath.Rel so a prefix like "/data/uploads" does not match "/data/uploads_evil".
func PathInside(root, candidate string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absCand, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absCand)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
