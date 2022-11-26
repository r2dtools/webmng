package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	nginxoptions "github.com/r2dtools/webmng/internal/nginx/options"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/stretchr/testify/assert"
)

const nginxDir = "../../../test/nginx/integration"

func TestGetHosts(t *testing.T) {
	nginxParser := getNginxParser(t)
	hosts, err := nginxParser.GetHosts()
	assert.Nilf(t, err, "could not get hosts: %v", err)

	expectedHosts := getHostsFromJSON(t)
	assert.Equal(t, len(expectedHosts), len(hosts))

	for _, host := range hosts {
		expectedHost, ok := expectedHosts[host.Offset]
		assert.Equal(t, true, ok, "could not find host in map")
		assert.Equal(t, expectedHost, host, "invalid host")
	}
}

func getNginxParser(t *testing.T) *Parser {
	options := nginxoptions.GetOptions(nil)
	parser, err := GetParser(options.Get(nginxoptions.ServerRoot), logger.NilLogger{})
	assert.Nil(t, err, fmt.Sprintf("could not create nginx parser: %v", err))

	return parser
}

func getHostsFromJSON(t *testing.T) map[int]nginxHost {
	hostsPath := nginxDir + "/hosts.json"
	assert.FileExists(t, hostsPath, "could not open hosts file")
	data, err := os.ReadFile(hostsPath)
	assert.Nilf(t, err, "could not read hosts file: %v", err)

	var hosts []nginxHost
	err = json.Unmarshal(data, &hosts)
	assert.Nilf(t, err, "could not decode hosts: %v", err)

	hostsMap := make(map[int]nginxHost)
	for _, host := range hosts {
		hostsMap[host.Offset] = host
	}

	return hostsMap
}
