package apachesite

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/r2dtools/webmng/pkg/utils"
)

// Site implements functionality for site enabling/disabling
type ApacheSite struct {
	dissiteBin, ensiteBin string
}

// Enable enables site via a2ensite utility
func (s *ApacheSite) Enable(hostConfigPath string) error {
	hostConfigName := filepath.Base(hostConfigPath)
	if !utils.IsCommandExist(s.ensiteBin) {
		return fmt.Errorf("could not enable host '%s': a2ensite utility is not available", hostConfigName)
	}

	_, err := s.execCmd(s.ensiteBin, []string{hostConfigName})

	if err != nil {
		return fmt.Errorf("could not enable host '%s': %v", hostConfigName, err)
	}

	return nil
}

// Disable disables site via a2dissite utility
func (s *ApacheSite) Disable(hostConfigPath string) error {
	hostConfigName := filepath.Base(hostConfigPath)
	if !utils.IsCommandExist(s.dissiteBin) {
		return fmt.Errorf("could not disable host '%s': a2dissite utility is not available", hostConfigName)
	}

	_, err := s.execCmd(s.dissiteBin, []string{hostConfigName})

	if err != nil {
		return fmt.Errorf("could not disable host '%s': %v", hostConfigName, err)
	}

	return nil
}

func (s *ApacheSite) execCmd(command string, params []string) ([]byte, error) {
	cmd := exec.Command(command, params...)
	output, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("could not execute '%s' command: %v", command, err)
	}

	return output, nil
}

func GetApacheSite(ensiteBin, dissiteBin string) *ApacheSite {
	return &ApacheSite{ensiteBin: ensiteBin, dissiteBin: dissiteBin}
}
