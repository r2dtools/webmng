package mng

import (
	"encoding/json"

	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/spf13/cobra"
)

func getHostsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "hosts",
		Short: "Show host list",
		RunE: func(cmd *cobra.Command, args []string) error {
			var output []byte
			var err error

			code := cmd.Flag(flag.WebServerFlag).Value.String()
			webServerManager, err := GetWebServerManager(code, nil)
			if err != nil {
				return err
			}

			hosts, err := webServerManager.GetHosts()
			if err != nil {
				return err
			}

			if isJson {
				output, err = json.Marshal(hosts)
			} else {
				output, err = json.MarshalIndent(hosts, "", "    ")
			}

			if err != nil {
				return err
			}

			_, err = cmd.OutOrStdout().Write(output)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return &cmd
}
