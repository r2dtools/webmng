package mng

import (
	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "webmng",
	Short: "A simple web server management utility",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var webServer string
var isJson bool

func init() {
	RootCmd.PersistentFlags().StringVarP(&webServer, flag.WebServerFlag, "w", "", "webserver name")
	RootCmd.PersistentFlags().MarkHidden(flag.WebServerFlag)
	RootCmd.PersistentFlags().BoolVarP(&isJson, flag.JsonOutput, "j", false, "show result in json format")
	RootCmd.AddCommand(apacheCmd)
	RootCmd.AddCommand(nginxCmd)
}

func writeOutput(cmd *cobra.Command, output string) error {
	_, err := cmd.OutOrStdout().Write([]byte(output))

	if err != nil {
		return err
	}

	return nil
}

func writelnOutput(cmd *cobra.Command, output string) error {
	return writeOutput(cmd, output+"\n")
}
