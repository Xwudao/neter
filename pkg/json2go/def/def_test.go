package def

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrom(t *testing.T) {
	stmt, err := From("{}").Declare("T")
	if err != nil {
		t.Fatal(err)
	}

	buffer := new(bytes.Buffer)
	if err = stmt.Render(buffer); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "type T map[string]any", buffer.String())
}
