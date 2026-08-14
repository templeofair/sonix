package service

import (
	"path/filepath"
	"testing"
)

func TestPathInside(t *testing.T) {
	root := filepath.Join(t.TempDir(), "uploads")
	ok := filepath.Join(root, "1", "page_0.jpg")
	if !PathInside(root, ok) {
		t.Fatal("page under uploads")
	}
	if !PathInside(root, root) {
		t.Fatal("root itself")
	}
	evil := filepath.Join(filepath.Dir(root), "uploads_evil", "x")
	if PathInside(root, evil) {
		t.Fatal("sibling prefix must not match")
	}
	if PathInside(root, filepath.Join(root, "..", "etc", "passwd")) {
		t.Fatal(".. escape")
	}
}
