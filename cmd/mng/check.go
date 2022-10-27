package mng

import (
	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/spf13/cobra"
)

func getCheckCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "check",
		Short: "check webserver configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			code := cmd.Flag(flag.WebServerFlag).Value.String()
			webServerManager, err := GetWebServerManager(code, nil)
			if err != nil {
				return writeOutput(cmd, err.Error())
			}

			err = webServerManager.CheckConfiguration()
			if err != nil {
				return writeOutput(cmd, err.Error())
			}

			return writelnOutput(cmd, "ok")
		},
	}

	return &cmd
}
