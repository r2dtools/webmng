package apache

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/r2dtools/webmng/internal/apache/apachectl"
	"github.com/unknwon/com"
	"honnef.co/go/augeas"
)

type Parser struct {
	augeas    augeas.Augeas
	apachectl *apachectl.ApacheCtl
	serverRoot,
	configRoot,
	configListen,
	hostRoot,
	version string
	existingPaths map[string][]string
}

// Close closes the Parser instance and frees any storage associated with it.
func (p *Parser) Close() {
	if p != nil {
		p.augeas.Close()
	}
}

func GetParser(apachectl *apachectl.ApacheCtl, version, serverRoot, hostRoot, hostFiles string) (*Parser, error) {
	serverRoot, err := getServerRootPath(serverRoot)
	if err != nil {
		return nil, err
	}

	if hostRoot != "" {
		hostRoot, err = filepath.Abs(hostRoot)
		if err != nil {
			return nil, err
		}
	}

	// try to detect apache root config file path (ex. /etc/apache2/apache2.conf), ports.conf file path
	configRoot, err := getConfigRootPath(serverRoot)
	if err != nil {
		return nil, err
	}

	configListen := getConfigListen(serverRoot, configRoot)

	aug, err := augeas.New("/", "", augeas.NoLoad|augeas.NoModlAutoload|augeas.EnableSpan)
	if err != nil {
		return nil, err
	}

	parser := Parser{
		augeas:        aug,
		apachectl:     apachectl,
		serverRoot:    serverRoot,
		configRoot:    configRoot,
		configListen:  configListen,
		hostRoot:      hostRoot,
		version:       version,
		existingPaths: make(map[string][]string),
	}

	/*
		if err = parser.ParseFile(parser.ConfigRoot); err != nil {
			parser.Close()
			return nil, fmt.Errorf("could not parse apache config: %v", err)
		}

		if err = parser.UpdateRuntimeVariables(); err != nil {
			return nil, err
		}

		// list of the active include paths, before modifications
		for k, v := range parser.Paths {
			dst := make([]string, len(v))
			copy(dst, v)
			parser.existingPaths[k] = dst
		}

		if hostRoot != "" && hostFiles != "" {
			vhostFilesPath := filepath.Join(hostRoot, hostFiles)

			if err = parser.ParseFile(vhostFilesPath); err != nil {
				return nil, err
			 }
		}*/

	return &parser, nil
}

func getServerRootPath(serverRootPath string) (string, error) {
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

// getConfigRootPath detects apache root config file
func getConfigRootPath(serverRoot string) (string, error) {
	configs := []string{"apache2.conf", "httpd.conf", "conf/httpd.conf"}

	for _, config := range configs {
		configRootPath := path.Join(serverRoot, config)

		if com.IsFile(configRootPath) {
			return configRootPath, nil
		}
	}

	return "", fmt.Errorf("could not find apache config file \"%s\" in the root directory \"%s\"", strings.Join(configs, ", "), serverRoot)
}

func getConfigListen(serverRoot, configRoot string) string {
	configPorts := filepath.Join(serverRoot, "ports.conf")

	if com.IsFile(configPorts) {
		return configPorts
	}

	return configRoot
}
