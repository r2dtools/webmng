package dumper

import (
	"errors"
	"strings"

	"github.com/r2dtools/webmng/internal/nginx/parser"
)

const (
	tab     = "\t"
	space   = " "
	newLine = "\n"
)

type RawDumper struct {
	nestingLevel int
}

func (d *RawDumper) Dump(config *parser.Config) (string, error) {
	if config == nil {
		return "", errors.New("config is empty")
	}

	result := d.dumpEntries(config.Entries)
	result += strings.Join(config.EndNewLines, "")

	return result, nil
}

func (d *RawDumper) dumpEntries(entries []*parser.Entry) string {
	var result string

	for _, entry := range entries {
		if entry != nil {
			result += d.dumpEntry(entry)
		}
	}

	return result
}

func (d *RawDumper) dumpEntry(entry *parser.Entry) string {
	result := strings.Join(entry.StartNewLines, "")

	if entry.Block != nil {
		result += d.dumpBlock(entry)
	} else {
		result += d.dumpDirective(entry)
	}

	return result
}

func (d *RawDumper) dumpBlock(entry *parser.Entry) string {
	result := d.getCurrentIdent() + entry.Identifier
	block := entry.Block
	parameters := strings.Join(block.GetParametersExpressions(), space)

	if parameters != "" {
		result += space + parameters
	}

	result += space + "{"

	if block.Content != nil {
		d.increaseNestingLevel()
		result += d.dumpEntries(block.Content.Entries)
		result += strings.Join(block.Content.EndNewLines, "")
		d.decreaseNestingLevel()
	}

	result += d.getCurrentIdent() + "}"

	return result
}

func (d *RawDumper) dumpDirective(entry *parser.Entry) string {
	expression := strings.Join(entry.GetExpressions(), space)

	return d.getCurrentIdent() + entry.Identifier + space + expression + ";"
}

func (d *RawDumper) getCurrentIdent() string {
	return strings.Repeat(tab, d.nestingLevel)
}

func (d *RawDumper) increaseNestingLevel() {
	d.nestingLevel++
}

func (d *RawDumper) decreaseNestingLevel() {
	if d.nestingLevel > 0 {
		d.nestingLevel--
	}
}
