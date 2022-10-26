package nginx

import (
	"github.com/r2dtools/webmng/internal/nginx/nginxcli"
	nginxoptions "github.com/r2dtools/webmng/internal/nginx/options"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/webserver"
)

type NginxManager struct {
	nginxCli *nginxcli.NginxCli
	logger   logger.LoggerInterface
}

func (m *NginxManager) GetHosts() ([]*webserver.Host, error) {
	return nil, nil
}

func (m *NginxManager) GetVersion() (string, error) {
	return m.nginxCli.GetVersion()
}

func (m *NginxManager) CheckConfiguration() error {
	return m.nginxCli.TestConfiguration()
}

func (m *NginxManager) Restart() error {
	return m.nginxCli.Restart()
}

func (m *NginxManager) SetLogger(logger logger.LoggerInterface) {
	m.logger = logger
}

func GetNginxManager(params map[string]string) (*NginxManager, error) {
	options := nginxoptions.GetOptions(params)

	nginxCli, err := nginxcli.GetNginxCli(options.Get(nginxoptions.NginxBinPath))
	if err != nil {
		return nil, err
	}

	manager := NginxManager{
		nginxCli: nginxCli,
		logger:   logger.NilLogger{},
	}

	return &manager, nil
}
