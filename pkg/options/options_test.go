package options

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	params := map[string]string{
		"param1": "value1",
	}
	defaults := map[string]string{
		"param1": "value1_1",
		"param2": "value2",
	}

	options := Options{
		Params:   params,
		Defaults: defaults,
	}

	assert.Equal(t, "value1", options.Get("param1"))
	assert.Equal(t, "value2", options.Get("param2"))
	assert.Equal(t, "", options.Get("param3"))
}
