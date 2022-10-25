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
				return err
			}

			version, err := webServerManager.GetVersion()
			if err != nil {
				return err
			}

			output := []byte(version + "\n")

			if isJson {
				output, err = json.Marshal(map[string]string{"version": version})
				if err != nil {
					return err
				}
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
