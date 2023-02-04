package options

import "github.com/r2dtools/webmng/pkg/options"
import webserverOptions "github.com/r2dtools/webmng/pkg/webserver/options"

const (
	// Nginx root directory
	ServerRoot = "server_root"
)

func GetOptions(params map[string]string) options.Options {
	if params == nil {
		params = make(map[string]string)
	}
	return options.Options{Defaults: GetDefaults(), Params: params}
}

// GetDefaults returns Nginx manager default options
func GetDefaults() map[string]string {
	defaults := make(map[string]string)
	defaults[ServerRoot] = "/etc/nginx"

	wsOptions := webserverOptions.GetDefaults()

	for key, value := range wsOptions {
		defaults[key] = value
	}

	return defaults
}
