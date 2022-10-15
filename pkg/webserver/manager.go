package webserver

import "github.com/r2dtools/webmng/pkg/logger"

type WebServerManagerInterface interface {
	GetHosts() ([]HostInterface, error)
	GetHostsByServerName(serverName string) ([]HostInterface, error)
	EnableHost(host HostInterface) error
	DeployCertificate(serverName, certPath, certKeyPath, chainPath, fullChainPath string) error
	EnsurePortIsListening(port string, https bool) error
	CheckConfiguration() bool
	Restart() error
	SetLogger(logger logger.LoggerInterface)
}
