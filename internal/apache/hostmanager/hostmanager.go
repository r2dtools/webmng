package hostmanager

import (
	"fmt"

	"github.com/r2dtools/webmng/internal/apache/parser"
)

type HostManager struct {
	parser *parser.Parser
}

func (m HostManager) Enable(hostConfigPath string) error {
	// file is already included
	if m.parser.IsFilenameExistInOriginalPaths(hostConfigPath) {
		return nil
	}

	if err := m.parser.AddInclude(m.parser.ConfigRoot, hostConfigPath); err != nil {
		return fmt.Errorf("could not enable host '%s': %v", hostConfigPath, err)
	}

	return nil
}

func (m HostManager) Disable(hostConfigPath string) error {
	return nil
}

func GetHostManager(parser *parser.Parser) HostManager {
	return HostManager{parser: parser}
}
