package apache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/utils"
	"github.com/r2dtools/webmng/pkg/webserver/host"
	"github.com/stretchr/testify/assert"
	"github.com/unknwon/com"
)

const (
	apacheDir = "../../test/apache/integration"
)

var rhel bool

func init() {
	var err error
	rhel, err = utils.IsRhelOsFamily()
	if err != nil {
		panic(err)
	}
}

func TestGetHosts(t *testing.T) {
	webServerManager := getWebServerManager(t)
	hosts := webServerManager.getApacheHosts()

	expectedHosts := getHostsFromJSON(t)
	for _, host := range hosts {
		expectedHost, ok := expectedHosts[host.AugPath]
		assert.Equal(t, true, ok, "could not find host in map")
		assert.Equal(t, expectedHost, host, "invalid host")
	}
}

func TestGetHostNames(t *testing.T) {
	webServerManager := getWebServerManager(t)
	hostNames, err := webServerManager.getHostNames("/files" + getSitesEnabledPath() + "/example2.com.conf/VirtualHost")
	assert.Nilf(t, err, "could not get host names: %v", err)
	assert.Equal(t, "example2.com", hostNames.ServerName)
	assert.Equal(t, 1, len(hostNames.ServerAliases))
	assert.Equal(t, "www.example2.com", hostNames.ServerAliases[0])
}

func TestGetDocumentRoot(t *testing.T) {
	webServerManager := getWebServerManager(t)
	docRoot, err := webServerManager.getDocumentRoot("/files" + getSitesEnabledPath() + "/example2.com.conf/VirtualHost")
	assert.Nilf(t, err, "could not get document root: %v", err)
	assert.Equal(t, "/var/www/html", docRoot)
}

func TestEnsurePortIsListening(t *testing.T) {
	webServerManager := getWebServerManager(t)
	ports := []string{"80", "8080"}

	for _, port := range ports {
		err := webServerManager.ensurePortIsListening(port, false)
		assert.Nilf(t, err, "failed to ensure that port '%s' is listening: %v", port, err)
	}
}

func TestGetSslHostFilePath(t *testing.T) {
	webServerManager := getWebServerManager(t)
	hostPath := getSitesEnabledPath() + "/example2.com.conf"
	sslHostPath, err := webServerManager.getSslHostFilePath(hostPath)
	assert.Nilf(t, err, "could not get ssl host file path: %v", err)
	assert.Equal(t, getSitesAvailablePath()+"/example2.com-ssl.conf", sslHostPath)
}

func TestGetHostBlockContent(t *testing.T) {
	webServerManager := getWebServerManager(t)
	hosts := getHosts(t, webServerManager, "example2.com")
	content, err := webServerManager.getApacheHostBlockContent(hosts[0])
	assert.Nilf(t, err, "could not get host block content: %v", err)
	expectedContent := getHostConfigContent(t, "example2.com.conf")
	expectedContent = prepareStringToCompare(expectedContent)
	// getVhostBlockContent returns block without ending </VirtualHost>
	content = append(content, "</VirtualHost>")
	actualContent := strings.Join(content, "\n")
	actualContent = prepareStringToCompare(actualContent)

	assert.Equal(t, expectedContent, actualContent)
}

func TestGetSuitableHostsSingle(t *testing.T) {
	type hostItem struct {
		serverName, sslConfigFilePath, docRoot string
		ssl, enabled                           bool
	}

	webServerManager := getWebServerManager(t)
	hostItems := []hostItem{
		{"example2.com", getSitesAvailablePath() + "/example2.com-ssl.conf", "/var/www/html", true, false},
		{"example.com", getSitesEnabledPath() + "/example-ssl.com.conf", "/var/www/html", true, true},
	}

	for _, hostItem := range hostItems {
		sslHosts, err := webServerManager.getApacheHostsByServerNameWithRequiredSSLConfigPart(hostItem.serverName)
		assert.Nilf(t, err, "could not get ssl host: %v", err)
		assert.Equal(t, 1, len(sslHosts))
		sslHost := sslHosts[0]
		assert.Equal(t, hostItem.sslConfigFilePath, sslHost.FilePath)
		// Check that ssl config file realy exists
		assert.Equal(t, true, com.IsFile(hostItem.sslConfigFilePath))
		assert.Equal(t, hostItem.serverName, sslHost.ServerName)
		assert.Equal(t, hostItem.docRoot, sslHost.DocRoot)
		assert.Equal(t, hostItem.ssl, sslHost.Ssl)
		assert.Equal(t, hostItem.enabled, sslHost.Enabled)
		assert.Equal(t, false, sslHost.ModMacro)

		// Check that addresses are corerct for ssl host
		var addresses []host.Address
		for _, address := range sslHost.Addresses {
			addresses = append(addresses, address)
		}

		assert.Equal(t, 1, len(addresses))
		assert.Equal(t, "*:443", addresses[0].ToString())
	}
}

func TestGetSuitableHostsMultiple(t *testing.T) {
	webServerManager := getWebServerManager(t)
	sslHosts, err := webServerManager.getApacheHostsByServerNameWithRequiredSSLConfigPart("example4.com")
	assert.Nilf(t, err, "could not get ssl host: %v", err)
	assert.Equal(t, 2, len(sslHosts))
	addresses := []string{"[2002:5bcc:18fd:c:10:52:43:96]", "10.52.43.96"}

	for _, sslHost := range sslHosts {
		assert.Equal(t, getSitesEnabledPath()+"/example4-ssl.com.conf", sslHost.FilePath)
		// Check that ssl config file realy exists
		assert.Equal(t, true, com.IsFile(getSitesEnabledPath()+"/example4-ssl.com.conf"))
		assert.Equal(t, "example4.com", sslHost.ServerName)
		assert.Equal(t, "/var/www/html", sslHost.DocRoot)
		assert.Equal(t, true, sslHost.Ssl)
		assert.Equal(t, true, sslHost.Enabled)
		assert.Equal(t, false, sslHost.ModMacro)
		assert.Equal(t, 1, len(sslHost.Addresses))
		assert.Equal(t, true, com.IsSliceContainsStr(addresses, sslHost.GetAddressesString(true)))
	}
}

func TestDeployCertificate(t *testing.T) {
	webServerManager := getWebServerManager(t)
	err := webServerManager.DeployCertificate("example5.com", "/opt/webmng/test/certificate/example.com.crt", "/opt/webmng/test/certificate/example.com.key", "", "/opt/webmng/test/certificate/example.com.crt")
	assert.Nilf(t, err, "could not deploy certificate to host: %v", err)
	err = webServerManager.Save()
	assert.Nilf(t, err, "could not save changes after certificate deploy: %v", err)
	err = webServerManager.CheckConfiguration()
	assert.Nilf(t, err, "could not check configuration")
	err = webServerManager.CommitChanges()
	assert.Nilf(t, err, "could not commit changes after certificate deploy: %v", err)
	err = webServerManager.Restart()
	assert.Nilf(t, err, "could not restart webserver after certificate deploy: %v", err)
	// Check that ssl config file realy exists
	sslConfigFilePath := getSitesAvailablePath() + "/example5.com-ssl.conf"
	assert.Equal(t, true, com.IsFile(sslConfigFilePath))

	sslConfigContent, err := os.ReadFile(sslConfigFilePath)
	assert.Nilf(t, err, "could not read apache host ssl config file '%s' content: %v", sslConfigFilePath, err)
	directives := []string{"SSLCertificateKeyFile /opt/webmng/test/certificate/example.com.key", "SSLEngine on", "SSLCertificateFile /opt/webmng/test/certificate/example.com.crt"}

	for _, directive := range directives {
		assert.Containsf(t, string(sslConfigContent), directive, "ssl config does not contain directive '%s'", directive)
	}
}

func getWebServerManager(t *testing.T) *ApacheManager {
	webServerManager, err := GetApacheManager(nil, logger.NilLogger{})
	assert.Nil(t, err, fmt.Sprintf("could not create apache webserver manager: %v", err))

	return webServerManager
}

func getHosts(t *testing.T, webServerManager *ApacheManager, serverName string) []apacheHost {
	hosts, err := webServerManager.getApacheHostsByServerName(serverName)
	assert.Nilf(t, err, "could not find suitable hosts: %v", err)
	assert.NotEmptyf(t, hosts, "could not find suitable hosts for '%s' servername", serverName)

	return hosts
}

func getHostsFromJSON(t *testing.T) map[string]apacheHost {
	var hostsPath string

	if rhel {
		hostsPath = apacheDir + "/hosts-httpd.json"
	} else {
		hostsPath = apacheDir + "/hosts-apache.json"
	}

	assert.FileExists(t, hostsPath, "could not open hosts file")
	data, err := os.ReadFile(hostsPath)
	assert.Nilf(t, err, "could not read hosts file: %v", err)

	var hosts []apacheHost
	err = json.Unmarshal(data, &hosts)
	assert.Nilf(t, err, "could not decode hosts: %v", err)

	hostsMap := make(map[string]apacheHost)
	for _, host := range hosts {
		hostsMap[host.AugPath] = host
	}

	return hostsMap
}

func getHostConfigContent(t *testing.T, name string) string {
	path := filepath.Join(apacheDir, "sites-available", name)
	content, err := os.ReadFile(path)
	assert.Nilf(t, err, "could not read apache host config file '%s' content: %v", name, err)

	return string(content)
}

func prepareStringToCompare(str string) string {
	re := regexp.MustCompile(`[\r\n\s]`)
	return re.ReplaceAllString(string(str), "")
}

func getSitesEnabledPath() string {
	if rhel {
		return "/etc/httpd/conf.d"
	}

	return "/etc/apache2/sites-enabled"
}

func getSitesAvailablePath() string {
	if rhel {
		return "/etc/httpd/sites-available"
	}

	return "/etc/apache2/sites-available"
}
