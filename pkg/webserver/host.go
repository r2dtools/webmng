package webserver

import (
	"path/filepath"
	"strings"

	"github.com/r2dtools/webmng/pkg/webserver/host"
)

type HostInterface interface {
	GetDocRoot() string
	GetServerName() string
	GetAliases() []string
	GetAddresses() map[string]host.Address
	GetAddressesString(hostOnly bool) string
	IsSslEnabled() bool
	IsEnabled() bool
	GetConfigName() string
	GetConfigPath() string
}

type Host struct {
	FilePath,
	ServerName,
	DocRoot,
	AugPath string
	Addresses map[string]host.Address
	Aliases   []string
	ModMacro,
	Ssl,
	Enabled bool
}

func (h *Host) GetServerName() string {
	return h.ServerName
}

func (h *Host) GetDocRoot() string {
	return h.DocRoot
}

func (h *Host) GetAddresses() map[string]host.Address {
	return h.Addresses
}

func (h *Host) IsSslEnabled() bool {
	return h.Ssl
}

func (h *Host) IsEnabled() bool {
	return h.Enabled
}

func (h *Host) GetAliases() []string {
	return h.Aliases
}

// GetConfigName returns config name of a host
func (h *Host) GetConfigName() string {
	return filepath.Base(h.FilePath)
}

func (h *Host) GetConfigPath() string {
	return h.FilePath
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
