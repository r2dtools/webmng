package mng

import (
	"fmt"
	"strings"

	"github.com/r2dtools/webmng/cmd/flag"
	"github.com/r2dtools/webmng/pkg/webserver"
	"github.com/spf13/cobra"
)

var (
	hostName,
	certPath,
	certKeyPath,
	certChainPath,
	certFullChainPath string
)

func getDeployCertificateCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "deploy-certificate",
		Short: "deploy certificate to host",
		RunE: func(cmd *cobra.Command, args []string) error {
			code := cmd.Flag(flag.WebServerFlag).Value.String()
			webServerManager, err := GetWebServerManager(code, nil)

			if err != nil {
				return writeOutput(cmd, err.Error())
			}

			if err = webServerManager.DeployCertificate(hostName, certPath, certKeyPath, certChainPath, certFullChainPath); err != nil {
				err = fmt.Errorf("could not deploy certificate to virtual host '%s': %v", hostName, err)

				return rollbackChanges(webServerManager, cmd, err)
			}

			if err = webServerManager.SaveChanges(); err != nil {
				err = fmt.Errorf("could not deploy certificate to host '%s': could not save changes for configuration: %v", hostName, err)

				return rollbackChanges(webServerManager, cmd, err)
			}

			if err = webServerManager.CheckConfiguration(); err != nil {
				err = fmt.Errorf("could not deploy certificate to host '%s': apache configuration is invalid: %v", hostName, err)

				return rollbackChanges(webServerManager, cmd, err)
			}

			if err = webServerManager.CommitChanges(); err != nil {
				return writeOutput(cmd, err.Error())
			}

			if err = webServerManager.Restart(); err != nil {
				return writeOutput(cmd, err.Error())
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&hostName, flag.HostFlag, "", "host name")
	cmd.MarkFlagRequired(flag.HostFlag)
	cmd.Flags().StringVar(&certPath, flag.CertPathFlag, "", "certificate path")
	cmd.Flags().StringVar(&certKeyPath, flag.CertKeyPathFlag, "", "certificate key path")
	cmd.MarkFlagRequired(flag.CertKeyPathFlag)
	cmd.Flags().StringVar(&certChainPath, flag.CertChainPathFlag, "", "certificate chain path")
	cmd.Flags().StringVar(&certFullChainPath, flag.CertFullChainPathFlag, "", "certificate full chain path")

	return &cmd
}

func rollbackChanges(webServerManager webserver.WebServerManagerInterface, cmd *cobra.Command, err error) error {
	var errMessages []string
	errMessages = append(errMessages, err.Error())

	if err = webServerManager.RollbackChanges(); err != nil {
		errMessages = append(errMessages, err.Error())
	}

	return writeOutput(cmd, strings.Join(errMessages, "\n"))
}
