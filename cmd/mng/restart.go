package mng

import (
	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/spf13/cobra"
)

func getRestartCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "restart",
		Short: "restart webserver",
		RunE: func(cmd *cobra.Command, args []string) error {
			code := cmd.Flag(flag.WebServerFlag).Value.String()
			webServerManager, err := GetWebServerManager(code, nil)

			if err != nil {
				return writeOutput(cmd, err.Error())
			}

			if err = webServerManager.Restart(); err != nil {
				return writeOutput(cmd, err.Error())
			}

			return writelnOutput(cmd, "ok")
		},
	}

	return &cmd
}
