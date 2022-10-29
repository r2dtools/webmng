package utils

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/unknwon/com"
)

// CheckMinVersion checks if version is higher or equal than minVersion
func CheckMinVersion(version, minVersion string) (bool, error) {
	c, err := semver.NewConstraint(">=" + minVersion)

	if err != nil {
		return false, err
	}

	v, err := semver.NewVersion(version)

	if err != nil {
		return false, err
	}

	return c.Check(v), nil
}

// IsCommandExist checks if command exists via which linus command
func IsCommandExist(name string) bool {
	cmd := exec.Command("which", name)
	if _, err := cmd.Output(); err == nil {
		return true
	}

	return false
}

func FindFirstExistedDirectory(directories []string) (string, error) {
	for _, directory := range directories {
		if com.IsDir(directory) {
			return filepath.Abs(directory)
		}
	}

	return "", fmt.Errorf("none of the directories exist: %s", strings.Join(directories, ", "))
}

// FindAnyFilesInDirectory detects any of the files in the specified directory
func FindAnyFilesInDirectory(directory string, files []string) (string, error) {
	for _, file := range files {
		path := path.Join(directory, file)

		if com.IsFile(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("could not find any of the files \"%s\" in the directory \"%s\"", strings.Join(files, ", "), directory)
}
