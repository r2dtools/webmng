package hostmanager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/unknwon/com"
)

// HostManager enables/disables host by creating/removing symlink for a configuration file
type HostManager struct {
	enabledHostConfigDir string
}

// path to "available" host config. For example /etc/.../sites-available/
func (m HostManager) Enable(hostConfigPath string) error {
	hostConfigName := filepath.Base(hostConfigPath)
	enabledHostConfigPath := filepath.Join(m.enabledHostConfigDir, hostConfigName)

	if err := os.Symlink(hostConfigPath, enabledHostConfigPath); err != nil {
		return fmt.Errorf("could not enable host %s: %v", hostConfigPath, err)
	}

	return nil
}

// path to "available" host config. For example /etc/.../sites-available/
func (m HostManager) Disable(hostConfigPath string) error {
	hostConfigName := filepath.Base(hostConfigPath)
	hostConfigPath = filepath.Join(m.enabledHostConfigDir, hostConfigName)

	if _, err := os.Lstat(hostConfigPath); err == nil {
		err = os.Remove(hostConfigPath)

		return fmt.Errorf("could not disable host %s: %v", hostConfigPath, err)
	}

	return nil
}

func GetHostManager(enabledHostConfigDir string) (HostManager, error) {
	if !com.IsDir(enabledHostConfigDir) {
		return HostManager{}, errors.New("directory for enabled hosts does not exist")
	}

	return HostManager{enabledHostConfigDir: enabledHostConfigDir}, nil
}
