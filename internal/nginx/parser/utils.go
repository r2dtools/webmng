package parser

import (
	"strings"

	"github.com/r2dtools/webmng/internal/nginx/rawparser"
)

func getBlockEntriesByIdentifier(blockDirective *rawparser.BlockDirective, identifier string) []*rawparser.Entry {
	entries := []*rawparser.Entry{}

	if blockDirective == nil {
		return entries
	}

	for _, entry := range blockDirective.GetEntries() {
		if entry == nil {
			continue
		}
		if strings.ToLower(entry.GetIdentifier()) == identifier {
			entries = append(entries, entry)
		}
	}

	return entries
}
