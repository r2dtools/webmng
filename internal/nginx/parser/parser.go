package parser

import (
	"errors"
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

var errInvalidDirective = errors.New("entry is not a directive")

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
	Listens          []Listen
	ServerBlockIndex int
	Offset           int
}

func (h NginxHost) IsIpv6Only() bool {
	for _, listen := range h.Listens {
		if listen.Ipv6only {
			return true
		}
	}

	return false
}

type NginxDirective struct {
	Name          string
	Values        []string
	NewLineBefore bool
	NewLineAfter  bool
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
			address := host.CreateHostAddressFromString(listen.HostPort)
			addresses[address.GetHash()] = address
			if listen.Ssl {
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
			Listens:          listens,
			Offset:           serverBlock.block.Pos.Offset,
			ServerBlockIndex: index,
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

func (p *Parser) addBlockDirectives(block *rawparser.BlockDirective, directives []*NginxDirective, insertAtTop bool) error {
	for _, directive := range directives {
		if err := p.addBlockDirective(block, directive, insertAtTop); err != nil {
			return err
		}
	}

	p.changedFiles[block.Pos.Filename] = true

	return nil
}

func (p *Parser) addBlockDirective(blockDirective *rawparser.BlockDirective, directive *NginxDirective, insertAtTop bool) error {
	existedEntries := blockDirective.FindEntriesWithIdentifier(directive.Name)

	if len(existedEntries) == 0 || slices.Contains(repeatableDirectives, directive.Name) {
		if blockDirective.Content == nil {
			return fmt.Errorf("unable to add directive: block content is nil")
		}

		var entryValues []*rawparser.Value

		for _, value := range directive.Values {
			entryValues = append(entryValues, &rawparser.Value{Expression: value})
		}

		entry := rawparser.Entry{
			Directive: &rawparser.Directive{
				Identifier: directive.Name,
				Values:     entryValues,
			},
		}

		if directive.NewLineBefore {
			entry.StartNewLines = []string{"\n"}
		}

		if directive.NewLineAfter {
			entry.EndNewLines = []string{"\n"}
		}

		if insertAtTop {
			blockDirective.Content.Entries = append([]*rawparser.Entry{&entry}, blockDirective.Content.Entries...)
		} else {
			blockDirective.Content.Entries = append(blockDirective.Content.Entries, &entry)
		}

		return nil
	}

	for _, entry := range existedEntries {
		if entry == nil || entry.Directive == nil {
			return errInvalidDirective
		}
		values := entry.Directive.GetExpressions()

		if entry.GetIdentifier() == directive.Name && !reflect.DeepEqual(values, directive.Values) {
			return fmt.Errorf("unable to add directive %s: conflict %v:%v", directive.Name, values, directive.Values)
		}
	}

	return nil
}

func (p *Parser) updateOrAddBlockDirectives(block *rawparser.BlockDirective, directives []*NginxDirective, insertAtTop bool) error {
	for _, directive := range directives {
		if err := p.updateOrAddBlockDirective(block, directive, insertAtTop); err != nil {
			return err
		}
	}

	p.changedFiles[block.Pos.Filename] = true

	return nil
}

func (p *Parser) updateOrAddBlockDirective(blockDirective *rawparser.BlockDirective, directive *NginxDirective, insertAtTop bool) error {
	existedEntries := blockDirective.FindEntriesWithIdentifier(directive.Name)

	if len(existedEntries) == 0 {
		return p.addBlockDirective(blockDirective, directive, insertAtTop)
	}

	for _, exexistedEntry := range existedEntries {
		if exexistedEntry.Directive == nil {
			return errInvalidDirective
		}

		exexistedEntry.Directive.SetValues(directive.Values)
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
			identifier := strings.ToLower(entry.GetIdentifier())
			// Parse the top-level included file
			if identifier == "include" {
				if entry.Directive == nil {
					return errInvalidDirective
				}

				includeFile := entry.Directive.GetFirstValueStr()
				if includeFile != "" {
					p.parseRecursively(includeFile)
				}
				continue
			}

			// Look for includes in the top-level 'http'/'server' context
			if identifier == "http" || identifier == "server" {
				if entry.BlockDirective == nil {
					continue
				}

				for _, subEntry := range entry.BlockDirective.GetEntries() {
					subIdentifier := strings.ToLower(subEntry.GetIdentifier())
					if subIdentifier == "include" {
						if subEntry.Directive == nil {
							return errInvalidDirective
						}

						includeFile := subEntry.Directive.GetFirstValueStr()
						if includeFile != "" {
							p.parseRecursively(includeFile)
						}
						continue
					}

					// Look for includes in a 'server' context within an 'http' context
					if identifier == "http" && subIdentifier == "server" {
						if subEntry.BlockDirective == nil {
							continue
						}

						for _, serverEntry := range subEntry.BlockDirective.GetEntries() {
							if strings.ToLower(serverEntry.GetIdentifier()) == "include" {
								if serverEntry.Directive == nil {
									return errInvalidDirective
								}

								includeFile := serverEntry.Directive.GetFirstValueStr()
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
	block := entry.BlockDirective
	serverBlock := serverBlock{block}

	if block == nil {
		return blocks
	}

	if strings.ToLower(entry.GetIdentifier()) == "server" {
		blocks = append(blocks, serverBlock)
		return blocks // server blocks could not be nested
	}

	for _, entry := range block.GetEntries() {
		blocks = append(blocks, p.getServerBlocksRecursively(entry)...)
	}

	return blocks
}

func (p *Parser) getHostServerBlock(host *NginxHost) (serverBlock, error) {
	filename := host.FilePath
	sBlock, ok := p.findServerBlockByIndex(host.ServerBlockIndex)

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
