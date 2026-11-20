package appimage

import (
	"path/filepath"
	"testing"
)

func TestPathOnlyUsesAnImageThatContainsTheExecutable(t *testing.T) {
	dir := t.TempDir()
	image := filepath.Join(dir, "Demo.AppImage")
	mount := filepath.Join(dir, ".mount_demo")
	t.Setenv("APPIMAGE", image)
	t.Setenv("APPDIR", mount)

	if got := Path(filepath.Join(mount, "usr", "bin", "demo")); got != image {
		t.Fatalf("Path inside image = %q, want %q", got, image)
	}
	if got := Path(filepath.Join(dir, "terminal")); got != "" {
		t.Fatalf("Path outside image = %q, want empty", got)
	}
}
