package apachesite

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/r2dtools/webmng/pkg/utils"
	"github.com/unknwon/com"
)

// Site implements functionality for site enabling/disabling
type ApacheSite struct {
	dissiteBin, ensiteBin string
}

// Enable enables site via a2ensite utility
func (s ApacheSite) Enable(hostConfigPath string) error {
	hostConfigName := filepath.Base(hostConfigPath)

	if _, err := s.execCmd(s.ensiteBin, []string{hostConfigName}); err != nil {
		return fmt.Errorf("could not enable host '%s': %v", hostConfigName, err)
	}

	return nil
}

// Disable disables site via a2dissite utility
func (s ApacheSite) Disable(hostConfigName string) error {
	if _, err := s.execCmd(s.dissiteBin, []string{hostConfigName}); err != nil {
		return fmt.Errorf("could not disable host '%s': %v", hostConfigName, err)
	}

	return nil
}

func (s ApacheSite) execCmd(command string, params []string) ([]byte, error) {
	cmd := exec.Command(command, params...)
	output, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("could not execute '%s' command: %v", command, err)
	}

	return output, nil
}

func GetApacheSite() (ApacheSite, error) {
	ensiteBinPaths := []string{"/usr/sbin/a2ensite"}
	dissiteBinPaths := []string{"/usr/sbin/a2dissite"}

	ensiteBin, err := utils.GetCommandBinPath("a2ensite")
	if err != nil {
		for _, cmdPath := range ensiteBinPaths {
			if com.IsFile(cmdPath) {
				ensiteBin = cmdPath
				break
			}
		}
	}

	dissiteBin, err := utils.GetCommandBinPath("a2dissite")
	if err != nil {
		for _, cmdPath := range dissiteBinPaths {
			if !com.IsFile(cmdPath) {
				dissiteBin = cmdPath
				break
			}
		}
	}

	if ensiteBin == "" || dissiteBin == "" {
		return ApacheSite{}, errors.New("a2ensite/a2dissite binaries do not exist")
	}

	return ApacheSite{ensiteBin: ensiteBin, dissiteBin: dissiteBin}, nil
}
