package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func Getwd() string {
	wd, _ := os.Getwd()
	return wd
}

func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
