package rawparser

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	parser, err := GetRawParser()
	assert.Nilf(t, err, "could not create parser: %v", err)

	parsedConfig, err := parser.Parse("../../../test/nginx/unit/nginx.conf")
	assert.Nilf(t, err, "could not parse config: %v", err)

	expectedData := &Config{}
	data, err := os.ReadFile("../../../test/nginx/unit/nginx.conf.json")
	assert.Nilf(t, err, "could not read file with expected data: %v", err)

	err = json.Unmarshal(data, expectedData)
	assert.Nilf(t, err, "could not decode expected data: %v", err)

	assert.Equal(t, expectedData, parsedConfig, "parsed data is invalid")
}
