package nginx

import (
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/webserver"
)

type NginxManager struct {
	logger logger.LoggerInterface
}

func (m *NginxManager) GetHosts() ([]*webserver.Host, error) {
	return nil, nil
}

func (m *NginxManager) CheckConfiguration() bool {
	return false
}

func (m *NginxManager) Restart() error {
	return nil
}

func (m *NginxManager) SetLogger(logger logger.LoggerInterface) {
	m.logger = logger
}

func GetNginxManager(params map[string]string) (*NginxManager, error) {
	manager := NginxManager{
		logger: logger.NilLogger{},
	}

	return &manager, nil
}
