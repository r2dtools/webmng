package apache

import (
	"fmt"

	"github.com/r2dtools/webmng/internal/apache/apachectl"
	"github.com/r2dtools/webmng/internal/apache/apachesite"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/utils"
)

const (
	minApacheVersion = "2.4.0"
)

var serverRootPaths = []string{"/etc/httpd", "/etc/apache2"}
var configFiles = []string{"apache2.conf", "httpd.conf", "conf/httpd.conf"}

type ApacheManager struct {
	apacheCtl     *apachectl.ApacheCtl
	apacheSite    *apachesite.ApacheSite
	parser        *Parser
	logger        logger.LoggerInterface
	apacheVersion string
}

func GetApacheManager(apacheCtl *apachectl.ApacheCtl, apacheSite *apachesite.ApacheSite, parser *Parser) (*ApacheManager, error) {
	version, err := apacheCtl.GetVersion()
	if err != nil {
		return nil, err
	}

	isVersionSupported, err := utils.CheckMinVersion(version, minApacheVersion)
	if err != nil {
		return nil, err
	}

	if !isVersionSupported {
		return nil, fmt.Errorf("current apache version '%s' is not supported. Minimal supported version is '%s'", version, minApacheVersion)
	}

	// Test apache configuration before creating ApacheConfigurator
	if err = apacheCtl.TestConfiguration(); err != nil {
		return nil, err
	}

	manager := ApacheManager{
		apacheCtl:     apacheCtl,
		apacheSite:    apacheSite,
		parser:        parser,
		logger:        logger.NilLogger{},
		apacheVersion: version,
	}

	return &manager, nil
}

func (m *ApacheManager) SetLogger(logger logger.LoggerInterface) {
	m.logger = logger
}
