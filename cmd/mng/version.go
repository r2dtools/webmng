package mng

import (
	"encoding/json"

	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/spf13/cobra"
)

func getVersionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "version",
		Short: "Show webserver version",
		RunE: func(cmd *cobra.Command, args []string) error {
			code := cmd.Flag(flag.WebServerFlag).Value.String()
			webServerManager, err := GetWebServerManager(code, nil)

			if err != nil {
				return writeOutput(cmd, err.Error())
			}

			version, err := webServerManager.GetVersion()

			if err != nil {
				return writeOutput(cmd, err.Error())
			}

			if isJson {
				output, err := json.Marshal(map[string]string{"version": version})

				if err != nil {
					return writeOutput(cmd, err.Error())
				}

				return writeOutput(cmd, string(output))
			}

			return writelnOutput(cmd, version)
		},
	}

	return &cmd
}
