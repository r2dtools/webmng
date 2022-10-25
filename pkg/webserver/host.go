package webserver

import (
	"path/filepath"
	"strings"

	"github.com/r2dtools/webmng/pkg/webserver/host"
)

type Host struct {
	FilePath,
	ServerName,
	DocRoot string
	Addresses map[string]host.Address
	Aliases   []string
	Ssl,
	Enabled bool
}

// GetConfigName returns config name of a host
func (h *Host) GetConfigName() string {
	return filepath.Base(h.FilePath)
}

// GetAddressesString return address as a string: "172.10.52.2:80 172.10.52.3:8080"
func (h *Host) GetAddressesString(hostsOnly bool) string {
	var addresses []string
	for _, address := range h.Addresses {
		if hostsOnly {
			addresses = append(addresses, address.Host)
		} else {
			addresses = append(addresses, address.ToString())
		}
	}

	return strings.Join(addresses, " ")
}
