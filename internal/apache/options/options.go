package options

import "github.com/r2dtools/webmng/pkg/options"
import webserverOptions "github.com/r2dtools/webmng/pkg/webserver/options"

const (
	// HostRoot is apache virtual host root directory
	HostRoot = "host_root"
	// ServerRoot is apache root directory
	ServerRoot = "server_root"
	// VhostFiles specifies config files for virtual host that will be used. By default all config files are used.
	HostFiles = "host_files"
	// ApacheCtl is a command for apache2ctl execution or a path to apache2ctl bin
	ApacheCtl = "ctl"
	// SslVhostlExt postfix for config files of created SSL virtual hosts
	SslVhostlExt = "ssl_vhost_ext"
)

func GetOptions(params map[string]string) options.Options {
	if params == nil {
		params = make(map[string]string)
	}
	return options.Options{Defaults: GetDefaults(), Params: params}
}

// GetDefaults returns Apache manager default options
func GetDefaults() map[string]string {
	defaults := make(map[string]string)
	defaults[ServerRoot] = ""
	defaults[HostRoot] = ""
	defaults[HostFiles] = "*"
	defaults[ApacheCtl] = ""
	defaults[SslVhostlExt] = "-ssl.conf"

	wsOptions := webserverOptions.GetDefaults()

	for key, value := range wsOptions {
		defaults[key] = value
	}

	return defaults
}
