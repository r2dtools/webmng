package webserver

import "github.com/r2dtools/webmng/pkg/logger"

const (
	Apache = "apache"
	Nginx  = "nginx"
)

type WebServerManagerInterface interface {
	GetHosts() ([]*Host, error)
	GetVersion() (string, error)
	// GetHostsByServerName(serverName string) ([]HostInterface, error)
	// EnableHost(host HostInterface) error
	// DeployCertificate(serverName, certPath, certKeyPath, chainPath, fullChainPath string) error
	// EnsurePortIsListening(port string, https bool) error
	CheckConfiguration() bool
	Restart() error
	SetLogger(logger logger.LoggerInterface)
}
