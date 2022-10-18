package options

import "strings"

type Options struct {
	Params   map[string]string
	Defaults map[string]string
}

func (o Options) Get(name string) string {
	name = strings.ToLower(name)

	if o.Params == nil {
		o.Params = make(map[string]string)
	}

	if option, ok := o.Params[name]; ok {
		return option
	}

	if o.Defaults == nil {
		o.Defaults = make(map[string]string)
	}

	if def, ok := o.Defaults[name]; ok {
		return def
	}

	return ""
}
