package apache

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/stretchr/testify/assert"
)

const (
	apacheDir = "../../test/apache/integration"
)

func TestGetHosts(t *testing.T) {
	webServerManager := getWebServerManager(t)
	hosts := webServerManager.getHosts()

	expectedHosts := getHostsFromJSON(t)
	for _, host := range hosts {
		expectedHost, ok := expectedHosts[host.AugPath]
		assert.Equal(t, true, ok, "could not find host in map")
		assert.Equal(t, expectedHost, host, "invalid host")
	}
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
	webServerManager, err := GetApacheManager(nil, logger.NilLogger{})
	assert.Nil(t, err, fmt.Sprintf("could not create apache webserver manager: %v", err))

	return webServerManager
}

func getHostsFromJSON(t *testing.T) map[string]*apacheHost {
	hostsPath := apacheDir + "/hosts.json"
	assert.FileExists(t, hostsPath, "could not open hosts file")
	data, err := os.ReadFile(hostsPath)
	assert.Nilf(t, err, "could not read hosts file: %v", err)

	var hosts []*apacheHost
	err = json.Unmarshal(data, &hosts)
	assert.Nilf(t, err, "could not decode hosts: %v", err)

	hostsMap := make(map[string]*apacheHost)
	for _, host := range hosts {
		hostsMap[host.AugPath] = host
	}

	return hostsMap
}
