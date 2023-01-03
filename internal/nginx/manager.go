package nginx

import (
	"github.com/r2dtools/webmng/internal/nginx/nginxcli"
	nginxoptions "github.com/r2dtools/webmng/internal/nginx/options"
	"github.com/r2dtools/webmng/internal/nginx/parser"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/webserver"
)

type NginxManager struct {
	nginxCli *nginxcli.NginxCli
	parser   *parser.Parser
	logger   logger.LoggerInterface
}

func (m *NginxManager) GetHosts() ([]webserver.Host, error) {
	nginxHosts, err := m.parser.GetHosts()

	if err != nil {
		return nil, err
	}

	var hosts []webserver.Host

	for _, nginxHost := range nginxHosts {
		hosts = append(hosts, nginxHost.Host)
	}

	return hosts, nil
}

func (m *NginxManager) GetHostsByServerName(serverName string) ([]webserver.Host, error) {
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

func (m *NginxManager) DeployCertificate(serverName, certPath, certKeyPath, chainPath, fullChainPath string) error {
	return nil
}

func (m *NginxManager) EnableHost(host *webserver.Host) error {
	return nil
}

func (m *NginxManager) CommitChanges() error {
	return nil
}

func (m *NginxManager) RollbackChanges() error {
	return nil
}

func (m *NginxManager) SaveChanges() error {
	return nil
}

func GetNginxManager(params map[string]string, logger logger.LoggerInterface) (*NginxManager, error) {
	options := nginxoptions.GetOptions(params)

	nginxCli, err := nginxcli.GetNginxCli(options.Get(nginxoptions.NginxBinPath))
	if err != nil {
		return nil, err
	}

	parser, err := parser.GetParser(options.Get(nginxoptions.ServerRoot), logger)
	if err != nil {
		return nil, err
	}

	manager := NginxManager{
		nginxCli: nginxCli,
		parser:   parser,
		logger:   logger,
	}

	return &manager, nil
}
