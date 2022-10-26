package mng

import (
	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/r2dtools/webmng/pkg/webserver"
	"github.com/spf13/cobra"
)

var nginxCmd = &cobra.Command{
	Use:   "nginx",
	Short: "Manage nginx webserver",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.Flags().Set(flag.WebServerFlag, webserver.Nginx)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	nginxCmd.AddCommand(getHostsCmd())
	nginxCmd.AddCommand(getVersionCmd())
	nginxCmd.AddCommand(getCheckCmd())
	nginxCmd.AddCommand(getRestartCmd())
}
