package lib

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
		return filepath.Join(Home(), path[2:])
	}
	return path
}

func MapHomeToTilde(path string) string {
	home := Home()
	if strings.HasPrefix(path, home) {
		return strings.Replace(path, home, "~", 1)
	}
	return path
}

func Home() string {
	// for Unix
	home := os.Getenv("HOME")
	if home == "" {
		// for Windows
		home = os.Getenv("USERPROFILE")
	}
	return home
}
