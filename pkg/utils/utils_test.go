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

func TestGetCommandPath(t *testing.T) {
	path, err := GetCommandBinPath("uname")
	assert.Nil(t, err)
	assert.Equal(t, "/usr/bin/uname", path)
	_, err = GetCommandBinPath("fakeCommand")
	assert.NotNil(t, err)
}
