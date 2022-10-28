package host

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Address represents host address
type Address struct {
	IsIpv6 bool
	Host   string
	Port   string
}

// CreateHostAddressFromString parses address string and returns Address structure
func CreateHostAddressFromString(addrStr string) Address {
	var host, port string
	// ipv6 addresses starts with
	if strings.HasPrefix(addrStr, "[") {
		lastIndex := strings.LastIndex(addrStr, "]")
		host = addrStr[:lastIndex+1]

		if len(addrStr) > lastIndex+2 && string(addrStr[lastIndex+1]) == ":" {
			port = addrStr[lastIndex+2:]
		}

		return Address{
			Host:   host,
			Port:   port,
			IsIpv6: true,
		}
	}

	parts := strings.Split(addrStr, ":")
	if len(parts) == 0 {
		return Address{}
	}

	if _, err := strconv.Atoi(parts[0]); err == nil {
		port = parts[0]
	} else {
		host = parts[0]
		if len(parts) > 1 {
			port = parts[1]
		}
	}

	return Address{
		Host: host,
		Port: port,
	}
}

func (a *Address) IsWildcardPort() bool {
	return a.Port == "*" || a.Port == ""
}

// GetHash returns addr hash based on host an port
func (a *Address) GetHash() string {
	addr := fmt.Sprintf("%s:%s", a.Host, a.Port)

	return base64.StdEncoding.EncodeToString([]byte(addr))
}

func (a *Address) ToString() string {
	if a.Port != "" {
		return fmt.Sprintf("%s:%s", a.Host, a.Port)
	}

	return a.Host
}

// GetAddressWithNewPort returns new a Address instance with changed port
func (a *Address) GetAddressWithNewPort(port string) *Address {
	return &Address{
		Host:   a.Host,
		Port:   port,
		IsIpv6: a.IsIpv6,
	}
}

// GetNormalizedHost returns normalized host.
// Normalization occurres only for ipv6 address. Ipv4 returns as is.
// For example: [fd00:dead:beaf::1] -> fd00:dead:beaf:0:0:0:0:1
func (a *Address) GetNormalizedHost() string {
	if a.IsIpv6 {
		return a.GetNormalizedIpv6()
	}

	return a.Host
}

// GetNormalizedIpv6 returns normalized IPv6
// For example: [fd00:dead:beaf::1] -> fd00:dead:beaf:0:0:0:0:1
func (a *Address) GetNormalizedIpv6() string {
	if !a.IsIpv6 {
		return ""
	}

	return strings.Join(a.normalizeIpv6(a.Host), ":")
}

func (a *Address) IsEqual(b *Address) bool {
	if a.Port != b.Port {
		return false
	}

	return a.GetNormalizedHost() == b.GetNormalizedHost()
}

func (a *Address) normalizeIpv6(addr string) []string {
	addr = strings.Trim(addr, "[]")

	return a.explodeIpv6(addr)
}

func (a *Address) explodeIpv6(addr string) []string {
	result := []string{"0", "0", "0", "0", "0", "0", "0", "0"}
	addrParts := strings.Split(addr, ":")
	var appendToEnd bool

	if len(addrParts) > len(result) {
		addrParts = addrParts[:len(result)]
	}

	for i, block := range addrParts {
		if block == "" {
			appendToEnd = true
			continue
		}

		if len(block) > 1 {
			block = strings.TrimLeft(block, "0")
		}

		if !appendToEnd {
			result[i] = block
		} else {
			result[len(result)-len(addrParts)+i] = block
		}
	}

	return result
}
