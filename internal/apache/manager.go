package apache

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/r2dtools/webmng/internal/apache/apachectl"
	"github.com/r2dtools/webmng/internal/apache/apachesite"
	apacheoptions "github.com/r2dtools/webmng/internal/apache/options"
	"github.com/r2dtools/webmng/pkg/aug"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/utils"
	"github.com/r2dtools/webmng/pkg/webserver"
	"github.com/r2dtools/webmng/pkg/webserver/host"
)

const (
	minApacheVersion = "2.4.0"
)

var serverRootPaths = []string{"/etc/httpd", "/etc/apache2"}
var configFiles = []string{"apache2.conf", "httpd.conf", "conf/httpd.conf"}

type ApacheManager struct {
	apachectl     *apachectl.ApacheCtl
	apachesite    *apachesite.ApacheSite
	parser        *Parser
	logger        logger.LoggerInterface
	apacheVersion string
	hosts         []*apacheHost
}

type apacheHost struct {
	webserver.Host

	AugPath string
}

type hsotNames struct {
	ServerName    string
	ServerAliases []string
}

func (m *ApacheManager) GetHosts() ([]*webserver.Host, error) {
	if m.hosts == nil {
		m.hosts = m.getHosts()
	}

	var hosts []*webserver.Host

	for _, host := range m.hosts {
		hosts = append(hosts, &host.Host)
	}

	return hosts, nil
}

// CheckConfiguration checks if apache configuration is correct
func (m *ApacheManager) CheckConfiguration() bool {
	if err := m.apachectl.TestConfiguration(); err != nil {
		return false
	}

	return true
}

// RestartWebServer restarts apache web server
func (m *ApacheManager) Restart() error {
	return m.apachectl.Restart()
}

func (m *ApacheManager) getHosts() []*apacheHost {
	filePaths := make(map[string]string)
	internalPaths := make(map[string]map[string]bool)
	var hosts []*apacheHost

	for hostPath := range m.parser.LoadedPaths {
		paths, err := m.parser.Augeas.Match(fmt.Sprintf("/files%s//*[label()=~regexp('VirtualHost', 'i')]", hostPath))
		if err != nil {
			continue
		}

		for _, path := range paths {
			if !strings.Contains(strings.ToLower(path), "virtualhost") {
				continue
			}

			host, err := m.createHost(path)
			if err != nil {
				m.logger.Error("error occured while creating host '%s': %v", host.FilePath, err)
				continue
			}

			internalPath := aug.GetInternalAugPath(host.AugPath)
			realPath, err := filepath.EvalSymlinks(host.FilePath)

			if _, ok := internalPaths[realPath]; !ok {
				internalPaths[realPath] = make(map[string]bool)
			}

			if err != nil {
				m.logger.Error(fmt.Sprintf("failed to eval symlinks for host '%s': %v", host.FilePath, err))
				continue
			}

			if _, ok := filePaths[realPath]; !ok {
				filePaths[realPath] = host.FilePath

				if iPaths, ok := internalPaths[realPath]; !ok {
					internalPaths[realPath] = map[string]bool{
						internalPath: true,
					}
				} else {
					if _, ok = iPaths[internalPath]; !ok {
						iPaths[internalPath] = true
					}
				}

				hosts = append(hosts, host)
			} else if realPath == host.FilePath && realPath != filePaths[realPath] {
				// Prefer "real" host paths instead of symlinked ones
				// for example: sites-enabled/vh.conf -> sites-available/vh.conf
				// remove old (most likely) symlinked one
				var nHosts []*apacheHost

				for _, h := range hosts {
					if h.FilePath == filePaths[realPath] {
						delete(internalPaths[realPath], aug.GetFilePathFromAugPath(h.AugPath))
					} else {
						nHosts = append(nHosts, h)
					}
				}

				hosts = nHosts
				filePaths[realPath] = realPath
				internalPaths[realPath][internalPath] = true
				hosts = append(hosts, host)

			} else if _, ok = internalPaths[realPath][internalPath]; !ok {
				internalPaths[realPath][internalPath] = true
				hosts = append(hosts, host)
			}
		}
	}

	return hosts
}

func (m *ApacheManager) createHost(path string) (*apacheHost, error) {
	args, err := m.parser.Augeas.Match(fmt.Sprintf("%s/arg", path))
	if err != nil {
		return nil, err
	}

	addrs := make(map[string]host.Address)
	for _, arg := range args {
		arg, err = m.parser.GetArg(arg)
		if err != nil {
			return nil, err
		}

		addr := host.CreateHostAddressFromString(arg)
		addrs[addr.GetHash()] = addr
	}

	var ssl bool
	sslDirectiveMatches, err := m.parser.FindDirective("SslEngine", "on", path, false)
	if err != nil {
		return nil, err
	}

	if len(sslDirectiveMatches) > 0 {
		ssl = true
	}

	for _, addr := range addrs {
		if addr.Port == "443" {
			ssl = true
			break
		}
	}

	fPath, err := m.parser.Augeas.Get(fmt.Sprintf("/augeas/files%s/path", aug.GetFilePathFromAugPath(path)))
	if err != nil {
		return nil, err
	}

	filename := aug.GetFilePathFromAugPath(fPath)
	if filename == "" {
		return nil, nil
	}

	var macro bool
	if strings.Contains(strings.ToLower(path), "/macro/") {
		macro = true
	}

	hostEnabled := m.parser.IsFilenameExistInOriginalPaths(filename)
	docRoot, err := m.getDocumentRoot(path)
	if err != nil {
		return nil, err
	}

	host := &apacheHost{
		Host: webserver.Host{
			FilePath:  filename,
			DocRoot:   docRoot,
			Ssl:       ssl,
			ModMacro:  macro,
			Enabled:   hostEnabled,
			Addresses: addrs,
		},
		AugPath: path,
	}
	m.addServerNames(host)

	return host, err
}

func (m *ApacheManager) addServerNames(host *apacheHost) error {
	hostNames, err := m.getHostNames(host.AugPath)
	if err != nil {
		return err
	}

	for _, alias := range hostNames.ServerAliases {
		if !host.ModMacro {
			host.Aliases = append(host.Aliases, alias)
		}
	}

	if !host.ModMacro {
		host.ServerName = hostNames.ServerName
	}

	return nil
}

func (m *ApacheManager) getHostNames(path string) (*hsotNames, error) {
	serverNameMatch, err := m.parser.FindDirective("ServerName", "", path, false)
	if err != nil {
		return nil, fmt.Errorf("failed searching ServerName directive: %v", err)
	}

	serverAliasMatch, err := m.parser.FindDirective("ServerAlias", "", path, false)
	if err != nil {
		return nil, fmt.Errorf("failed searching ServerAlias directive: %v", err)
	}

	var serverAliases []string
	var serverName string

	for _, alias := range serverAliasMatch {
		serverAlias, err := m.parser.GetArg(alias)
		if err != nil {
			return nil, err
		}

		serverAliases = append(serverAliases, serverAlias)
	}

	if len(serverNameMatch) > 0 {
		serverName, err = m.parser.GetArg(serverNameMatch[len(serverNameMatch)-1])
		if err != nil {
			return nil, err
		}
	}

	return &hsotNames{serverName, serverAliases}, nil
}

func (m *ApacheManager) getDocumentRoot(path string) (string, error) {
	var docRoot string
	docRootMatch, err := m.parser.FindDirective("DocumentRoot", "", path, false)
	if err != nil {
		return "", fmt.Errorf("could not get host document root: %v", err)
	}

	if len(docRootMatch) > 0 {
		docRoot, err = m.parser.GetArg(docRootMatch[len(docRootMatch)-1])
		if err != nil {
			return "", fmt.Errorf("could not get host document root: %v", err)
		}

		//  If the directory-path is not absolute then it is assumed to be relative to the ServerRoot.
		if !strings.HasPrefix(docRoot, string(filepath.Separator)) {
			docRoot = filepath.Join(m.parser.ServerRoot, docRoot)
		}
	}

	return docRoot, nil
}

func GetApacheManager(params map[string]string) (*ApacheManager, error) {
	options := apacheoptions.GetOptions(params)

	aCtl, err := apachectl.GetApacheCtl(options.Get(apacheoptions.ApacheCtl))
	if err != nil {
		return nil, err
	}

	aSite := apachesite.GetApacheSite(options.Get(apacheoptions.ApacheEnsite), options.Get(apacheoptions.ApacheDissite))
	parser, err := GetParser(
		aCtl,
		"Httpd",
		options.Get(apacheoptions.ServerRoot),
		options.Get(apacheoptions.HostRoot),
		options.Get(apacheoptions.HostFiles),
	)
	if err != nil {
		return nil, err
	}

	version, err := aCtl.GetVersion()
	if err != nil {
		return nil, err
	}

	isVersionSupported, err := utils.CheckMinVersion(version, minApacheVersion)
	if err != nil {
		return nil, err
	}

	if !isVersionSupported {
		return nil, fmt.Errorf("current apache version '%s' is not supported. Minimal supported version is '%s'", version, minApacheVersion)
	}

	// Test apache configuration before creating manager
	if err = aCtl.TestConfiguration(); err != nil {
		return nil, err
	}

	manager := ApacheManager{
		apachectl:     aCtl,
		apachesite:    aSite,
		parser:        parser,
		logger:        logger.NilLogger{},
		apacheVersion: version,
	}

	return &manager, nil
}

func (m *ApacheManager) SetLogger(logger logger.LoggerInterface) {
	m.logger = logger
}
