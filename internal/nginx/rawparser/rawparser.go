package rawparser

import (
	"os"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Config struct {
	Pos lexer.Position

	Entries []*Entry `@@*`
}

type Entry struct {
	Pos lexer.Position

	StartNewLines  []string        `@NewLine*`
	Comment        *Comment        `( @@`
	Directive      *Directive      `| @@`
	BlockDirective *BlockDirective `| @@ )`
	EndNewLines    []string        `@NewLine*`
}

type Comment struct {
	Pos lexer.Position

	Value string `@Comment`
}

type Directive struct {
	Pos lexer.Position

	Identifier string   `@Ident`
	Values     []*Value `@@+";"`
}

func (d *Directive) GetFirstValueStr() string {
	if len(d.Values) == 0 {
		return ""
	}

	return d.Values[0].Expression
}

func (d *Directive) GetExpressions() []string {
	return getExpressions(d.Values)
}

func (d *Directive) GetValues() []*Value {
	values := []*Value{}

	for _, value := range d.Values {
		if value != nil {
			values = append(values, value)
		}
	}

	return values
}

func (d *Directive) SetValues(expressions []string) {
	values := []*Value{}

	for _, expression := range expressions {
		values = append(values, &Value{Expression: expression})
	}

	d.Values = values
}

type BlockDirective struct {
	Pos lexer.Position

	Identifier string        `@Ident`
	Parameters []*Value      `@@*`
	Content    *BlockContent `"{" @@ "}"`
}

func (b *BlockDirective) GetEntries() []*Entry {
	entries := make([]*Entry, 0)

	if b.Content == nil {
		return entries
	}

	return b.Content.Entries
}

func (b *BlockDirective) FindEntriesWithIdentifier(identifier string) []*Entry {
	entries := []*Entry{}

	for _, entry := range b.GetEntries() {
		if entry != nil && entry.GetIdentifier() == identifier {
			entries = append(entries, entry)
		}
	}

	return entries
}

func (b *BlockDirective) GetParametersExpressions() []string {
	return getExpressions(b.Parameters)
}

type BlockContent struct {
	Pos lexer.Position

	Entries []*Entry `@@*`
}

type Value struct {
	Pos lexer.Position

	Expression string `@Expression | @String | @StringSingleQuoted`
}

func (e *Entry) GetIdentifier() string {
	if e.Directive != nil {
		return e.Directive.Identifier
	}

	if e.BlockDirective != nil {
		return e.BlockDirective.Identifier
	}

	return ""
}

type RawParser struct {
	participleParser *participle.Parser[Config]
}

func (p *RawParser) Parse(configPath string) (*Config, error) {
	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()

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
			{`Comment`, `(?:#)[^\n]*\n?`, nil},
			{"BlockEnd", `}`, nil},
			{`Ident`, `\w+`, lexer.Push("IdentParse")},
		},
		"IdentParse": {
			{`NewLine`, `[\r\n]+`, nil},
			{`whitespace`, `[^\S\r\n]+`, nil},
			{`String`, `"[^"]*"`, nil},
			{`StringSingleQuoted`, `'[^']*'`, nil},
			{"Semicolon", `;`, lexer.Pop()},
			{"BlockStart", `{`, lexer.Pop()},
			{"BlockEnd", `}`, lexer.Pop()},
			{"Expression", `[^;{}#\s]+`, nil},
			{`Comment`, `(?:#)[^\n]*\n?`, nil},
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
