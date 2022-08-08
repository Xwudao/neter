package filex

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFiles(t *testing.T) {
	files, err := LoadFiles(".", func(s string) bool {
		fmt.Println(s)
		return true
	})
	assert.Nil(t, err)
	assert.Equal(t, true, len(files) > 0)
}
