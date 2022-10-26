package apachectl

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type ApacheCtl struct {
	binPath string
}

func GetApacheCtl(binPath string) (*ApacheCtl, error) {
	if binPath != "" {
		return &ApacheCtl{binPath: binPath}, nil
	}

	binPath, err := detectCtlCmd()

	if err != nil {
		return nil, err
	}

	return &ApacheCtl{binPath: binPath}, nil
}

// ParseIncludes returns Include directives from httpd process and returns a list of their values.
func (a *ApacheCtl) ParseIncludes() ([]string, error) {
	params := []string{"-t", "-D", "DUMP_INCLUDES"}

	return a.parseCmdOutput(params, `\(.*\) (.*)`, 1)
}

// ParseModules return the list of loaded module names.
func (a *ApacheCtl) ParseModules() ([]string, error) {
	params := []string{"-t", "-D", "DUMP_MODULES"}

	return a.parseCmdOutput(params, `(.*)_module`, 1)
}

// ParseDefines returns a map of the defined variables.
func (a *ApacheCtl) ParseDefines() (map[string]string, error) {
	params := []string{"-t", "-D", "DUMP_RUN_CFG"}
	items, err := a.parseCmdOutput(params, `Define: ([^ \n]*)`, 1)

	if err != nil {
		return nil, err
	}

	variables := make(map[string]string)

	for _, item := range items {
		if item == "DUMP_RUN_CFG" {
			continue
		}

		if strings.Count(item, "=") > 1 {
			return nil, fmt.Errorf("error parsing apache runtime variables")
		}

		parts := strings.Split(item, "=")

		if len(parts) == 1 {
			variables[parts[0]] = ""
		} else {
			variables[parts[0]] = parts[1]
		}
	}

	return variables, nil
}

// GetVersion returns apache version
func (a *ApacheCtl) GetVersion() (string, error) {
	params := []string{"-v"}
	result, err := a.parseCmdOutput(params, `(?i)Apache/([0-9\.]*)`, 1)

	if err != nil {
		return "", err
	}

	if len(result) < 1 {
		return "", errors.New("could not detect apache version")
	}

	return result[0], nil
}

// TestConfiguration checks the syntax of apache configuration files
func (a *ApacheCtl) TestConfiguration() error {
	if _, err := a.execCmd([]string{"-t"}); err != nil {
		return err
	}

	return nil
}

// Restart restarts apache webserver
func (a *ApacheCtl) Restart() error {
	if _, err := a.execCmd([]string{"-k", "restart"}); err != nil {
		return err
	}

	return nil
}

func (a *ApacheCtl) parseCmdOutput(params []string, regexpStr string, captureGroup uint) ([]string, error) {
	output, err := a.execCmd(params)
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

func (a *ApacheCtl) execCmd(params []string) ([]byte, error) {
	cmd := exec.Command(a.binPath, params...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, output)
	}

	return output, nil
}

func detectCtlCmd() (string, error) {
	ctlCmds := []string{"apache2ctl", "httpd"}

	for _, ctlCmd := range ctlCmds {
		cmd := exec.Command("which", ctlCmd)

		if _, err := cmd.Output(); err == nil {
			return ctlCmd, nil
		}
	}

	return "", errors.New("could not find apachectl utility binary")
}
