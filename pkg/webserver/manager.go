package webserver

type WebServerManagerInterface interface {
	GetHosts() ([]*Host, error)
	GetHostsByServerName(serverName string) ([]*Host, error)
	EnableHost(vhost *Host) error
	DeployCertificate(serverName, certPath, certKeyPath, chainPath, fullChainPath string) error
	EnsurePortIsListening(port string, https bool) error
	CheckConfiguration() bool
	Restart() error
}
