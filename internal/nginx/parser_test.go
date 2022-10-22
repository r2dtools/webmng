package nginx

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	parser, err := GetParser("../../test/nginx/nginx-simple.conf")
	assert.Nilf(t, err, "could not create parser: %v", err)

	parsedConfigs, err := parser.Parse()
	assert.Nilf(t, err, "could not parse config: %v", err)

	expectedData := make(map[string]*Config, 0)
	data, err := os.ReadFile("../../test/nginx/nginx-simple.conf.json")
	assert.Nilf(t, err, "could not read file with expected data: %v", err)

	err = json.Unmarshal(data, &expectedData)
	assert.Nilf(t, err, "could not decode expected data: %v", err)

	assert.Equal(t, expectedData, parsedConfigs, "parsed data is invalid")
}
