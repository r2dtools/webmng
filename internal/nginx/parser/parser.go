package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/r2dtools/webmng/internal/nginx/dumper"
	"github.com/r2dtools/webmng/internal/nginx/rawparser"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/utils"
	"github.com/r2dtools/webmng/pkg/webserver"
	"github.com/r2dtools/webmng/pkg/webserver/host"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var includeDirective = "include"
var repeatableDirectives = []string{"server_name", "listen", includeDirective, "rewrite", "add_header"}

type Parser struct {
	rawParser   *rawparser.RawParser
	dumper      dumper.RawDumper
	parsedFiles map[string]*rawparser.Config
	logger      logger.LoggerInterface
	serverRoot,
	configRoot string
	changedFiles map[string]bool
}

type NginxHost struct {
	webserver.Host
	listens          []listen
	serverBlockIndex int
	Offset           int
}

func (h NginxHost) IsIpv6Only() bool {
	for _, listen := range h.listens {
		if listen.ipv6only {
			return true
		}
	}

	return false
}

type NginxDirective struct {
	Name          string
	Values        []string
	NewLineBefore bool
}

func (d *NginxDirective) AddValues(values ...string) {
	d.Values = append(d.Values, values...)
}

func (p *Parser) GetHosts() ([]NginxHost, error) {
	var hosts []NginxHost
	serverBlocks := p.getServerBlocks()

	for index, serverBlock := range serverBlocks {
		serverNames := serverBlock.getServerNames()
		serverName := ""
		aliases := []string{}

		if len(serverNames) > 0 {
			serverName = serverNames[0]
			aliases = serverNames[1:]
		}

		listens := serverBlock.getListens()
		addresses := make(map[string]host.Address)
		ssl := false

		for _, listen := range listens {
			address := host.CreateHostAddressFromString(listen.hostPort)
			addresses[address.GetHash()] = address
			if listen.ssl {
				ssl = true
			}

		}

		host := NginxHost{
			Host: webserver.Host{
				FilePath:   serverBlock.block.Pos.Filename,
				ServerName: serverName,
				DocRoot:    serverBlock.getDocumentRoot(),
				Aliases:    aliases,
				Addresses:  addresses,
				Ssl:        ssl,
				Enabled:    true, // only enabled hosts are parsed for now
			},
			Offset:           serverBlock.block.Pos.Offset,
			serverBlockIndex: index,
		}
		hosts = append(hosts, host)
	}

	return hosts, nil
}

func (p *Parser) Parse() error {
	p.changedFiles = make(map[string]bool)
	p.parsedFiles = make(map[string]*rawparser.Config)

	if err := p.parseRecursively(p.configRoot); err != nil {
		return err
	}

	return nil
}

func (p *Parser) GetChangedFiles() []string {
	files := make([]string, 0)

	for file, _ := range p.changedFiles {
		files = append(files, file)
	}

	return files
}

func (p *Parser) Dump() error {
	for changedFile := range p.changedFiles {
		config, ok := p.parsedFiles[changedFile]

		if !ok {
			continue
		}

		content, err := p.dumper.Dump(config)
		if err != nil {
			return fmt.Errorf("failed to dump config %s: %v", changedFile, err)
		}

		if err = os.WriteFile(changedFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write config %s: %v", changedFile, err)
		}
	}

	return nil
}

func (p *Parser) AddServerDirectives(host *NginxHost, directives []*NginxDirective, insertAtTop bool) error {
	serverBlock, err := p.getHostServerBlock(host)
	if err != nil {
		return err
	}

	return p.addBlockDirectives(serverBlock.block, directives, insertAtTop)
}

func (p *Parser) UpdateOrAddServerDirectives(host *NginxHost, directives []*NginxDirective, insertAtTop bool) error {
	serverBlock, err := p.getHostServerBlock(host)
	if err != nil {
		return err
	}

	return p.updateOrAddBlockDirectives(serverBlock.block, directives, insertAtTop)
}

func (p *Parser) addBlockDirectives(block *rawparser.Block, directives []*NginxDirective, insertAtTop bool) error {
	for _, directive := range directives {
		if err := p.addBlockDirective(block, directive, insertAtTop); err != nil {
			return err
		}
	}

	p.changedFiles[block.Pos.Filename] = true

	return nil
}

func (p *Parser) addBlockDirective(block *rawparser.Block, directive *NginxDirective, insertAtTop bool) error {
	existedEntries := block.FindEntriesWithIdentifier(directive.Name)

	if len(existedEntries) == 0 || slices.Contains(repeatableDirectives, directive.Name) {
		if block.Content == nil {
			return fmt.Errorf("unable to add directive: block content is nil")
		}

		var entryValues []*rawparser.Value

		for _, value := range directive.Values {
			entryValues = append(entryValues, &rawparser.Value{Expression: value})
		}

		entry := rawparser.Entry{
			Identifier: directive.Name,
			Values:     entryValues,
		}

		if directive.NewLineBefore {
			entry.StartNewLines = []string{"\n"}
		}

		if insertAtTop {
			block.Content.Entries = append([]*rawparser.Entry{&entry}, block.Content.Entries...)
		} else {
			block.Content.Entries = append(block.Content.Entries, &entry)
		}

		return nil
	}

	for _, entry := range existedEntries {
		values := entry.GetExpressions()

		if entry.Identifier == directive.Name && !reflect.DeepEqual(values, directive.Values) {
			return fmt.Errorf("unable to add directive %s: conflict %v:%v", directive.Name, values, directive.Values)
		}
	}

	return nil
}

func (p *Parser) updateOrAddBlockDirectives(block *rawparser.Block, directives []*NginxDirective, insertAtTop bool) error {
	for _, directive := range directives {
		if err := p.updateOrAddBlockDirective(block, directive, insertAtTop); err != nil {
			return err
		}
	}

	p.changedFiles[block.Pos.Filename] = true

	return nil
}

func (p *Parser) updateOrAddBlockDirective(block *rawparser.Block, directive *NginxDirective, insertAtTop bool) error {
	existedEntries := block.FindEntriesWithIdentifier(directive.Name)

	if len(existedEntries) == 0 {
		return p.addBlockDirective(block, directive, insertAtTop)
	}

	for _, exexistedEntry := range existedEntries {
		exexistedEntry.SetValues(directive.Values)
	}

	return nil
}

func (p *Parser) parseRecursively(configFilePath string) error {
	configFilePathAbs := p.getAbsPath(configFilePath)
	trees, err := p.parseFilesByPath(configFilePathAbs, false)
	if err != nil {
		return err
	}

	for _, tree := range trees {
		for _, entry := range tree.Entries {
			identifier := strings.ToLower(entry.Identifier)
			// Parse the top-level included file
			if identifier == "include" {
				includeFile := entry.GetFirstValueStr()
				if includeFile != "" {
					p.parseRecursively(includeFile)
				}
				continue
			}

			// Look for includes in the top-level 'http'/'server' context
			if identifier == "http" || identifier == "server" {
				if entry.Block == nil || entry.Block.Content == nil {
					continue
				}

				for _, subEntry := range entry.Block.Content.Entries {
					subIdentifier := strings.ToLower(subEntry.Identifier)
					if subIdentifier == "include" {
						includeFile := subEntry.GetFirstValueStr()
						if includeFile != "" {
							p.parseRecursively(includeFile)
						}
						continue
					}

					// Look for includes in a 'server' context within an 'http' context
					if identifier == "http" && subIdentifier == "server" {
						if subEntry.Block == nil || subEntry.Block.Content == nil {
							continue
						}

						for _, serverEntry := range subEntry.Block.Content.Entries {
							if strings.ToLower(serverEntry.Identifier) == "include" {
								includeFile := serverEntry.GetFirstValueStr()
								if includeFile != "" {
									p.parseRecursively(includeFile)
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (p *Parser) parseFilesByPath(filePath string, override bool) ([]*rawparser.Config, error) {
	files, err := filepath.Glob(filePath)
	if err != nil {
		return nil, err
	}

	var trees []*rawparser.Config

	for _, file := range files {
		if _, ok := p.parsedFiles[file]; ok && !override {
			continue
		}

		config, err := p.rawParser.Parse(file)
		if err != nil {
			p.logger.Warning("could not parse file %s: %v", file, err)
			continue
		}

		p.parsedFiles[file] = config
		trees = append(trees, config)
	}

	return trees, nil
}

func (p *Parser) getAbsPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	return filepath.Clean(filepath.Join(p.serverRoot, path))
}

func (p *Parser) findServerBlockByIndex(index int) (block serverBlock, ok bool) {
	serverBlocks := p.getServerBlocks()

	for i, sBlock := range serverBlocks {
		if index == i {
			return sBlock, true
		}
	}

	return serverBlock{}, false
}

func (p *Parser) getServerBlocks() []serverBlock {
	var blocks []serverBlock
	keys := maps.Keys[map[string]*rawparser.Config](p.parsedFiles)
	sort.Strings(keys)

	for _, key := range keys {
		tree, ok := p.parsedFiles[key]

		if !ok {
			continue
		}

		for _, entry := range tree.Entries {
			blocks = append(blocks, p.getServerBlocksRecursively(entry)...)
		}
	}

	return blocks
}

func (p *Parser) getServerBlocksRecursively(entry *rawparser.Entry) []serverBlock {
	var blocks []serverBlock
	block := entry.Block
	serverBlock := serverBlock{block}

	if block == nil {
		return blocks
	}

	if strings.ToLower(entry.Identifier) == "server" {
		blocks = append(blocks, serverBlock)
		return blocks // server blocks could not be nested
	}

	if block.Content == nil {
		return blocks
	}

	for _, entry := range block.Content.Entries {
		blocks = append(blocks, p.getServerBlocksRecursively(entry)...)
	}

	return blocks
}

func (p *Parser) getHostServerBlock(host *NginxHost) (serverBlock, error) {
	filename := host.FilePath
	sBlock, ok := p.findServerBlockByIndex(host.serverBlockIndex)

	if !ok {
		return serverBlock{}, fmt.Errorf("unable to find server block for host %s in %s", host.ServerName, filename)
	}

	return sBlock, nil
}

func GetParser(serverRoot string, logger logger.LoggerInterface) (*Parser, error) {
	serverRoot, err := filepath.Abs(serverRoot)
	if err != nil {
		return nil, err
	}

	configRoot, err := utils.FindAnyFilesInDirectory(serverRoot, []string{"nginx.conf"})
	if err != nil {
		return nil, err
	}

	rawParser, err := rawparser.GetRawParser()
	if err != nil {
		return nil, err
	}

	parser := Parser{
		rawParser:  rawParser,
		dumper:     dumper.RawDumper{},
		logger:     logger,
		serverRoot: serverRoot,
		configRoot: configRoot,
	}

	if err := parser.Parse(); err != nil {
		return nil, err
	}

	return &parser, nil
}
