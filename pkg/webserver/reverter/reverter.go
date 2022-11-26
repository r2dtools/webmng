package reverter

type Reverter interface {
	BackupFile(filePath string)
	BackupFiles(files []string) error
	AddFileToDeletion(filePath string)
	AddHostConfigToDisable(configName string)
	Commit() error
	Rollback() error
}

type NullReverter struct{}

func (r NullReverter) BackupFile(filePath string) {}

func (r NullReverter) BackupFiles(files []string) error {
	return nil
}

func (r NullReverter) AddFileToDeletion(filePath string) {}

func (r NullReverter) AddHostConfigToDisable(configName string) {}

func (r NullReverter) Commit() error {
	return nil
}

func (r NullReverter) Rollback() error {
	return nil
}
