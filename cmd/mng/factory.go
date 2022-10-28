package mng

import (
	"fmt"

	"github.com/r2dtools/webmng/internal/apache"
	"github.com/r2dtools/webmng/internal/nginx"
	"github.com/r2dtools/webmng/pkg/logger"
	"github.com/r2dtools/webmng/pkg/webserver"
)

func GetWebServerManager(code string, params map[string]string) (webserver.WebServerManagerInterface, error) {
	logger := logger.NilLogger{}

	switch code {
	case webserver.Apache:
		return apache.GetApacheManager(params, logger)
	case webserver.Nginx:
		return nginx.GetNginxManager(params, logger)
	default:
		return nil, fmt.Errorf("webserver %s is not supported", code)
	}
}
