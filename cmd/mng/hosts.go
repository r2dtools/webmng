package mng

import (
	"encoding/json"
	"strings"

	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func getHostsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "hosts",
		Short: "show host list",
		RunE: func(cmd *cobra.Command, args []string) error {
			var output []byte
			var err error

			code := cmd.Flag(flag.WebServerFlag).Value.String()
			webServerManager, err := GetWebServerManager(code, nil)
			if err != nil {
				return writeOutput(cmd, err.Error())
			}

			hosts, err := webServerManager.GetHosts()
			if err != nil {
				return writeOutput(cmd, err.Error())
			}

			if isJson {
				output, err = json.Marshal(hosts)
				if err != nil {
					return writeOutput(cmd, err.Error())
				}

				return writeOutput(cmd, string(output))
			}

			var outputParts []string

			for _, host := range hosts {
				output, err = yaml.Marshal(host)
				if err != nil {
					return writeOutput(cmd, err.Error())
				}
				outputParts = append(outputParts, string(output))
			}

			return writeOutput(cmd, strings.Join(outputParts, "\n"))
		},
	}

	return &cmd
}
