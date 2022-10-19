package apache

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	apacheDir = "../../test/apache"
)

func TestGetHosts(t *testing.T) {
	webServerManager := getWebServerManager(t)
	hosts, err := webServerManager.GetHosts()
	assert.Nilf(t, err, "could not get hosts: %v", err)

	hostsJSON, err := json.Marshal(hosts)
	assert.Nilf(t, err, "could not marshal hosts: %v", err)
	expectedHostsJSON := getHostsJSON(t)
	assert.Equal(t, expectedHostsJSON, string(hostsJSON), "invalid hosts")
}

func TestGetHostNames(t *testing.T) {
	webServerManager := getWebServerManager(t)
	hostNames, err := webServerManager.getHostNames("/files/etc/apache2/sites-enabled/example2.com.conf/VirtualHost")
	assert.Nilf(t, err, "could not get host names: %v", err)
	assert.Equal(t, "example2.com", hostNames.ServerName)
	assert.Equal(t, 1, len(hostNames.ServerAliases))
	assert.Equal(t, "www.example2.com", hostNames.ServerAliases[0])
}

func TestGetDocumentRoot(t *testing.T) {
	webServerManager := getWebServerManager(t)
	docRoot, err := webServerManager.getDocumentRoot("/files/etc/apache2/sites-enabled/example2.com.conf/VirtualHost")
	assert.Nilf(t, err, "could not get document root: %v", err)
	assert.Equal(t, "/var/www/html", docRoot)
}

func getWebServerManager(t *testing.T) *ApacheManager {
	webServerManager, err := GetApacheManager(nil)
	assert.Nil(t, err, fmt.Sprintf("could not create apache webserver manager: %v", err))

	return webServerManager
}

func getHostsJSON(t *testing.T) string {
	hostsPath := apacheDir + "/hosts.json"
	assert.FileExists(t, hostsPath, "could not open hosts file")
	data, err := os.ReadFile(hostsPath)
	assert.Nilf(t, err, "could not read hosts file: %v", err)

	return prepareStringToCompare(string(data))
}

func prepareStringToCompare(str string) string {
	re := regexp.MustCompile(`[\r\n\s]`)
	return re.ReplaceAllString(string(str), "")
}
