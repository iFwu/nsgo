// Package appimage identifies the AppImage an executable is running from.
package appimage

import (
	"os"
	"path/filepath"
	"strings"
)

// Path is the AppImage exe runs from, or "". The AppImage runtime names the image in APPIMAGE and its mount in APPDIR, and every process an AppImage app starts inherits both, so the image counts only when exe lies inside that mount.
func Path(exe string) string {
	image, dir := os.Getenv("APPIMAGE"), os.Getenv("APPDIR")
	if !filepath.IsAbs(image) || !filepath.IsAbs(dir) {
		return ""
	}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	rel, err := filepath.Rel(dir, exe)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return image
}
