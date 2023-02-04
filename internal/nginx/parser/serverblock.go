package parser

import (
	"strings"

	"github.com/r2dtools/webmng/internal/nginx/rawparser"
)

type serverBlock struct {
	block *rawparser.Block
}

type listen struct {
	hostPort string
	ssl      bool
	ipv6only bool
}

func (b serverBlock) getServerNames() []string {
	serverNames := []string{}

	entries := getBlockEntriesByIdentifier(b.block, "server_name")
	if len(entries) == 0 {
		return serverNames
	}

	for _, value := range entries[0].GetValues() {
		serverNames = append(serverNames, strings.TrimSpace(value.Expression))
	}

	return serverNames
}

func (b serverBlock) getDocumentRoot() string {
	entries := getBlockEntriesByIdentifier(b.block, "root")
	if len(entries) == 0 {
		return ""
	}

	return entries[0].GetFirstValueStr()
}

func (b serverBlock) getListens() []listen {
	listens := []listen{}
	entries := getBlockEntriesByIdentifier(b.block, "listen")
	sslEntries := getBlockEntriesByIdentifier(b.block, "ssl")
	serverSsl := false
	ipv6only := false

	// check first server block directive: ssl "on"
	for _, sslEntry := range sslEntries {
		if sslEntry.GetFirstValueStr() == "on" {
			serverSsl = true
			break
		}
	}

	for _, entry := range entries {
		if entry == nil || len(entry.Values) == 0 {
			continue
		}

		hostPort := entry.Values[0].Expression
		ssl := serverSsl

		for _, value := range entry.GetValues() {
			// check listen directive for "ssl" value
			// listen 443 ssl http2;
			if !ssl && value.Expression == "ssl" {
				ssl = true
			}

			if value.Expression == "ipv6only=on" {
				ipv6only = true
			}
		}

		listen := listen{
			hostPort: hostPort,
			ssl:      ssl,
			ipv6only: ipv6only,
		}
		listens = append(listens, listen)
	}

	return listens
}
