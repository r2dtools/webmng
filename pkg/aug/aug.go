package aug

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type augPathParts struct {
	Filepath     string
	InternalPath string
}

// GetFilePathFromAugPath get file path from augeas_vhost_path
func GetFilePathFromAugPath(vhostPath string) string {
	if vhostPath == "" || !strings.HasPrefix(vhostPath, "/files/") {
		return ""
	}

	return splitAugPath(vhostPath).Filepath
}

// GetInternalAugPath returns file augeas internal path
func GetInternalAugPath(vhostPath string) string {
	return splitAugPath(vhostPath).InternalPath
}

// GetAugPath returns Augeas path for the file full path
func GetAugPath(fullPath string) string {
	return fmt.Sprintf("/files/%s", fullPath)
}

func splitAugPath(vhostPath string) augPathParts {
	// exclude trailing '/files'
	var internalPaths []string
	var internalPath string
	var pathExists bool
	path := vhostPath[6:]
	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")

	for !pathExists {
		_, err := os.Stat(path)

		if err != nil {
			internalPath, parts = parts[len(parts)-1], parts[:len(parts)-1]
			internalPaths = append(internalPaths, internalPath)
			path = strings.Join(parts, "/")

			continue
		}

		pathExists = true
		sort.Sort(sort.Reverse(sort.StringSlice(internalPaths)))
	}

	return augPathParts{
		path,
		strings.Join(internalPaths, "/"),
	}
}
