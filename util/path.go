package util

import (
	"os"
	"path/filepath"
	"strings"
)

func NormalizePath(path string) (string, error) {
	p := ExpandTilde(path)
	return filepath.Abs(p)
}

func ExpandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		// for Unix
		home := os.Getenv("HOME")
		if home == "" {
			// for Windows
			home = os.Getenv("USERPROFILE")
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
