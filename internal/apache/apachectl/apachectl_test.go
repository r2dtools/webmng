package apachectl

import (
	"regexp"
	"testing"

	"github.com/r2dtools/webmng/pkg/utils"
	"github.com/stretchr/testify/assert"
)

var rhel bool

func init() {
	var err error
	rhel, err = utils.IsRhelOsFamily()
	if err != nil {
		panic(err)
	}
}

func TestGetVersion(t *testing.T) {
	version, err := getApacheCtl(t).GetVersion()
	assert.Nil(t, err)

	reg, err := regexp.Compile(`^\d+\.\d+\.\d+$`)
	assert.Nilf(t, err, "invalid regex: %v", err)
	assert.Equal(t, true, reg.MatchString(version), "invalid version: %s", version)
}

func TestRestart(t *testing.T) {
	assert.Nil(t, getApacheCtl(t).Restart())
}

func TestTestConfiguration(t *testing.T) {
	assert.Nil(t, getApacheCtl(t).TestConfiguration())
}

func TestParseModules(t *testing.T) {
	modules, err := getApacheCtl(t).ParseModules()
	assert.Nil(t, err)
	assert.Contains(t, modules, "core")
}

func TestParseIncludes(t *testing.T) {
	includes, err := getApacheCtl(t).ParseIncludes()
	assert.Nil(t, err)
	expectedInclude := "/etc/apache2/sites-enabled/example-ssl.com.conf"
	if rhel {
		expectedInclude = "/etc/httpd/conf.d/example-ssl.com.conf"
	}
	assert.Contains(t, includes, expectedInclude)
}

func TestParseDefines(t *testing.T) {
	_, err := getApacheCtl(t).ParseDefines()
	assert.Nil(t, err)
}

func getApacheCtl(t *testing.T) ApacheCtl {
	apacheCtl, err := GetApacheCtl()
	assert.Nilf(t, err, "failed to create apachectl: %v", err)

	return apacheCtl
}
