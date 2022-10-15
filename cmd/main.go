package main

import (
	"github.com/r2dtools/webmng/cmd/host"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "webmng",
	Short: "A simple web server management utility",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var WebServer string

func main() {
	rootCmd.PersistentFlags().StringVarP(&WebServer, "webserver", "w", "", "webserver name (required)")
	rootCmd.MarkPersistentFlagRequired("webserver")
	rootCmd.AddCommand(host.HostListCmd)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
