package nginxcli

import (
	"regexp"
	"testing"

	nginxoptions "github.com/r2dtools/webmng/internal/nginx/options"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration(t *testing.T) {
	cli := getNginxCli(t)
	err := cli.TestConfiguration()
	assert.Nilf(t, err, "error while testing nginx configuration: %v", err)
}

func TestRestart(t *testing.T) {
	cli := getNginxCli(t)
	err := cli.Restart()
	assert.Nilf(t, err, "error while restarting nginx: %v", err)
}

func TestGetVersion(t *testing.T) {
	cli := getNginxCli(t)
	version, err := cli.GetVersion()
	assert.Nilf(t, err, "could not get nginx version: %v", err)

	reg, err := regexp.Compile(`^\d+\.\d+\.\d+$`)
	assert.Nilf(t, err, "invalid regex: %v", err)
	assert.Equal(t, true, reg.MatchString(version), "invalid version: %s", version)
}

func getNginxCli(t *testing.T) *NginxCli {
	options := nginxoptions.GetOptions(nil)
	cli, err := GetNginxCli(options.Get(nginxoptions.NginxBinPath))
	assert.Nilf(t, err, "could not create nginx cli: %v", err)

	return cli
}
