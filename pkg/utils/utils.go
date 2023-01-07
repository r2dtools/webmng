package utils

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/shirou/gopsutil/host"
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

// GetCommandPath returns path to the binary via "which" linux command
// returns path to the command binary
func GetCommandBinPath(name string) (string, error) {
	var output []byte
	var err error
	cmd := exec.Command("which", name)

	if output, err = cmd.Output(); err != nil {
		return "", err
	}

	return strings.Trim(string(output), "\n"), nil
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

// StrSlicesDifference returns all elements of the first slice which do not present in the second one
func StrSlicesDifference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	var diff []string

	for _, x := range b {
		mb[x] = struct{}{}
	}

	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func IsRhelOsFamily() (bool, error) {
	info, err := host.Info()
	if err != nil {
		return false, err
	}

	return info.PlatformFamily == "rhel" || info.Platform == "almalinux", nil
}
