package mng

import (
	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/r2dtools/webmng/pkg/webserver"
	"github.com/spf13/cobra"
)

var apacheCmd = &cobra.Command{
	Use:   "apache",
	Short: "Manage apache webserver",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.Flags().Set(flag.WebServerFlag, webserver.Apache)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	apacheCmd.AddCommand(getHostsCmd())
	apacheCmd.AddCommand(getVersionCmd())
}
