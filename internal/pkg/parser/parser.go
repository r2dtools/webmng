package parser

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/unknwon/com"
	"honnef.co/go/augeas"
)

type BaseParser struct {
	Augeas augeas.Augeas
	LensModule,
	ServerRoot,
	ConfigRoot,
	HostRoot string
	LoadedPaths   map[string][]string
	ExistingPaths map[string][]string
}

// ParseFile parses file with Auegause
func (p *BaseParser) ParseFile(fPath string) error {
	useNew, removeOld := p.checkPath(fPath)
	if !useNew {
		return nil
	}

	includedPaths, err := p.Augeas.Match(fmt.Sprintf("/augeas/load/%s['%s' =~ glob(incl)]", p.LensModule, fPath))
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

// GetUnsavedFiles returns unsaved paths
func (p *BaseParser) GetUnsavedFiles() ([]string, error) {
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

	saveErr := p.GetAugeasError(nil)
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

// GetAugeasError return Augeas errors
func (p *BaseParser) GetAugeasError(errorsToExclude []string) error {
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

// Close closes the Parser instance and frees any storage associated with it.
func (p *BaseParser) Close() {
	if p != nil {
		p.Augeas.Close()
	}
}

// ConvertPathFromServerRootToAbs convert path to absolute if it is relative to server root
func (p *BaseParser) ConvertPathFromServerRootToAbs(path string) string {
	path = strings.Trim(path, "'\"")

	if strings.HasPrefix(path, "/") {
		path = filepath.Clean(path)
	} else {
		path = filepath.Clean(filepath.Join(p.ServerRoot, path))
	}

	return path
}

// IsFilenameExistInLoadedPaths checks if the file path is parsed by current Augeas parser config
func (p *BaseParser) IsFilenameExistInLoadedPaths(filename string) bool {
	return p.isFilenameExistInPaths(filename, p.LoadedPaths)
}

// IsFilenameExistInOriginalPaths checks if the file path is parsed by existing Apache config
func (p *BaseParser) IsFilenameExistInOriginalPaths(filename string) bool {
	return p.isFilenameExistInPaths(filename, p.ExistingPaths)
}

// GetRootAugPath returns Augeas path of the root configuration
func (p *BaseParser) GetRootAugPath() (string, error) {
	return p.GetAugPath(p.ConfigRoot), nil
}

// GetAugPath returns Augeas path for the file full path
func (p *BaseParser) GetAugPath(fullPath string) string {
	return fmt.Sprintf("/files/%s", fullPath)
}

func (p *BaseParser) isFilenameExistInPaths(filename string, paths map[string][]string) bool {
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
func (p *BaseParser) checkPath(fPath string) (useNew, removeOld bool) {
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
func (p *BaseParser) removeTransform(fPath string) {
	dirnameToRemove := filepath.Dir(fPath)
	existedFilenames := p.LoadedPaths[dirnameToRemove]

	for _, filename := range existedFilenames {
		pathToRemove := filepath.Join(dirnameToRemove, filename)
		includesToRemove, err := p.Augeas.Match(fmt.Sprintf("/augeas/load/%s/incl [. ='%s']", p.LensModule, pathToRemove))

		if err == nil && len(includesToRemove) > 0 {
			p.Augeas.Remove(includesToRemove[0])
		}
	}

	delete(p.LoadedPaths, dirnameToRemove)
}

// Add a transform to Augeas
func (p *BaseParser) addTransform(fPath string) error {
	lastInclude, err := p.Augeas.Match(fmt.Sprintf("/augeas/load/%s/incl [last()]", p.LensModule))
	if err != nil {
		return err
	}

	dirnameToAdd := filepath.Dir(fPath)
	fileNameToAdd := filepath.Base(fPath)

	if len(lastInclude) > 0 {
		p.Augeas.Insert(lastInclude[0], "incl", false)
		p.Augeas.Set(fmt.Sprintf("/augeas/load/%s/incl[last()]", p.LensModule), fPath)
	} else {
		p.Augeas.Set(fmt.Sprintf("/augeas/load/%s/lens", p.LensModule), p.LensModule+".lns")
		p.Augeas.Set(fmt.Sprintf("/augeas/load/%s/incl", p.LensModule), fPath)
	}

	paths := append(p.LoadedPaths[dirnameToAdd], fileNameToAdd)
	p.LoadedPaths[dirnameToAdd] = paths

	return nil
}
