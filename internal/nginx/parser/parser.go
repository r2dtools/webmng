package parser

import (
	"path/filepath"
	"strings"

	iwebserver "github.com/r2dtools/webmng/internal/pkg/webserver"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/webserver"
	"github.com/r2dtools/webmng/pkg/webserver/host"
)

type Parser struct {
	rawParser   *RawParser
	parsedFiles map[string]*Config
	logger      logger.LoggerInterface
	serverRoot,
	configRoot string
}

func (p *Parser) GetHosts() ([]*webserver.Host, error) {
	if err := p.Parse(); err != nil {
		return nil, err
	}

	var hosts []*webserver.Host
	serverBlocks := p.getServerBlocks()

	for _, serverBlock := range serverBlocks {
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

		host := webserver.Host{
			FilePath:   serverBlock.block.Pos.Filename,
			ServerName: serverName,
			DocRoot:    serverBlock.getDocumentRoot(),
			Aliases:    aliases,
			Addresses:  addresses,
			Ssl:        ssl,
			Enabled:    true, // only enabled hosts are parsed for now
		}
		hosts = append(hosts, &host)
	}

	return hosts, nil
}

func (p *Parser) Parse() error {
	if p.parsedFiles != nil {
		return nil
	}

	p.parsedFiles = make(map[string]*Config)

	if err := p.parseRecursively(p.configRoot); err != nil {
		return err
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

func (p *Parser) parseFilesByPath(filePath string, override bool) ([]*Config, error) {
	files, err := filepath.Glob(filePath)
	if err != nil {
		return nil, err
	}

	var trees []*Config

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

func (p *Parser) getServerBlocks() []serverBlock {
	var blocks []serverBlock

	for _, tree := range p.parsedFiles {
		for _, entry := range tree.Entries {
			blocks = append(blocks, p.getServerBlocksRecursively(entry)...)
		}
	}

	return blocks
}

func (p *Parser) getServerBlocksRecursively(entry *Entry) []serverBlock {
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

func GetParser(serverRoot string, logger logger.LoggerInterface) (*Parser, error) {
	serverRoot, err := filepath.Abs(serverRoot)
	if err != nil {
		return nil, err
	}

	configRoot, err := iwebserver.GetConfigRootPath(serverRoot, []string{"nginx.conf"})
	if err != nil {
		return nil, err
	}

	rawParser, err := GetRawParser()
	if err != nil {
		return nil, err
	}

	parser := Parser{
		rawParser:  rawParser,
		logger:     logger,
		serverRoot: serverRoot,
		configRoot: configRoot,
	}

	return &parser, nil
}
