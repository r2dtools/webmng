package parser

import (
	"strings"

	"github.com/r2dtools/webmng/internal/nginx/rawparser"
)

type serverBlock struct {
	block *rawparser.BlockDirective
}

type Listen struct {
	HostPort string
	Ssl      bool
	Ipv6only bool
}

func (b serverBlock) getServerNames() []string {
	serverNames := []string{}

	entries := getBlockEntriesByIdentifier(b.block, "server_name")
	if len(entries) == 0 || entries[0].Directive == nil {
		return serverNames
	}

	for _, value := range entries[0].Directive.GetValues() {
		serverNames = append(serverNames, strings.TrimSpace(value.Expression))
	}

	return serverNames
}

func (b serverBlock) getDocumentRoot() string {
	entries := getBlockEntriesByIdentifier(b.block, "root")
	if len(entries) == 0 || entries[0].Directive == nil {
		return ""
	}

	return entries[0].Directive.GetFirstValueStr()
}

func (b serverBlock) getListens() []Listen {
	listens := []Listen{}
	entries := getBlockEntriesByIdentifier(b.block, "listen")
	sslEntries := getBlockEntriesByIdentifier(b.block, "ssl")
	serverSsl := false
	ipv6only := false

	// check first server block directive: ssl "on"
	for _, sslEntry := range sslEntries {
		directive := sslEntry.Directive

		if directive != nil && directive.GetFirstValueStr() == "on" {
			serverSsl = true
			break
		}
	}

	for _, entry := range entries {
		if entry == nil || entry.Directive == nil {
			continue
		}

		hostPort := entry.Directive.GetFirstValueStr()
		ssl := serverSsl

		for _, value := range entry.Directive.GetValues() {
			// check listen directive for "ssl" value
			// listen 443 ssl http2;
			if !ssl && value.Expression == "ssl" {
				ssl = true
			}

			if value.Expression == "ipv6only=on" {
				ipv6only = true
			}
		}

		listen := Listen{
			HostPort: hostPort,
			Ssl:      ssl,
			Ipv6only: ipv6only,
		}
		listens = append(listens, listen)
	}

	return listens
}
