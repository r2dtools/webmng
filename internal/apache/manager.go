package apache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/r2dtools/webmng/internal/apache/apachectl"
	"github.com/r2dtools/webmng/internal/apache/apachesite"
	apacheoptions "github.com/r2dtools/webmng/internal/apache/options"
	"github.com/r2dtools/webmng/internal/apache/parser"
	apacheutils "github.com/r2dtools/webmng/internal/apache/utils"
	"github.com/r2dtools/webmng/pkg/aug"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/options"
	"github.com/r2dtools/webmng/pkg/utils"
	"github.com/r2dtools/webmng/pkg/webserver"
	"github.com/r2dtools/webmng/pkg/webserver/host"
	"github.com/r2dtools/webmng/pkg/webserver/reverter"
	"github.com/unknwon/com"
	"golang.org/x/exp/slices"
)

const (
	minApacheVersion = "2.4.0"
)

type ApacheManager struct {
	apachectl     apachectl.ApacheCtl
	apachesite    *apachesite.ApacheSite
	parser        *parser.Parser
	logger        logger.LoggerInterface
	apacheVersion string
	apacheHosts   []apacheHost
	reverter      reverter.Reverter
	options       options.Options
}

type apacheHost struct {
	webserver.Host

	AugPath  string
	ModMacro bool
}

type hsotNames struct {
	ServerName    string
	ServerAliases []string
}

func (m *ApacheManager) GetHosts() ([]webserver.Host, error) {
	if m.apacheHosts == nil {
		m.apacheHosts = m.getApacheHosts()
	}

	return m.convertApacheHostsToWebserverHosts(m.apacheHosts), nil
}

func (m *ApacheManager) GetVersion() (string, error) {
	return m.apacheVersion, nil
}

// Commit applies all current changes
func (m *ApacheManager) CommitChanges() error {
	return m.reverter.Commit()
}

// CheckConfiguration checks if apache configuration is correct
func (m *ApacheManager) CheckConfiguration() error {
	return m.apachectl.TestConfiguration()
}

// RestartWebServer restarts apache web server
func (m *ApacheManager) Restart() error {
	return m.apachectl.Restart()
}

func (m *ApacheManager) EnableHost(host *webserver.Host) error {
	if host.Enabled {
		m.logger.Debug(fmt.Sprintf("host '%s' is already enabled. Skip site enabling.", host.FilePath))
		return nil
	}

	// First, try to enable host via a2ensite utility
	err := m.apachesite.Enable(host.FilePath)

	if err == nil {
		m.reverter.AddHostConfigToDisable(host.FilePath)
		host.Enabled = true
		return nil
	} else {
		m.logger.Debug(err.Error())
	}

	// If host could not be enabled via a2ensite, than try to enable it via Include directive in apache config
	if !m.parser.IsFilenameExistInOriginalPaths(host.FilePath) {
		m.logger.Debug(fmt.Sprintf("try to enable host '%s' via 'include' directive.", host.FilePath))
		if err := m.parser.AddInclude(m.parser.ConfigRoot, host.FilePath); err != nil {
			return fmt.Errorf("could not enable host '%s': %v", host.FilePath, err)
		}

		host.Enabled = true
	}

	return nil
}

func (m *ApacheManager) DeployCertificate(serverName, certPath, certKeyPath, chainPath, fullChainPath string) error {
	aHosts, err := m.getApacheHostsByServerNameWithRequiredSSLConfigPart(serverName)

	if err != nil {
		return err
	}

	if len(aHosts) == 0 {
		return fmt.Errorf("could not find suitable hosts with serverName: %s", serverName)
	}

	if err = m.prepareServerForHTTPS("443", false); err != nil {
		return err
	}

	for _, aHost := range aHosts {
		if err = m.cleanSSLApacheHost(aHost); err != nil {
			return err
		}

		if err = m.addDummySSLDirectives(aHost.AugPath); err != nil {
			return err
		}

		augCertPath, err := m.parser.FindDirective("SSLCertificateFile", "", aHost.AugPath, true)

		if err != nil {
			return fmt.Errorf("error while searching directive 'SSLCertificateFile': %v", err)
		}

		augCertKeyPath, err := m.parser.FindDirective("SSLCertificateKeyFile", "", aHost.AugPath, true)

		if err != nil {
			return fmt.Errorf("error while searching directive 'SSLCertificateKeyFile': %v", err)
		}

		res, err := utils.CheckMinVersion(m.apacheVersion, "2.4.8")

		if err != nil {
			return err
		}

		if !res || (chainPath != "" && fullChainPath == "") {
			if err = m.parser.Augeas.Set(augCertPath[len(augCertPath)-1], certPath); err != nil {
				return fmt.Errorf("could not set certificate path for vhost '%s': %v", serverName, err)
			}

			if err = m.parser.Augeas.Set(augCertKeyPath[len(augCertKeyPath)-1], certKeyPath); err != nil {
				return fmt.Errorf("could not set certificate key path for vhost '%s': %v", serverName, err)
			}

			if chainPath != "" {
				if err = m.parser.AddDirective(aHost.AugPath, "SSLCertificateChainFile", []string{chainPath}); err != nil {
					return fmt.Errorf("could not add 'SSLCertificateChainFile' directive to vhost '%s': %v", serverName, err)
				}
			} else {
				return fmt.Errorf("SSL certificate chain path is required for the current Apache version '%s', but is not specified", m.apacheVersion)
			}
		} else {
			if fullChainPath == "" {
				return errors.New("SSL certificate fullchain path is required, but is not specified")
			}

			if err = m.parser.Augeas.Set(augCertPath[len(augCertPath)-1], fullChainPath); err != nil {
				return fmt.Errorf("could not set certificate path for vhost '%s': %v", serverName, err)
			}
			if err = m.parser.Augeas.Set(augCertKeyPath[len(augCertKeyPath)-1], certKeyPath); err != nil {
				return fmt.Errorf("could not set certificate key path for vhost '%s': %v", serverName, err)
			}
		}

		if !aHost.Enabled {
			if err = m.EnableHost(&aHost.Host); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *ApacheManager) GetHostsByServerName(serverName string) ([]webserver.Host, error) {
	aHosts, err := m.getApacheHostsByServerName(serverName)

	if err != nil {
		return nil, err
	}

	return m.convertApacheHostsToWebserverHosts(aHosts), nil
}

func (m *ApacheManager) Save() error {
	if err := m.parser.Save(m.reverter); err != nil {
		return fmt.Errorf("could not save changes: %v", err)
	}

	return nil
}

func (m *ApacheManager) getApacheHostsByServerName(serverName string) ([]apacheHost, error) {
	aHosts := m.getApacheHosts()

	var suitableHosts []apacheHost
	var suitableNonSslHosts []apacheHost
	var sslHostsAddresses []string

	for _, aHost := range aHosts {
		if aHost.ModMacro {
			m.logger.Warning(fmt.Sprintf("host '%s' has mod macro enabled. Skip it.", aHost.FilePath))
			continue
		}

		// Prefer host with ssl
		if aHost.ServerName == serverName {
			if aHost.Ssl {
				suitableHosts = append(suitableHosts, aHost)
				sslHostsAddresses = append(sslHostsAddresses, aHost.GetAddressesString(true))
			} else {
				suitableNonSslHosts = append(suitableNonSslHosts, aHost)
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

// If createIfNoSsl is true then ssl part will be created if neccessary.
func (m *ApacheManager) getApacheHostsByServerNameWithRequiredSSLConfigPart(serverName string) ([]apacheHost, error) {
	aHosts, err := m.getApacheHostsByServerName(serverName)

	if err != nil {
		return nil, err
	}

	return m.makeSslHosts(aHosts)
}

// makeSslVhosts makes an ssl host version of a nonssl host
func (m *ApacheManager) makeSslHosts(hosts []apacheHost) ([]apacheHost, error) {
	var totalHosts []apacheHost
	var newApacheSslHosts []apacheHost
	var newMatches []string

	for _, host := range hosts {
		if host.Ssl {
			totalHosts = append(totalHosts, host)
			continue
		}

		noSslFilePath := host.FilePath
		sslFilePath, err := m.getSslHostFilePath(noSslFilePath)

		if err != nil {
			return nil, fmt.Errorf("could not get config file path for ssl virtual host: %v", err)
		}

		originMatches, err := m.parser.Augeas.Match(fmt.Sprintf("/files%s//*[label()=~regexp('VirtualHost', 'i')]", apacheutils.Escape(sslFilePath)))

		if err != nil {
			return nil, err
		}

		if err = m.copyCreateSslHostSkeleton(host, sslFilePath); err != nil {
			return nil, fmt.Errorf("could not create config for ssl virtual host: %v", err)
		}

		// Reload augeas to take into account the new vhost
		m.parser.Augeas.Load()
		newMatches, err = m.parser.Augeas.Match(fmt.Sprintf("/files%s//*[label()=~regexp('VirtualHost', 'i')]", apacheutils.Escape(sslFilePath)))

		if err != nil {
			return nil, err
		}

		sslHostPath := m.getNewHostPathFromAugesMatches(originMatches, newMatches)

		if sslHostPath == "" {
			newMatches, err = m.parser.Augeas.Match(fmt.Sprintf("/files%s//*[label()=~regexp('VirtualHost', 'i')]", apacheutils.Escape(sslFilePath)))

			if err != nil {
				return nil, err
			}

			sslHostPath = m.getNewHostPathFromAugesMatches(originMatches, newMatches)

			if sslHostPath == "" {
				return nil, errors.New("could not reverse map the HTTPS VirtualHost to the original")
			}
		}

		m.updateSslHostAddresses(sslHostPath)

		if err := m.Save(); err != nil {
			return nil, err
		}

		apacheSslHost, err := m.createApacheHost(sslHostPath)

		if err != nil {
			return nil, err
		}

		newApacheSslHosts = append(newApacheSslHosts, apacheSslHost)
	}

	m.updateHostsAugPath(newApacheSslHosts, newMatches)
	totalHosts = append(totalHosts, newApacheSslHosts...)

	return totalHosts, nil
}

func (m *ApacheManager) copyCreateSslHostSkeleton(noSslHost apacheHost, sslHostFilePath string) error {
	_, err := os.Stat(sslHostFilePath)

	if os.IsNotExist(err) {
		m.reverter.AddFileToDeletion(sslHostFilePath)
	} else if err == nil {
		m.reverter.BackupFile(sslHostFilePath)
	} else {
		return err
	}

	noSslHostContents, err := m.getApacheHostBlockContent(noSslHost)

	if err != nil {
		return err
	}

	sslHostContent, _ := apacheutils.DisableDangerousForSslRewriteRules(noSslHostContents)
	sslVhostFile, err := os.OpenFile(sslHostFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	if err != nil {
		return err
	}

	defer sslVhostFile.Close()
	sslContent := []string{
		"<IfModule mod_ssl.c>\n",
		strings.Join(sslHostContent, "\n"),
		"</VirtualHost>\n",
		"</IfModule>\n",
	}

	for _, line := range sslContent {
		_, err = sslVhostFile.WriteString(line)

		if err != nil {
			return fmt.Errorf("could not write to ssl virtual host file '%s': %v", sslHostFilePath, err)
		}
	}

	if !m.parser.IsFilenameExistInLoadedPaths(sslHostFilePath) {
		err = m.parser.ParseFile(sslHostFilePath)

		if err != nil {
			return fmt.Errorf("could not parse ssl virtual host file '%s': %v", sslHostFilePath, err)
		}
	}

	m.parser.Augeas.Set(fmt.Sprintf("/augeas/files%s/mtime", apacheutils.Escape(sslHostFilePath)), "0")
	m.parser.Augeas.Set(fmt.Sprintf("/augeas/files%s/mtime", apacheutils.Escape(noSslHost.FilePath)), "0")

	return nil
}

func (m *ApacheManager) getApacheHostBlockContent(host apacheHost) ([]string, error) {
	span, err := m.parser.Augeas.Span(host.AugPath)

	if err != nil {
		return nil, fmt.Errorf("could not get VirtualHost '%s' from the file %s: %v", host.ServerName, host.FilePath, err)
	}

	file, err := os.Open(span.Filename)

	if err != nil {
		return nil, err
	}

	defer file.Close()
	_, err = file.Seek(int64(span.SpanStart), 0)

	if err != nil {
		return nil, err
	}

	bContent := make([]byte, span.SpanEnd-span.SpanStart)
	_, err = file.Read(bContent)

	if err != nil {
		return nil, err
	}

	content := string(bContent)
	lines := strings.Split(content, "\n")
	apacheutils.RemoveClosingHostTag(lines)

	return lines, nil
}

func (m *ApacheManager) getSslHostFilePath(noSslHostFilePath string) (string, error) {
	var filePath string
	var err error

	hostRoot := m.options.Get(apacheoptions.HostRoot)

	if hostRoot != "" {
		_, err = os.Stat(hostRoot)

		if err == nil {
			eVhostRoot, err := filepath.EvalSymlinks(hostRoot)

			if err != nil {
				return "", err
			}

			filePath = filepath.Join(eVhostRoot, filepath.Base(noSslHostFilePath))
		}
	} else {
		filePath, err = filepath.EvalSymlinks(noSslHostFilePath)

		if err != nil {
			return "", err
		}
	}

	sslHostExt := m.options.Get(apacheoptions.SslVhostlExt)

	if strings.HasSuffix(filePath, ".conf") {
		return filePath[:len(filePath)-len("conf.")] + sslHostExt, nil
	}

	return filePath + sslHostExt, nil
}

func (m *ApacheManager) updateSslHostAddresses(sslVhostPath string) ([]host.Address, error) {
	var sslAddresses []host.Address
	sslAddrMatches, err := m.parser.Augeas.Match(sslVhostPath + "/arg")

	if err != nil {
		return nil, err
	}

	for _, sslAddrMatch := range sslAddrMatches {
		addrString, err := m.parser.GetArg(sslAddrMatch)

		if err != nil {
			return nil, err
		}

		oldAddress := host.CreateHostAddressFromString(addrString)
		sslAddress := oldAddress.GetAddressWithNewPort("443") // TODO: it should be passed in an external code
		err = m.parser.Augeas.Set(sslAddrMatch, sslAddress.ToString())

		if err != nil {
			return nil, err
		}

		var exists bool

		for _, addr := range sslAddresses {
			if sslAddress.IsEqual(addr) {
				exists = true
				break
			}
		}

		if !exists {
			sslAddresses = append(sslAddresses, sslAddress)
		}
	}

	return sslAddresses, nil
}

func (m *ApacheManager) getNewHostPathFromAugesMatches(originMatches []string, newMatches []string) string {
	var mOriginMatches []string

	for _, originMatch := range originMatches {
		mOriginMatches = append(mOriginMatches, strings.Replace(originMatch, "[1]", "", -1))
	}

	for _, newMatch := range newMatches {
		mNewMatch := strings.Replace(newMatch, "[1]", "", -1)

		if !com.IsSliceContainsStr(mOriginMatches, mNewMatch) {
			return newMatch
		}
	}

	return ""
}

func (m *ApacheManager) updateHostsAugPath(apacheHosts []apacheHost, newMatches []string) {
	for _, newMatch := range newMatches {
		mNewMatch := strings.Replace(newMatch, "[1]", "", -1)

		for _, apacheHost := range apacheHosts {
			if apacheHost.AugPath == mNewMatch {
				apacheHost.AugPath = newMatch
			}
		}
	}
}

func (m *ApacheManager) getApacheHosts() []apacheHost {
	filePaths := make(map[string]string)
	internalPaths := make(map[string]map[string]bool)
	var apacheHosts []apacheHost

	for hostPath := range m.parser.LoadedPaths {
		paths, err := m.parser.Augeas.Match(fmt.Sprintf("/files%s//*[label()=~regexp('VirtualHost', 'i')]", hostPath))

		if err != nil {
			continue
		}

		for _, path := range paths {
			if !strings.Contains(strings.ToLower(path), "virtualhost") {
				continue
			}

			host, err := m.createApacheHost(path)

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

				apacheHosts = append(apacheHosts, host)
			} else if realPath == host.FilePath && realPath != filePaths[realPath] {
				// Prefer "real" host paths instead of symlinked ones
				// for example: sites-enabled/vh.conf -> sites-available/vh.conf
				// remove old (most likely) symlinked one
				var nApacheHosts []apacheHost

				for _, h := range apacheHosts {
					if h.FilePath == filePaths[realPath] {
						delete(internalPaths[realPath], aug.GetFilePathFromAugPath(h.AugPath))
					} else {
						nApacheHosts = append(nApacheHosts, h)
					}
				}

				apacheHosts = nApacheHosts
				filePaths[realPath] = realPath
				internalPaths[realPath][internalPath] = true
				apacheHosts = append(apacheHosts, host)

			} else if _, ok = internalPaths[realPath][internalPath]; !ok {
				internalPaths[realPath][internalPath] = true
				apacheHosts = append(apacheHosts, host)
			}
		}
	}

	return apacheHosts
}

func (m *ApacheManager) createApacheHost(path string) (apacheHost, error) {
	var aHost apacheHost
	args, err := m.parser.Augeas.Match(fmt.Sprintf("%s/arg", path))

	if err != nil {
		return aHost, err
	}

	addrs := make(map[string]host.Address)
	for _, arg := range args {
		arg, err = m.parser.GetArg(arg)

		if err != nil {
			return aHost, err
		}

		addr := host.CreateHostAddressFromString(arg)
		addrs[addr.GetHash()] = addr
	}

	var ssl bool
	sslDirectiveMatches, err := m.parser.FindDirective("SslEngine", "on", path, false)

	if err != nil {
		return aHost, err
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
		return aHost, err
	}

	filename := aug.GetFilePathFromAugPath(fPath)
	if filename == "" {
		return aHost, nil
	}

	var macro bool
	if strings.Contains(strings.ToLower(path), "/macro/") {
		macro = true
	}

	hostEnabled := m.parser.IsFilenameExistInOriginalPaths(filename)
	docRoot, err := m.getDocumentRoot(path)

	if err != nil {
		return aHost, err
	}

	aHost = apacheHost{
		Host: webserver.Host{
			FilePath:  filename,
			DocRoot:   docRoot,
			Ssl:       ssl,
			Enabled:   hostEnabled,
			Addresses: addrs,
		},
		AugPath:  path,
		ModMacro: macro,
	}
	m.addServerNames(&aHost)

	return aHost, err
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

func (m *ApacheManager) getHostNames(path string) (hsotNames, error) {
	var hNames hsotNames
	serverNameMatch, err := m.parser.FindDirective("ServerName", "", path, false)

	if err != nil {
		return hNames, fmt.Errorf("failed searching ServerName directive: %v", err)
	}

	serverAliasMatch, err := m.parser.FindDirective("ServerAlias", "", path, false)

	if err != nil {
		return hNames, fmt.Errorf("failed searching ServerAlias directive: %v", err)
	}

	var serverAliases []string
	var serverName string

	for _, alias := range serverAliasMatch {
		serverAlias, err := m.parser.GetArg(alias)

		if err != nil {
			return hNames, err
		}

		serverAliases = append(serverAliases, serverAlias)
	}

	if len(serverNameMatch) > 0 {
		serverName, err = m.parser.GetArg(serverNameMatch[len(serverNameMatch)-1])

		if err != nil {
			return hNames, err
		}
	}

	return hsotNames{serverName, serverAliases}, nil
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

func (m *ApacheManager) convertApacheHostsToWebserverHosts(apacheHosts []apacheHost) []webserver.Host {
	var hosts []webserver.Host

	for _, aHost := range apacheHosts {
		hosts = append(hosts, aHost.Host)
	}

	return hosts
}

func (m *ApacheManager) prepareServerForHTTPS(port string, temp bool) error {
	if err := m.prepareHTTPSModules(temp); err != nil {
		return err
	}

	if err := m.ensurePortIsListening(port, true); err != nil {
		return err
	}

	return nil
}

func (m *ApacheManager) prepareHTTPSModules(temp bool) error {
	if m.parser.ModuleExists("ssl_module") {
		return nil
	}

	if err := m.enableModule("ssl", temp); err != nil {
		return err
	}

	// save all changes before
	if err := m.Save(); err != nil {
		return err
	}

	if err := m.parser.Augeas.Load(); err != nil {
		return err
	}

	if err := m.parser.ResetModules(); err != nil {
		return err
	}

	return nil
}

func (m *ApacheManager) addDummySSLDirectives(hPath string) error {
	if err := m.parser.AddDirective(hPath, "SSLEngine", []string{"on"}); err != nil {
		return fmt.Errorf("could not add 'SSLEngine' directive to host %s: %v", hPath, err)
	}

	if err := m.parser.AddDirective(hPath, "SSLCertificateFile", []string{"insert_cert_file_path"}); err != nil {
		return fmt.Errorf("could not add 'SSLCertificateFile' directive to host %s: %v", hPath, err)
	}

	if err := m.parser.AddDirective(hPath, "SSLCertificateKeyFile", []string{"insert_key_file_path"}); err != nil {
		return fmt.Errorf("could not add 'SSLCertificateKeyFile' directive to host %s: %v", hPath, err)
	}

	return nil
}

func (m *ApacheManager) cleanSSLApacheHost(aHost apacheHost) error {
	if err := m.removeDirectives(aHost.AugPath, []string{"SSLEngine", "SSLCertificateFile", "SSLCertificateKeyFile", "SSLCertificateChainFile"}); err != nil {
		return err
	}

	return nil
}

func (m *ApacheManager) removeDirectives(hPath string, directives []string) error {
	for _, directive := range directives {
		directivePaths, err := m.parser.FindDirective(directive, "", hPath, false)

		if err != nil {
			return err
		}

		reg := regexp.MustCompile(`/\w*$`)

		for _, directivePath := range directivePaths {
			m.parser.Augeas.Remove(reg.ReplaceAllString(directivePath, ""))
		}
	}

	return nil
}

// EnsurePortIsListening ensures that the provided port is listening
// The port will be added to config file it is not listened
func (m *ApacheManager) ensurePortIsListening(port string, https bool) error {
	var portService string
	var listens []string
	var listenDirs []string

	if https && port != "443" {
		// https://httpd.apache.org/docs/2.4/bind.html
		// Listen 192.170.2.1:8443 https
		// running an https site on port 8443 (if protocol is not specified than 443 is used by default for https)
		portService = fmt.Sprintf("%s %s", port, "https")
	} else {
		portService = port
	}

	listenMatches, err := m.parser.FindDirective("Listen", "", "", true)

	if err != nil {
		return err
	}

	for _, lMatch := range listenMatches {
		listen, err := m.parser.GetArg(lMatch)

		if err != nil {
			return err
		}

		// listenDirs contains only unique items
		listenDirs = com.AppendStr(listenDirs, listen)
		listens = append(listens, listen)
	}

	if apacheutils.IsPortListened(listens, port) {
		m.logger.Debug(fmt.Sprintf("port %s is already listended.", port))
		return nil
	}

	if len(listens) == 0 {
		listenDirs = append(listenDirs, portService)
	}

	for _, listen := range listens {
		lParts := strings.Split(listen, ":")

		// only port is specified -> all interfaces are listened
		if len(lParts) == 1 {
			if !com.IsSliceContainsStr(listenDirs, port) && !com.IsSliceContainsStr(listenDirs, portService) {
				listenDirs = com.AppendStr(listenDirs, portService)
			}
		} else {
			lDir := fmt.Sprintf("%s:%s", apacheutils.GetIPFromListen(listen), portService)
			listenDirs = com.AppendStr(listenDirs, lDir)
		}
	}

	if https {
		return m.addListensForHTTPS(listenDirs, listens, port)
	}

	return m.addListensForHTTP(listenDirs, listens, port)
}

func (m *ApacheManager) addListensForHTTP(listens []string, listensOrigin []string, port string) error {
	newListens := utils.StrSlicesDifference(listens, listensOrigin)
	augListenPath := aug.GetAugPath(m.parser.ConfigListen)

	if com.IsSliceContainsStr(newListens, port) {
		if err := m.parser.AddDirective(augListenPath, "Listen", []string{port}); err != nil {
			return fmt.Errorf("could not add port %s to listen config: %v", port, err)
		}
	} else {
		for _, listen := range newListens {
			if err := m.parser.AddDirective(augListenPath, "Listen", strings.Split(listen, " ")); err != nil {
				return fmt.Errorf("could not add port %s to listen config: %v", port, err)
			}
		}
	}

	return nil
}

func (m *ApacheManager) addListensForHTTPS(listens []string, listensOrigin []string, port string) error {
	var portService string
	augListenPath := aug.GetAugPath(m.parser.ConfigListen)
	newListens := utils.StrSlicesDifference(listens, listensOrigin)

	if port != "443" {
		portService = fmt.Sprintf("%s %s", port, "https")
	} else {
		portService = port
	}

	if com.IsSliceContainsStr(newListens, port) || com.IsSliceContainsStr(newListens, portService) {
		if err := m.parser.AddDirectiveToIfModSSL(augListenPath, "Listen", strings.Split(portService, " ")); err != nil {
			return fmt.Errorf("could not add port %s to listen config: %v", port, err)
		}
	} else {
		for _, listen := range listens {
			if err := m.parser.AddDirectiveToIfModSSL(augListenPath, "Listen", strings.Split(listen, " ")); err != nil {
				return fmt.Errorf("could not add port %s to listen config: %v", port, err)
			}
		}
	}

	return nil
}

func (m *ApacheManager) enableModule(module string, temp bool) error {
	return fmt.Errorf("apache needs to have module %s active. please install the module manually", module)
}

func GetApacheManager(params map[string]string, logger logger.LoggerInterface) (*ApacheManager, error) {
	options := apacheoptions.GetOptions(params)
	aCtl, err := apachectl.GetApacheCtl(options.Get(apacheoptions.ApacheCtl))

	if err != nil {
		return nil, err
	}

	aSite := apachesite.GetApacheSite(options.Get(apacheoptions.ApacheEnsite), options.Get(apacheoptions.ApacheDissite))
	parser, err := parser.GetParser(
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
		logger:        logger,
		apacheVersion: version,
		options:       options,
		reverter:      reverter.GetConfigReveter(aSite, logger),
	}

	return &manager, nil
}
