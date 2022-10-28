package parser

import "strings"

func getBlockEntriesByIdentifier(block *Block, identifier string) []*Entry {
	entries := []*Entry{}

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
