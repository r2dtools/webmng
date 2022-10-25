package options

import "github.com/r2dtools/webmng/pkg/options"

const (
	// Nginx root directory
	ServerRoot = "server_root"
	// Nginx binary path
	NginxBinPath = "nginx_bin_path"
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
	defaults[ServerRoot] = ""
	defaults[NginxBinPath] = ""

	return defaults
}
