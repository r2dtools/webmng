package webserver

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
	CheckConfiguration() error
	Restart() error
}
