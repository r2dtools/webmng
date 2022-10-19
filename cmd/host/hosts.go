package host

import (
	"encoding/json"

	"github.com/r2dtools/webmng/cmd/webserver"
	"github.com/spf13/cobra"
)

var HostListCmd = &cobra.Command{
	Use:   "hosts",
	Short: "Show host list",
	RunE: func(cmd *cobra.Command, args []string) error {
		code := cmd.Flag("webserver").Value.String()
		webServerManager, err := webserver.GetWebServerManager(code, nil)
		if err != nil {
			return err
		}

		hosts, err := webServerManager.GetHosts()
		if err != nil {
			return err
		}

		data, err := json.MarshalIndent(hosts, "", " ")
		if err != nil {
			return err
		}

		_, err = cmd.OutOrStdout().Write(data)
		if err != nil {
			return err
		}

		return nil
	},
}
