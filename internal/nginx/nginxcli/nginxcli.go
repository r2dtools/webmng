package nginxcli

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

const (
	nginxCmd = "nginx"
)

type NginxCli struct {
	binPath string
}

func (n *NginxCli) Restart() error {
	if _, err := n.execCmd([]string{"-s", "reload"}); err != nil {
		return err
	}

	return nil
}

func (n *NginxCli) TestConfiguration() error {
	if _, err := n.execCmd([]string{"-t"}); err != nil {
		return err
	}

	return nil
}

func (n *NginxCli) GetVersion() (string, error) {
	params := []string{"-v"}
	result, err := n.parseCmdOutput(params, `nginx/(\d+\.\d+\.\d+)`, 1)
	if err != nil {
		return "", err
	}

	return result[0], nil
}

func (n *NginxCli) parseCmdOutput(params []string, regexpStr string, captureGroup uint) ([]string, error) {
	output, err := n.execCmd(params)
	if err != nil {
		return nil, err
	}

	reg, err := regexp.Compile(regexpStr)
	if err != nil {
		return nil, err
	}

	items := reg.FindAllSubmatch(output, -1)
	var rItems []string

	for _, item := range items {
		rItems = append(rItems, string(item[captureGroup]))
	}

	return rItems, nil
}

func (n *NginxCli) execCmd(params []string) ([]byte, error) {
	cmd := exec.Command(n.binPath, params...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, output)
	}

	return output, nil
}

func GetNginxCli(binPath string) (*NginxCli, error) {
	if binPath != "" {
		return &NginxCli{binPath: binPath}, nil
	}

	binPath, err := detectNginxCmd()
	if err != nil {
		return nil, err
	}

	return &NginxCli{binPath: binPath}, nil
}

func detectNginxCmd() (string, error) {
	cmd := exec.Command("which", nginxCmd)

	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	return "", errors.New("could not find nginx binary")
}
