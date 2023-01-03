package reverter

import (
	"fmt"
	"os"

	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/unknwon/com"
	"golang.org/x/exp/slices"
)

type HostDisabler interface {
	Disable(hostConfigPath string) error
}

type Reverter interface {
	BackupFile(filePath string) error
	BackupFiles(filePaths []string) error
	AddFileToDeletion(filePath string)
	AddHostConfigToDisable(configPath string)
	Commit() error
	Rollback() error
}

type rollbackError struct {
	err error
}

func (e rollbackError) Error() string {
	return fmt.Sprintf("configuration rollback failed: %v", e.err)
}

// configReverter reverts change back for configuration files of virtual hosts
type configReverter struct {
	filesToDelete    []string
	filesToRestore   map[string]string
	configsToDisable []string
	hostDisabler     HostDisabler
	logger           logger.LoggerInterface
}

// AddFileToDeletion marks file to delete on rollback
func (r *configReverter) AddFileToDeletion(filePath string) {
	r.filesToDelete = append(r.filesToDelete, filePath)
}

// AddHostConfigToDisable marks apache site config as needed to be disabled on rollback
func (r *configReverter) AddHostConfigToDisable(configPath string) {
	r.configsToDisable = append(r.configsToDisable, configPath)
}

// BackupFiles makes files backups
func (r *configReverter) BackupFiles(filePaths []string) error {
	for _, filePath := range filePaths {
		if err := r.BackupFile(filePath); err != nil {
			return fmt.Errorf("could not make file '%s' backup: %v", filePath, err)
		}
	}

	return nil
}

// BackupFile makes file backup. The file content will be restored on rollback.
func (r *configReverter) BackupFile(filePath string) error {
	bFilePath := getBackupFilePath(filePath)

	if _, ok := r.filesToRestore[filePath]; ok {
		r.logger.Debug(fmt.Sprintf("file '%s' is already backed up.", filePath))
		return nil
	}

	// Skip file backup if it should be removed
	if slices.Contains(r.filesToDelete, filePath) {
		r.logger.Debug(fmt.Sprintf("file '%s' will be removed on rollback. Skip its backup.", filePath))
		return nil
	}

	content, err := os.ReadFile(filePath)

	if err != nil {
		return err
	}

	err = os.WriteFile(bFilePath, content, 0644)

	if err != nil {
		return err
	}

	r.filesToRestore[filePath] = bFilePath

	return nil
}

// Rollback rollback all changes
func (r *configReverter) Rollback() error {
	// Disable all enabled before hosts
	for _, configPath := range r.configsToDisable {
		if err := r.hostDisabler.Disable(configPath); err != nil {
			return rollbackError{err}
		}
	}

	// remove created files
	for _, fileToDelete := range r.filesToDelete {
		if !com.IsFile(fileToDelete) {
			r.logger.Debug(fmt.Sprintf("file '%s' does not exist. Skip its deletion.", fileToDelete))
			continue
		}

		if err := os.Remove(fileToDelete); err != nil {
			return rollbackError{err}
		}
	}

	// restore the content of backed up files
	for originFilePath, bFilePath := range r.filesToRestore {
		bContent, err := os.ReadFile(bFilePath)

		if err != nil {
			return rollbackError{err}
		}

		err = os.WriteFile(originFilePath, bContent, 0644)

		if err != nil {
			return rollbackError{err}
		}

		if err := os.Remove(bFilePath); err != nil {
			r.logger.Error(fmt.Sprintf("could not remove file '%s' on reverter rollback: %v", bFilePath, err))
		}

		delete(r.filesToRestore, originFilePath)
	}

	return nil
}

// Commit commits changes. All *.back files will be removed.
func (r *configReverter) Commit() error {
	for filePath, bFilePath := range r.filesToRestore {
		if com.IsFile(bFilePath) {
			if err := os.Remove(bFilePath); err != nil {
				r.logger.Error(fmt.Sprintf("could not remove file '%s' on commit: %v", bFilePath, err))
			}
		}

		delete(r.filesToRestore, filePath)
	}

	r.filesToDelete = nil

	return nil
}

func getBackupFilePath(filePath string) string {
	return filePath + ".back"
}

func GetConfigReveter(hostDisabler HostDisabler, logger logger.LoggerInterface) Reverter {
	reverter := configReverter{
		hostDisabler:   hostDisabler,
		logger:         logger,
		filesToRestore: make(map[string]string),
	}

	return &reverter
}
