package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBaseDirectives(t *testing.T) {
	type testData struct {
		block               *Block
		expectedServerNames []string
	}

	var serverName = "example.com"
	var serverAlias = "alias.example.com"
	var docRoot = "/var/www/html"

	items := []testData{
		{
			block:               nil,
			expectedServerNames: []string{},
		},
		{
			block:               &Block{Content: nil},
			expectedServerNames: []string{},
		},
		{
			block:               &Block{Content: &BlockContent{}},
			expectedServerNames: []string{},
		},
		{
			block: &Block{
				Content: &BlockContent{
					Entries: []*Entry{
						nil,
						{
							Identifier: "server_name",
							Values: []*Value{
								{Expression: serverName},
								{Expression: serverAlias},
							},
						},
						{
							Identifier: "fake",
							Values:     nil,
						},
					},
				},
			},
			expectedServerNames: []string{serverName, serverAlias},
		},
	}

	for _, item := range items {
		serverBlock := serverBlock{item.block}
		assert.ElementsMatch(t, item.expectedServerNames, serverBlock.getServerNames(), "invalid server names received")
	}

	docRootBlock := &Block{
		Content: &BlockContent{
			Entries: []*Entry{
				nil,
				{
					Identifier: "root",
					Values: []*Value{
						{Expression: docRoot},
					},
				},
			},
		},
	}
	serverBlock := serverBlock{docRootBlock}
	assert.Equal(t, docRoot, serverBlock.getDocumentRoot())
}

func TestGetListens(t *testing.T) {
	type testData struct {
		block    *Block
		expected []listen
	}

	items := []testData{
		{
			block: &Block{
				Content: &BlockContent{
					Entries: []*Entry{
						{
							Identifier: "ssl",
							Values: []*Value{
								{Expression: "on"},
							},
						},
						{
							Identifier: "listen",
							Values: []*Value{
								{Expression: "8443"},
							},
						},
						{
							Identifier: "listen",
							Values: []*Value{
								{Expression: "[::]:8443"},
							},
						},
					},
				},
			},
			expected: []listen{
				{
					hostPort: "8443",
					ssl:      true,
				},
				{
					hostPort: "[::]:8443",
					ssl:      true,
				},
			},
		},
		{
			block: &Block{
				Content: &BlockContent{
					Entries: []*Entry{
						{
							Identifier: "listen",
							Values: []*Value{
								{Expression: "443"},
								{Expression: "ssl"},
								{Expression: "http2"},
							},
						},
						{
							Identifier: "listen",
							Values: []*Value{
								{Expression: "[::]:443"},
								{Expression: "ssl"},
								{Expression: "http2"},
							},
						},
					},
				},
			},
			expected: []listen{
				{
					hostPort: "443",
					ssl:      true,
				},
				{
					hostPort: "[::]:443",
					ssl:      true,
				},
			},
		},
		{
			block: &Block{
				Content: &BlockContent{
					Entries: []*Entry{
						{
							Identifier: "listen",
							Values: []*Value{
								{Expression: "80"},
							},
						},
						{
							Identifier: "listen",
							Values: []*Value{
								{Expression: "[::]:80"},
							},
						},
					},
				},
			},
			expected: []listen{
				{
					hostPort: "80",
					ssl:      false,
				},
				{
					hostPort: "[::]:80",
					ssl:      false,
				},
			},
		},
	}

	for _, item := range items {
		serverBlock := serverBlock{item.block}
		listens := serverBlock.getListens()

		assert.Equal(t, item.expected, listens)
	}
}
