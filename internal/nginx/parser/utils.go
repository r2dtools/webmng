package parser

import (
	"strings"

	"github.com/r2dtools/webmng/internal/nginx/rawparser"
)

func getBlockEntriesByIdentifier(block *rawparser.Block, identifier string) []*rawparser.Entry {
	entries := []*rawparser.Entry{}

	if block == nil || block.Content == nil {
		return entries
	}

	for _, entry := range block.Content.Entries {
		if entry == nil {
			continue
		}
		if strings.ToLower(entry.Identifier) == identifier {
			entries = append(entries, entry)
		}
	}

	return entries
}
