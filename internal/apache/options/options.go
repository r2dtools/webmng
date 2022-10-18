package options

import "github.com/r2dtools/webmng/pkg/options"

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
	// ApacheEnsite is a command for a2ensite command or a pth to a2ensite bin
	ApacheEnsite = "apache_ensite"
	// ApacheDissite is a command for a2dissite command or a pth to a2dissite bin
	ApacheDissite = "apache_dissite"
)

func GetOptions(params map[string]string) options.Options {
	return options.Options{Defaults: GetDefaults(), Params: params}
}

// GetDefaults returns ApacheConfigurator default options
func GetDefaults() map[string]string {
	defaults := make(map[string]string)
	defaults[ServerRoot] = ""
	defaults[HostRoot] = ""
	defaults[HostFiles] = "*"
	defaults[ApacheCtl] = ""
	defaults[SslVhostlExt] = "-ssl.conf"
	defaults[ApacheEnsite] = "a2ensite"
	defaults[ApacheDissite] = "a2dissite"

	return defaults
}
