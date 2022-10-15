package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckMinVersion(t *testing.T) {
	result, err := CheckMinVersion("2.4.8", "2.4.0")
	assert.Nilf(t, err, "failed to check min version %v:", err)
	assert.Equal(t, true, result)
	result, err = CheckMinVersion("2.4.8", "2.5.0")
	assert.Nilf(t, err, "failed to check min version %v:", err)
	assert.Equal(t, false, result)
}

func TestIsCommandExist(t *testing.T) {
	result := IsCommandExist("uname")
	assert.Equal(t, true, result)
	result = IsCommandExist("fakeCommand")
	assert.Equal(t, false, result)
}
