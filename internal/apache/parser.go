package apache

import (
	"fmt"
	"path/filepath"

	"github.com/r2dtools/webmng/internal/apache/apachectl"
	"github.com/unknwon/com"
	"honnef.co/go/augeas"
)

type Parser struct {
	augeas    augeas.Augeas
	apachectl *apachectl.ApacheCtl
	serverRoot,
	hostRoot,
	version string
}

func GetParser(apachectl *apachectl.ApacheCtl, version, serverRoot, hostRoot string) (*Parser, error) {
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

	aug, err := augeas.New("/", "", augeas.NoLoad|augeas.NoModlAutoload|augeas.EnableSpan)
	if err != nil {
		return nil, err
	}

	parser := Parser{
		augeas:     aug,
		apachectl:  apachectl,
		serverRoot: serverRoot,
		hostRoot:   hostRoot,
		version:    version,
	}

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
