package host

import (
	"fmt"

	"github.com/spf13/cobra"
)

var HostListCmd = &cobra.Command{
	Use:   "hosts",
	Short: "Show host list",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Not implemented yet")

		return nil
	},
}
