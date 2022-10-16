package apache

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/r2dtools/webmng/internal/apache/apachectl"
	"github.com/r2dtools/webmng/internal/apache/utils"
	"github.com/r2dtools/webmng/internal/pkg/parser"
	"github.com/r2dtools/webmng/internal/pkg/webserver"
	"github.com/unknwon/com"
	"honnef.co/go/augeas"
)

const (
	argVarRegex = `\$\{[^ \}]*}`
)

var fnMatchChars = []string{"*", "?", "\\", "[", "]"}

type directiveFilter struct {
	Name  string
	Value []string
}

type Parser struct {
	parser.BaseParser

	apachectl    *apachectl.ApacheCtl
	configListen string
	variables    map[string]string
	modules      map[string]bool
}

// FindDirective finds directive in configuration
// directive - directive to look for
// arg - directive value. If empty string then all directives should be considrered
// start - Augeas path that should be used to begin looking for the directive
// exclude - whether or not to exclude directives based on variables and enabled modules
func (p *Parser) FindDirective(directive, arg, start string, exclude bool) ([]string, error) {
	if start == "" {
		start = p.GetAugPath(p.ConfigRoot)
	}

	regStr := fmt.Sprintf("(%s)|(%s)|(%s)", directive, "Include", "IncludeOptional")
	matches, err := p.Augeas.Match(fmt.Sprintf("%s//*[self::directive=~regexp('%s', 'i')]", start, regStr))
	if err != nil {
		return nil, err
	}

	if exclude {
		matches, err = p.excludeDirectives(matches)
		if err != nil {
			return nil, err
		}
	}

	var argSuffix string
	var orderedMatches []string

	if arg == "" {
		argSuffix = "/arg"
	} else {
		argSuffix = fmt.Sprintf("/*[self::arg=~regexp('%s', 'i')]", arg)
	}

	for _, match := range matches {
		dir, err := p.Augeas.Get(match)
		if err != nil {
			return nil, err
		}

		dir = strings.ToLower(dir)

		if dir == "include" || dir == "includeoptional" {
			nArg, err := p.GetArg(match + "/arg")
			if err != nil {
				return nil, err
			}

			nStart, err := p.getIncludePath(nArg)
			if err != nil {
				return nil, err
			}

			nMatches, err := p.FindDirective(directive, arg, nStart, exclude)
			if err != nil {
				return nil, err
			}

			orderedMatches = append(orderedMatches, nMatches...)
		}

		if dir == strings.ToLower(directive) {
			nMatches, err := p.Augeas.Match(match + argSuffix)
			if err != nil {
				return nil, err
			}

			orderedMatches = append(orderedMatches, nMatches...)
		}

	}

	return orderedMatches, nil
}

// AddDirective adds directive to the end of the file given by augConfPath
func (p *Parser) AddDirective(augConfPath string, directive string, args []string) error {
	if err := p.Augeas.Set(augConfPath+"/directive[last() + 1]", directive); err != nil {
		return err
	}

	for i, arg := range args {
		if err := p.Augeas.Set(fmt.Sprintf("%s/directive[last()]/arg[%d]", augConfPath, i+1), arg); err != nil {
			return err
		}
	}

	return nil
}

// GetArg returns argument value and interprets result
func (p *Parser) GetArg(match string) (string, error) {
	value, err := p.Augeas.Get(match)

	if err != nil {
		return "", err
	}

	value = strings.Trim(value, "'\"")
	re := regexp.MustCompile(argVarRegex)
	variables := re.FindAll([]byte(value), -1)

	for _, variable := range variables {
		variableStr := string(variable)
		// Since variable is satisfied regex, it has at least length 3: ${}
		variableKey := variableStr[2 : len(variableStr)-1]
		replaceVariable, ok := p.variables[variableKey]

		if !ok {
			return "", fmt.Errorf("could not parse variable: %s", variableStr)
		}

		value = strings.Replace(value, variableStr, replaceVariable, -1)
	}

	return value, nil
}

// UpdateRuntimeVariables Updates Includes, Defines and Includes from httpd config dump data
func (p *Parser) UpdateRuntimeVariables() error {
	if err := p.updateDefines(); err != nil {
		return err
	}

	if err := p.updateIncludes(); err != nil {
		return err
	}

	if err := p.updateModules(); err != nil {
		return err
	}

	return nil
}

// excludeDirectives excludes directives that are not loaded into the configuration.
func (p *Parser) excludeDirectives(matches []string) ([]string, error) {
	var validMatches []string
	filters := []directiveFilter{
		{"ifmodule", p.getModules()},
		{"ifdefine", p.getVariblesNames()},
	}

	for _, match := range matches {
		isPassed := true

		for _, filter := range filters {
			fPassed, err := p.isDirectivePassedFilter(match, filter)
			if err != nil {
				return nil, fmt.Errorf("failed to check the directive '%s' passed the filter '%s'", match, filter.Name)
			}

			if !fPassed {
				isPassed = false
				break
			}
		}

		if isPassed {
			validMatches = append(validMatches, match)
		}
	}

	return validMatches, nil
}

// GetModules returns loaded modules from httpd process
func (p *Parser) getModules() []string {
	modules := make([]string, len(p.modules))

	for module := range p.modules {
		modules = append(modules, module)
	}

	return modules
}

func (p *Parser) getVariblesNames() []string {
	names := make([]string, len(p.variables))

	for name := range p.variables {
		names = append(names, name)
	}

	return names
}

// isDirectivePassedFilter checks if directive can pass a filter
func (p *Parser) isDirectivePassedFilter(match string, filter directiveFilter) (bool, error) {
	lMatch := strings.ToLower(match)
	lastMatchIdx := strings.Index(lMatch, filter.Name)

	for lastMatchIdx != -1 {
		endOfIfIdx := strings.Index(lMatch[lastMatchIdx:], "/")

		if endOfIfIdx == -1 {
			endOfIfIdx = len(lMatch)
		} else {
			endOfIfIdx += lastMatchIdx
		}

		expression, err := p.Augeas.Get(match[:endOfIfIdx] + "/arg")

		if err != nil {
			return false, err
		}

		if strings.HasPrefix(expression, "!") {
			if com.IsSliceContainsStr(filter.Value, expression[1:]) {
				return false, nil
			}
		} else {
			if !com.IsSliceContainsStr(filter.Value, expression) {
				return false, nil
			}
		}

		lastMatchIdx = strings.Index(lMatch[endOfIfIdx:], filter.Name)

		if lastMatchIdx != -1 {
			lastMatchIdx += endOfIfIdx
		}
	}

	return true, nil
}

// getIncludePath converts Apache Include directive to Augeas path
func (p *Parser) getIncludePath(arg string) (string, error) {
	arg = p.ConvertPathFromServerRootToAbs(arg)
	info, err := os.Stat(arg)

	if err == nil && info.IsDir() {
		p.ParseFile(filepath.Join(arg, "*"))
	} else {
		p.ParseFile(arg)
	}

	argParts := strings.Split(arg, "/")

	for index, part := range argParts {
		for _, char := range part {
			if com.IsSliceContainsStr(fnMatchChars, string(char)) {
				argParts[index] = fmt.Sprintf("* [label()=~regexp('%s')]", p.fnMatchToRegex(part))
				break
			}
		}
	}

	arg = strings.Join(argParts, "/")

	return p.GetAugPath(arg), nil
}

func (p *Parser) fnMatchToRegex(fnMatch string) string {
	regex := utils.TranslateFnmatchToRegex(fnMatch)

	return regex[4 : len(regex)-2]
}

// updateDefines Updates the map of known variables in the configuration
func (p *Parser) updateDefines() error {
	variables, err := p.apachectl.ParseDefines()
	if err != nil {
		return fmt.Errorf("could not parse defines: %v", err)
	}

	p.variables = variables

	return nil
}

// updateIncludes gets includes from httpd process, and add them to DOM if needed
func (p *Parser) updateIncludes() error {
	// FindDirective iterates over configuration for Include and IncludeOptional
	// directives to make sure we see the full include tree present in the
	// configuration files
	p.FindDirective("Include", "", "", true)

	matches, err := p.apachectl.ParseIncludes()
	if err != nil {
		return fmt.Errorf("could not parse inlcludes: %v", err)
	}

	for _, match := range matches {
		if !p.IsFilenameExistInLoadedPaths(match) {
			p.ParseFile(match)
		}
	}

	return nil
}

// ResetModules resets the loaded modules list
func (p *Parser) ResetModules() error {
	p.modules = make(map[string]bool)
	if err := p.updateModules(); err != nil {
		return err
	}

	// p.ParseModules() TODO: apache config should be also parsed for LoadModule directive
	return nil
}

func (p *Parser) updateModules() error {
	matches, err := p.apachectl.ParseModules()
	if err != nil {
		return err
	}

	for _, module := range matches {
		p.addModule(strings.TrimSpace(module))
	}

	return nil
}

func (p *Parser) addModule(name string) {
	modKey := fmt.Sprintf("%s_module", name)

	if _, ok := p.modules[modKey]; !ok {
		p.modules[modKey] = true
	}

	modKey = fmt.Sprintf("mod_%s.c", name)

	if _, ok := p.modules[modKey]; !ok {
		p.modules[modKey] = true
	}
}

func GetParser(apachectl *apachectl.ApacheCtl, version, serverRoot, hostRoot, hostFiles string) (*Parser, error) {
	serverRoot, err := webserver.GetServerRootPath(serverRoot, serverRootPaths)
	if err != nil {
		return nil, err
	}

	if hostRoot != "" {
		hostRoot, err = filepath.Abs(hostRoot)
		if err != nil {
			return nil, err
		}
	}

	// try to detect apache root config file path (ex. /etc/apache2/apache2.conf), ports.conf file path
	configRoot, err := webserver.GetConfigRootPath(serverRoot, configFiles)
	if err != nil {
		return nil, err
	}

	configListen := getConfigListen(serverRoot, configRoot)

	aug, err := augeas.New("/", "", augeas.NoLoad|augeas.NoModlAutoload|augeas.EnableSpan)
	if err != nil {
		return nil, err
	}

	parser := Parser{
		BaseParser: parser.BaseParser{
			Augeas:        aug,
			ServerRoot:    serverRoot,
			ConfigRoot:    configRoot,
			HostRoot:      hostRoot,
			Version:       version,
			ExistingPaths: make(map[string][]string),
		},
		apachectl:    apachectl,
		configListen: configListen,
		variables:    make(map[string]string),
		modules:      make(map[string]bool),
	}

	if err = parser.ParseFile(parser.ConfigRoot); err != nil {
		parser.Close()
		return nil, fmt.Errorf("could not parse webserver config: %v", err)
	}

	if err = parser.UpdateRuntimeVariables(); err != nil {
		return nil, err
	}

	// prepare the list of an active include paths, before modifications
	for k, v := range parser.LoadedPaths {
		dst := make([]string, len(v))
		copy(dst, v)
		parser.ExistingPaths[k] = dst
	}

	if hostRoot != "" && hostFiles != "" {
		vhostFilesPath := filepath.Join(hostRoot, hostFiles)

		if err = parser.ParseFile(vhostFilesPath); err != nil {
			return nil, err
		}
	}

	return &parser, nil
}

func getConfigListen(serverRoot, configRoot string) string {
	configPorts := filepath.Join(serverRoot, "ports.conf")

	if com.IsFile(configPorts) {
		return configPorts
	}

	return configRoot
}
