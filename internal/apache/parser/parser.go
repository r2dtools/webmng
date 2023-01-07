package parser

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/huandu/xstrings"
	"github.com/r2dtools/webmng/internal/apache/apachectl"
	"github.com/r2dtools/webmng/internal/apache/utils"
	"github.com/r2dtools/webmng/pkg/aug"
	commonutils "github.com/r2dtools/webmng/pkg/utils"
	"github.com/r2dtools/webmng/pkg/webserver/reverter"
	"github.com/unknwon/com"
	"honnef.co/go/augeas"
)

const (
	argVarRegex = `\$\{[^ \}]*}`
)

var configFiles = []string{"apache2.conf", "httpd.conf", "conf/httpd.conf"}
var fnMatchChars = []string{"*", "?", "\\", "[", "]"}

type directiveFilter struct {
	Name  string
	Value []string
}

type Parser struct {
	Augeas      augeas.Augeas
	LoadedPaths map[string][]string
	ServerRoot,
	ConfigListen,
	ConfigRoot,
	lensModule,
	hostRoot string
	existingPaths map[string][]string
	apachectl     apachectl.ApacheCtl
	variables     map[string]string
	modules       map[string]bool
}

// FindDirective finds directive in configuration
// directive - directive to look for
// arg - directive value. If empty string then all directives should be considrered
// start - Augeas path that should be used to begin looking for the directive
// exclude - whether or not to exclude directives based on variables and enabled modules
func (p *Parser) FindDirective(directive, arg, start string, exclude bool) ([]string, error) {
	if start == "" {
		start = aug.GetAugPath(p.ConfigRoot)
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

// AddDirectiveToIfModSSL adds directive to the end of the file given by augConfPath within IfModule ssl block
func (p *Parser) AddDirectiveToIfModSSL(augConfPath string, directive string, args []string) error {
	ifModPath, err := p.getIfModule(augConfPath, "mod_ssl.c", false)

	if err != nil {
		return err
	}

	if err = p.Augeas.Insert(ifModPath+"arg", "directive", false); err != nil {
		return fmt.Errorf("could not insert directive within IfModule SSL block: %v", err)
	}

	nPath := ifModPath + "directive[1]"

	if err = p.Augeas.Set(nPath, directive); err != nil {
		return fmt.Errorf("could not set directive value within IfModule SSL block: %v", err)
	}

	if len(args) == 0 {
		if err = p.Augeas.Set(nPath+"/arg", args[0]); err != nil {
			return fmt.Errorf("could not set directive argument within IfModule SSL block: %v", err)
		}
	} else {
		for i, arg := range args {
			if err = p.Augeas.Set(fmt.Sprintf("%s/arg[%d]", nPath, i+1), arg); err != nil {
				return fmt.Errorf("could not set directive argument within IfModule SSL block: %v", err)
			}
		}
	}

	return nil
}

// AddInclude adds Include directive for a configuration file
func (p *Parser) AddInclude(mainConfigPath string, inclPath string) error {
	matches, err := p.FindDirective("Include", inclPath, "", true)

	if err != nil {
		return fmt.Errorf("failed searching 'Include' directive in the config '%s': %v", mainConfigPath, err)
	}

	if len(matches) == 0 {
		if err = p.AddDirective(aug.GetAugPath(mainConfigPath), "Include", []string{inclPath}); err != nil {
			return fmt.Errorf("could not add 'Include' directive to config '%s': %v", mainConfigPath, err)
		}
	}

	newDir := filepath.Dir(inclPath)
	newFile := filepath.Base(inclPath)

	if _, ok := p.existingPaths[newDir]; !ok {
		p.existingPaths[newDir] = make([]string, 0)
	}

	p.existingPaths[newDir] = append(p.existingPaths[newDir], newFile)

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

// UpdateRuntimeVariables Updates Includes, Defines and Modules from httpd config dump data
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

// ResetModules resets the loaded modules list
func (p *Parser) ResetModules() error {
	p.modules = make(map[string]bool)
	if err := p.updateModules(); err != nil {
		return err
	}

	// p.ParseModules() TODO: apache config should be also parsed for LoadModule directive
	return nil
}

func (p *Parser) ModuleExists(name string) bool {
	_, ok := p.modules[name]

	return ok
}

// IsFilenameExistInLoadedPaths checks if the file path is parsed by current Augeas parser config
func (p *Parser) IsFilenameExistInLoadedPaths(filename string) bool {
	return p.isFilenameExistInPaths(filename, p.LoadedPaths)
}

// IsFilenameExistInOriginalPaths checks if the file path is parsed by existing Apache config
func (p *Parser) IsFilenameExistInOriginalPaths(filename string) bool {
	return p.isFilenameExistInPaths(filename, p.existingPaths)
}

// Close closes the Parser instance and frees any storage associated with it.
func (p *Parser) Close() {
	if p != nil {
		p.Augeas.Close()
	}
}

func (p *Parser) Save(reverter reverter.Reverter) error {
	unsavedFiles, err := p.getUnsavedFiles()

	if err != nil {
		return err
	}

	if len(unsavedFiles) == 0 {
		return nil
	}

	if reverter != nil {
		if err = reverter.BackupFiles(unsavedFiles); err != nil {
			return err
		}
	}

	if err = p.Augeas.Save(); err != nil {
		return err
	}

	for _, unsavedFile := range unsavedFiles {
		p.Augeas.Remove(fmt.Sprintf("/files/%s", unsavedFile))
	}

	if err = p.Augeas.Load(); err != nil {
		return err
	}

	return nil
}

// getIfModule returns the path to <IfModule mod> and creates one if it does not exist
func (p *Parser) getIfModule(augConfPath string, mod string, begining bool) (string, error) {
	ifMods, err := p.Augeas.Match(fmt.Sprintf("%s/IfModule/*[self::arg='%s']", augConfPath, mod))

	if err != nil {
		return "", fmt.Errorf("could not get IfModule directive: %v", err)
	}

	if len(ifMods) == 0 {
		return p.createIfModule(augConfPath, mod, begining)
	}

	path, _, _ := xstrings.LastPartition(ifMods[0], "arg")

	return path, nil
}

// CreateIfModule creates a new <IfMod mod> and returns its path
func (p *Parser) createIfModule(augConfPath string, mod string, begining bool) (string, error) {
	var argPath, retPath string
	var err error

	if begining {
		argPath = fmt.Sprintf("%s/IfModule[1]/arg", augConfPath)

		if err = p.Augeas.Insert(fmt.Sprintf("%s/directive[1]", augConfPath), "IfModule", true); err != nil {
			return "", fmt.Errorf("could not insert IfModule directive: %v", err)
		}

		retPath = fmt.Sprintf("%s/IfModule[1]/", augConfPath)
	} else {
		path := fmt.Sprintf("%s/IfModule[last() + 1]", augConfPath)
		argPath = fmt.Sprintf("%s/IfModule[last()]/arg", augConfPath)

		if err = p.Augeas.Set(path, ""); err != nil {
			return "", fmt.Errorf("could not set IfModule directive: %v", err)
		}

		retPath = fmt.Sprintf("%s/IfModule[last()]/", augConfPath)
	}

	if err = p.Augeas.Set(argPath, mod); err != nil {
		return "", fmt.Errorf("could not set argument %s: %v", mod, err)
	}

	return retPath, nil
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
	arg = p.convertPathFromServerRootToAbs(arg)
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

	return aug.GetAugPath(arg), nil
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

func (p *Parser) ParseFile(fPath string) error {
	useNew, removeOld := p.checkPath(fPath)
	if !useNew {
		return nil
	}

	includedPaths, err := p.Augeas.Match(fmt.Sprintf("/augeas/load/%s['%s' =~ glob(incl)]", p.lensModule, fPath))

	if err != nil {
		return err
	}

	if len(includedPaths) == 0 {
		if removeOld {
			p.removeTransform(fPath)
		}

		p.addTransform(fPath)

		if err = p.Augeas.Load(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) getUnsavedFiles() ([]string, error) {
	// Current save method
	saveMethod, err := p.Augeas.Get("/augeas/save")

	if err != nil {
		return nil, err
	}

	// See https://github.com/hercules-team/augeas/wiki/Change-how-files-are-saved
	if err = p.Augeas.Set("/augeas/save", "noop"); err != nil {
		return nil, err
	}

	if err = p.Augeas.Save(); err != nil {
		p.Augeas.Set("/augeas/save", saveMethod)
		return nil, err
	}

	saveErr := p.getAugeasError(nil)
	p.Augeas.Set("/augeas/save", saveMethod)

	if saveErr != nil {
		return nil, saveErr
	}

	var paths []string
	matchesToSave, err := p.Augeas.Match("/augeas/events/saved")
	if err != nil {
		return nil, err
	}

	for _, matchToSave := range matchesToSave {
		pathToSave, err := p.Augeas.Get(matchToSave)
		if err != nil {
			return nil, err
		}

		paths = append(paths, pathToSave[6:])
	}

	return paths, nil
}

func (p *Parser) getAugeasError(errorsToExclude []string) error {
	newErrors, err := p.Augeas.Match("/augeas//error")
	if err != nil {
		return fmt.Errorf("could not get augeas errors: %v", err)
	}

	if len(newErrors) == 0 {
		return nil
	}

	var rootErrors []string

	for _, newError := range newErrors {
		if !com.IsSliceContainsStr(errorsToExclude, newError) {
			rootErrors = append(rootErrors, newError)
		}
	}

	if len(rootErrors) == 0 {
		return nil
	}

	var detailedRootErrors []string

	for _, rError := range rootErrors {
		details, _ := p.Augeas.Get(rError + "/message")

		if details == "" {
			detailedRootErrors = append(detailedRootErrors, rError)
		} else {
			detailedRootErrors = append(detailedRootErrors, fmt.Sprintf("%s: %s", rError, details))
		}
	}

	return fmt.Errorf(strings.Join(detailedRootErrors, ", "))
}

func (p *Parser) convertPathFromServerRootToAbs(path string) string {
	path = strings.Trim(path, "'\"")

	if strings.HasPrefix(path, "/") {
		path = filepath.Clean(path)
	} else {
		path = filepath.Clean(filepath.Join(p.ServerRoot, path))
	}

	return path
}

func (p *Parser) isFilenameExistInPaths(filename string, paths map[string][]string) bool {
	for dir, fNames := range paths {
		for _, fName := range fNames {
			isMatch, err := path.Match(path.Join(dir, fName), filename)

			if err != nil {
				continue
			}

			if isMatch {
				return true
			}
		}
	}

	return false
}

// Checks if fPath exists in augeas paths
// We should try to append a new fPath to augeas
// parser paths, and/or remove the old one with more
// narrow matching.
func (p *Parser) checkPath(fPath string) (useNew, removeOld bool) {
	filename := filepath.Base(fPath)
	dirname := filepath.Dir(fPath)
	exisingMatches, ok := p.LoadedPaths[dirname]

	if !ok {
		return true, false
	}

	removeOld = filename == "*"

	for _, existingMatch := range exisingMatches {
		if existingMatch == "*" {
			return false, removeOld
		}
	}

	return true, removeOld
}

// Remove a transform from Augeas
func (p *Parser) removeTransform(fPath string) {
	dirnameToRemove := filepath.Dir(fPath)
	existedFilenames := p.LoadedPaths[dirnameToRemove]

	for _, filename := range existedFilenames {
		pathToRemove := filepath.Join(dirnameToRemove, filename)
		includesToRemove, err := p.Augeas.Match(fmt.Sprintf("/augeas/load/%s/incl [. ='%s']", p.lensModule, pathToRemove))

		if err == nil && len(includesToRemove) > 0 {
			p.Augeas.Remove(includesToRemove[0])
		}
	}

	delete(p.LoadedPaths, dirnameToRemove)
}

// Add a transform to Augeas
func (p *Parser) addTransform(fPath string) error {
	lastInclude, err := p.Augeas.Match(fmt.Sprintf("/augeas/load/%s/incl [last()]", p.lensModule))

	if err != nil {
		return err
	}

	dirnameToAdd := filepath.Dir(fPath)
	fileNameToAdd := filepath.Base(fPath)

	if len(lastInclude) > 0 {
		p.Augeas.Insert(lastInclude[0], "incl", false)
		p.Augeas.Set(fmt.Sprintf("/augeas/load/%s/incl[last()]", p.lensModule), fPath)
	} else {
		p.Augeas.Set(fmt.Sprintf("/augeas/load/%s/lens", p.lensModule), p.lensModule+".lns")
		p.Augeas.Set(fmt.Sprintf("/augeas/load/%s/incl", p.lensModule), fPath)
	}

	paths := append(p.LoadedPaths[dirnameToAdd], fileNameToAdd)
	p.LoadedPaths[dirnameToAdd] = paths

	return nil
}

func GetParser(apachectl apachectl.ApacheCtl, lensModule, serverRoot, hostRoot, hostFiles string) (*Parser, error) {
	var err error

	if hostRoot != "" {
		hostRoot, err = filepath.Abs(hostRoot)

		if err != nil {
			return nil, err
		}
	}

	// try to detect apache root config file path (ex. /etc/apache2/apache2.conf), ports.conf file path
	configRoot, err := commonutils.FindAnyFilesInDirectory(serverRoot, configFiles)

	if err != nil {
		return nil, err
	}

	configListen := getConfigListen(serverRoot, configRoot)
	aug, err := augeas.New("/", "", augeas.NoLoad|augeas.NoModlAutoload|augeas.EnableSpan)

	if err != nil {
		return nil, err
	}

	parser := Parser{
		Augeas:        aug,
		ServerRoot:    serverRoot,
		ConfigListen:  configListen,
		ConfigRoot:    configRoot,
		hostRoot:      hostRoot,
		existingPaths: make(map[string][]string),
		LoadedPaths:   make(map[string][]string),
		lensModule:    lensModule,
		apachectl:     apachectl,
		variables:     make(map[string]string),
		modules:       make(map[string]bool),
	}

	if err = parser.ParseFile(parser.ConfigRoot); err != nil {
		parser.Close()
		return nil, fmt.Errorf("could not parse webserver config: %v", err)
	}

	// load all variables, modules and parse all includes
	if err = parser.UpdateRuntimeVariables(); err != nil {
		return nil, err
	}

	// prepare the list of an active include paths, before modifications
	for k, v := range parser.LoadedPaths {
		dst := make([]string, len(v))
		copy(dst, v)
		parser.existingPaths[k] = dst
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
