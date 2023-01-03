package webserver

const (
	Apache = "apache"
	Nginx  = "nginx"
)

type WebServerManagerInterface interface {
	GetHosts() ([]Host, error)
	GetVersion() (string, error)
	GetHostsByServerName(serverName string) ([]Host, error)
	EnableHost(host *Host) error
	DeployCertificate(serverName, certPath, certKeyPath, chainPath, fullChainPath string) error
	CheckConfiguration() error
	Restart() error
	SaveChanges() error
	CommitChanges() error
	RollbackChanges() error
}
