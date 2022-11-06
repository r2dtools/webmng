package parser

import (
	"os"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Value struct {
	Pos lexer.Position

	Expression string `@Expression | @String | @StringSingleQuoted`
}

type Block struct {
	Pos lexer.Position

	Parameters []*Value      `@@*`
	Content    *BlockContent `"{" @@ "}"`
}

func (b *Block) GetParametersExpressions() []string {
	return getExpressions(b.Parameters)
}

type BlockContent struct {
	Pos lexer.Position

	Entries     []*Entry `@@*`
	EndNewLines []string `@NewLine*`
}

type Entry struct {
	Pos lexer.Position

	StartNewLines []string `@NewLine*`
	Identifier    string   `@Ident`
	Values        []*Value `( @@+";"`
	Block         *Block   `| @@)`
}

func (e *Entry) GetFirstValueStr() string {
	if len(e.Values) == 0 {
		return ""
	}

	return e.Values[0].Expression
}

func (e *Entry) GetExpressions() []string {
	return getExpressions(e.Values)
}

func (e *Entry) GetValues() []*Value {
	values := []*Value{}

	for _, value := range e.Values {
		if value != nil {
			values = append(values, value)
		}
	}

	return values
}

type Config struct {
	Pos lexer.Position

	Entries     []*Entry `@@*`
	EndNewLines []string `@NewLine*`
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
			{`NewLine`, `[\r\n]+`, nil},
			{`whitespace`, `[^\S\r\n]+`, nil},
			{`comment`, `#.*`, nil},
			{"BlockEnd", `}`, nil},
			{`Ident`, `\w+`, lexer.Push("IdentParse")},
		},
		"IdentParse": {
			{`NewLine`, `[\r\n]+`, nil},
			{`whitespace`, `[^\S\r\n]+`, nil},
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

func getExpressions(values []*Value) []string {
	expressions := []string{}

	for _, value := range values {
		if value != nil {
			expressions = append(expressions, value.Expression)
		}
	}

	return expressions
}
