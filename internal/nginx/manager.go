package nginx

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/r2dtools/webmng/internal/nginx/nginxcli"
	nginxoptions "github.com/r2dtools/webmng/internal/nginx/options"
	"github.com/r2dtools/webmng/internal/nginx/parser"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/options"
	"github.com/r2dtools/webmng/pkg/webserver"
	"github.com/r2dtools/webmng/pkg/webserver/hostmanager"
	webserverOptions "github.com/r2dtools/webmng/pkg/webserver/options"
	"github.com/r2dtools/webmng/pkg/webserver/reverter"
	"github.com/unknwon/com"
	"golang.org/x/exp/slices"
)

const defaultListenPort = 80

var enabledHostConfigDirNames = []string{"sites-enabled", "conf.d"}

type ipv6Info struct {
	isIpv6Active, isIpv6OnlyPresent bool
}

type NginxManager struct {
	nginxCli nginxcli.NginxCli
	parser   *parser.Parser
	logger   logger.LoggerInterface
	options  options.Options
	reverter reverter.Reverter
}

func (m *NginxManager) GetHosts() ([]webserver.Host, error) {
	nginxHosts, err := m.parser.GetHosts()
	if err != nil {
		return nil, err
	}

	return m.convertNginxHostsToWebserverHosts(nginxHosts), nil
}

func (m *NginxManager) GetHostsByServerName(serverName string) ([]webserver.Host, error) {
	nginxHosts, err := m.getNginxHostsByServerName(serverName)
	if err != nil {
		return nil, err
	}

	return m.convertNginxHostsToWebserverHosts(nginxHosts), nil
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
	if fullChainPath == "" {
		return errors.New("nginx requires fullchain-path to deploy a certificate")
	}

	if certKeyPath == "" {
		return errors.New("nginx requires cert key path to deploy a certificate")
	}

	hosts, err := m.getNginxHostsByServerName(serverName)
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		return fmt.Errorf("unable to install certificate to %s: host does not exist", serverName)
	}

	for _, host := range hosts {
		if !host.Ssl {
			if err := m.makeSslHost(&host, certKeyPath, fullChainPath); err != nil {
				return err
			}
		}

		if err = m.deployCertificateToHost(&host, certKeyPath, fullChainPath); err != nil {
			return err
		}
	}

	return nil
}

func (m *NginxManager) makeSslHost(host *parser.NginxHost, certKeyPath, fullChainPath string) error {
	httpsPort := m.options.Get(webserverOptions.HttpsPort)

	ipv6Info, err := m.getIpv6Info(httpsPort)
	var ipv4Block, ipv6Block *parser.NginxDirective

	if err != nil {
		return err
	}

	if len(host.Addresses) == 0 {
		listenBlock := []*parser.NginxDirective{
			{
				Name:          "listen",
				Values:        []string{strconv.Itoa(defaultListenPort)},
				NewLineBefore: true,
			},
		}

		m.parser.AddServerDirectives(host, listenBlock, true)
	}

	if host.IsIpv6Enabled() {
		ipv6Block = &parser.NginxDirective{
			Name:          "listen",
			Values:        []string{fmt.Sprintf("[::]:%s", httpsPort), "ssl"},
			NewLineBefore: true,
		}

		// ipv6only=on is absent in global config
		if !ipv6Info.isIpv6OnlyPresent {
			ipv6Block.AddValues("ipv6only=on")
		}
	}

	if host.IsIpv4Enabled() {
		ipv4Block = &parser.NginxDirective{
			Name:          "listen",
			Values:        []string{httpsPort, " ", "ssl"},
			NewLineBefore: true,
		}
	}

	sslBlock := []*parser.NginxDirective{
		ipv6Block,
		ipv4Block,
		{
			Name:          "ssl_certificate_key",
			Values:        []string{certKeyPath},
			NewLineBefore: true,
		},
		{
			Name:          "ssl_certificate",
			Values:        []string{fullChainPath},
			NewLineBefore: true,
			NewLineAfter:  true,
		},
	}

	if err := m.parser.AddServerDirectives(host, sslBlock, false); err != nil {
		return err
	}

	host.Ssl = true

	return nil
}

func (m *NginxManager) deployCertificateToHost(host *parser.NginxHost, certKeyPath, fullChainPath string) error {
	certDirectives := []*parser.NginxDirective{
		{
			Name:          "ssl_certificate_key",
			Values:        []string{certKeyPath},
			NewLineBefore: true,
		},
		{
			Name:          "ssl_certificate",
			Values:        []string{fullChainPath},
			NewLineBefore: true,
			NewLineAfter:  true,
		},
	}

	return m.parser.UpdateOrAddServerDirectives(host, certDirectives, false)
}

func (m *NginxManager) EnableHost(host *webserver.Host) error {
	return nil
}

func (m *NginxManager) CommitChanges() error {
	return m.reverter.Commit()
}

func (m *NginxManager) RollbackChanges() error {
	return m.reverter.Rollback()
}

func (m *NginxManager) SaveChanges() error {
	changedFiles := m.parser.GetChangedFiles()
	if err := m.reverter.BackupFiles(changedFiles); err != nil {
		return err
	}

	return m.parser.Dump()
}

func (m *NginxManager) getNginxHostsByServerName(serverName string) ([]parser.NginxHost, error) {
	nHosts, err := m.parser.GetHosts()

	if err != nil {
		return nil, err
	}

	var suitableHosts []parser.NginxHost
	var suitableNonSslHosts []parser.NginxHost
	var sslHostsAddresses []string

	for _, nHost := range nHosts {
		// Prefer host with ssl
		if nHost.ServerName == serverName {
			if nHost.Ssl {
				suitableHosts = append(suitableHosts, nHost)
				sslHostsAddresses = append(sslHostsAddresses, nHost.GetAddressesString(true))
			} else {
				suitableNonSslHosts = append(suitableNonSslHosts, nHost)
			}
		}
	}

	for _, host := range suitableNonSslHosts {
		// skip non ssl hosts if there is already ssl host with the same address
		if !slices.Contains(sslHostsAddresses, host.GetAddressesString(true)) {
			suitableHosts = append(suitableHosts, host)
		}
	}

	return suitableHosts, nil
}

func (m *NginxManager) convertNginxHostsToWebserverHosts(nginxHosts []parser.NginxHost) []webserver.Host {
	var hosts []webserver.Host

	for _, nHost := range nginxHosts {
		hosts = append(hosts, nHost.Host)
	}

	return hosts
}

func (m *NginxManager) getIpv6Info(port string) (ipv6Info, error) {
	var info ipv6Info

	hosts, err := m.parser.GetHosts()
	if err != nil {
		return info, err
	}

	for _, host := range hosts {
		for _, address := range host.Addresses {
			if address.IsIpv6 {
				info.isIpv6Active = true
			}

			if host.IsIpv6Only() && address.Port == port {
				info.isIpv6OnlyPresent = true
			}
		}
	}

	return info, nil
}

func GetNginxManager(params map[string]string, logger logger.LoggerInterface) (*NginxManager, error) {
	options := nginxoptions.GetOptions(params)

	nginxCli, err := nginxcli.GetNginxCli()
	if err != nil {
		return nil, err
	}

	serverRootDirectory := options.Get(nginxoptions.ServerRoot)
	parser, err := parser.GetParser(serverRootDirectory, logger)
	if err != nil {
		return nil, err
	}

	enabledHostConfigDirectory, err := getEnabledHostConfigDirectory(serverRootDirectory)
	if err != nil {
		return nil, err
	}

	defaultHostManager, err := hostmanager.GetHostManager(enabledHostConfigDirectory)
	if err != nil {
		return nil, err
	}

	manager := NginxManager{
		nginxCli: nginxCli,
		parser:   parser,
		logger:   logger,
		options:  options,
		reverter: reverter.GetConfigReveter(defaultHostManager, logger),
	}

	return &manager, nil
}

func getEnabledHostConfigDirectory(serverRootDir string) (string, error) {
	for _, dirName := range enabledHostConfigDirNames {
		enabledHostDir := filepath.Join(serverRootDir, dirName)
		if com.IsDir(enabledHostDir) {
			return enabledHostDir, nil
		}
	}

	return "", errors.New("unable to find enabled hosts configuration directory")
}
