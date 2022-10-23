package nginx

import (
	"os"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Value struct {
	Pos lexer.Position

	Expression *string `@Expression | @String | @StringSingleQuoted`
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

type RawParser struct {
	participleParser *participle.Parser[Config]
}

func (p *RawParser) Parse(configPath string) (*Config, error) {
	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	config, err := p.participleParser.Parse("", configFile)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func GetRawParser() (*RawParser, error) {
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
			{`String`, `"[^"]*"`, nil},
			{`StringSingleQuoted`, `'[^']*'`, nil},
			{"Semicolon", `;`, lexer.Pop()},
			{"BlockStart", `{`, lexer.Pop()},
			{"BlockEnd", `}`, lexer.Pop()},
			{"Expression", `[^;{}#\s]+`, nil},
		},
	})

	participleParser, err := participle.Build[Config](
		participle.Lexer(def),
		participle.UseLookahead(50),
	)
	if err != nil {
		return nil, err
	}

	parser := RawParser{
		participleParser: participleParser,
	}

	return &parser, nil
}
