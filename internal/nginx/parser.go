package nginx

import (
	"os"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Value struct {
	Pos lexer.Position

	Expression *string `@Expression | @String`
}

type Block struct {
	Pos lexer.Position

	Parameters []*Value      `@@*`
	Content    *BlockContent `"{" @@ "}"`
}

type BlockContent struct {
	Pos lexer.Position

	Entries []*Entry `@@*`
}

type Entry struct {
	Pos lexer.Position

	Identifier string   `@Ident`
	Values     []*Value `( @@+";"`
	Block      *Block   `| @@)`
}

type Config struct {
	Pos lexer.Position

	Entries []*Entry `@@*`
}

type Parser struct {
	configRoot       string
	participleParser *participle.Parser[Config]
	parsedConfigs    map[string]*Config
}

func (p *Parser) Parse() (map[string]*Config, error) {
	configFile, err := os.Open(p.configRoot)
	if err != nil {
		return nil, err
	}
	config, err := p.participleParser.Parse("", configFile)
	if err != nil {
		return nil, err
	}

	p.parsedConfigs[p.configRoot] = config

	return p.parsedConfigs, nil
}

func (p *Parser) parseRecursively(configPath string) error {
	return nil
}

func GetParser(configRoot string) (*Parser, error) {
	def := lexer.MustStateful(lexer.Rules{
		"Root": {
			{`whitespace`, `\s+`, nil},
			{`comment`, `#.*`, nil},
			{"BlockEnd", `}`, nil},
			{`Ident`, `\w+`, lexer.Push("IdentParse")},
		},
		"IdentParse": {
			{`whitespace`, `\s+`, nil},
			{`comment`, `#.*`, nil},
			{`String`, `"[\"]*"`, nil},
			{"Semicolon", `;`, lexer.Pop()},
			{"BlockStart", `{`, lexer.Pop()},
			{"BlockEnd", `}`, lexer.Pop()},
			{"Expression", `[^;{}#\s]+`, nil},
		},
	})

	participleParser, err := participle.Build[Config](
		participle.Lexer(def),
		participle.Unquote(),
		participle.UseLookahead(50),
	)
	if err != nil {
		return nil, err
	}

	parser := Parser{
		configRoot:       configRoot,
		participleParser: participleParser,
		parsedConfigs:    make(map[string]*Config),
	}

	return &parser, nil
}
