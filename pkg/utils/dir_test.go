package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindModPath(t *testing.T) {
	modPath, err := FindModPath(3)
	assert.Nil(t, err)

	t.Logf("path: %s\n", modPath)
}
