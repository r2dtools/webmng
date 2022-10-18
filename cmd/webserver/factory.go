package webserver

import (
	"fmt"

	"github.com/r2dtools/webmng/internal/apache"
	"github.com/r2dtools/webmng/internal/apache/apachectl"
	"github.com/r2dtools/webmng/internal/apache/apachesite"
	apacheoptions "github.com/r2dtools/webmng/internal/apache/options"
	"github.com/r2dtools/webmng/pkg/webserver"
)

func GetWebServerManager(code string, params map[string]string) (webserver.WebServerManagerInterface, error) {
	switch code {
	case webserver.Apache:
		options := apacheoptions.GetOptions(params)

		aCtl, err := apachectl.GetApacheCtl(options.Get(apacheoptions.ApacheCtl))
		if err != nil {
			return nil, err
		}

		aSite := apachesite.GetApacheSite(options.Get(apacheoptions.ApacheEnsite), options.Get(apacheoptions.ApacheDissite))
		parser, err := apache.GetParser(
			aCtl,
			"Httpd",
			options.Get(apacheoptions.ServerRoot),
			options.Get(apacheoptions.HostRoot),
			options.Get(apacheoptions.HostFiles),
		)
		if err != nil {
			return nil, err
		}

		apache, err := apache.GetApacheManager(aCtl, aSite, parser)
		if err != nil {
			return nil, err
		}

		return apache, nil
	default:
		return nil, fmt.Errorf("webserver %s is not supported", code)
	}
}
