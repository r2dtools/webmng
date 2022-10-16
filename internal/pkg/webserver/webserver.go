package webserver

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/unknwon/com"
)

func GetServerRootPath(serverRootPath string, serverRootPaths []string) (string, error) {
	if serverRootPath != "" {
		return filepath.Abs(serverRootPath)
	}

	// check default paths
	for _, serverRootPath := range serverRootPaths {
		if com.IsDir(serverRootPath) {
			return filepath.Abs(serverRootPath)
		}
	}

	return "", fmt.Errorf("could not find server root path")
}

// getConfigRootPath detects webserver root config file
func GetConfigRootPath(serverRoot string, configs []string) (string, error) {
	for _, config := range configs {
		configRootPath := path.Join(serverRoot, config)

		if com.IsFile(configRootPath) {
			return configRootPath, nil
		}
	}

	return "", fmt.Errorf("could not find webserver config file \"%s\" in the root directory \"%s\"", strings.Join(configs, ", "), serverRoot)
}
