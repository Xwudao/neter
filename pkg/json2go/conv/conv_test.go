package conv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToCamelCase(t *testing.T) {
	assert.Equal(t, "", ToCamelCase(""))
	assert.Equal(t, "T", ToCamelCase("t"))
	assert.Equal(t, "To", ToCamelCase("to"))
	assert.Equal(t, "ToCamelCase", ToCamelCase("to camel  case  "))
	assert.Equal(t, "ToCamelCase", ToCamelCase("to_camel__case"))
	assert.Equal(t, "ToCamelCase", ToCamelCase("to-camel--case"))
	assert.Equal(t, "ToCamelCase", ToCamelCase("to_camel -case"))
	assert.Equal(t, "JsonToGo", ToCamelCase("JSON to Go"))

	converter := NewDefaultCamelCaseConverter([]string{"ID", "JSON"})
	assert.Equal(t, "ID", converter.ToCamelCase("id"))
	assert.Equal(t, "JSON", converter.ToCamelCase("json"))
	assert.Equal(t, "JSONToGo", converter.ToCamelCase("JSON to Go"))
}

func TestToSnakeCase(t *testing.T) {
	assert.Equal(t, "", ToSnakeCase(""))
	assert.Equal(t, "json_to_go", ToSnakeCase("JSONToGo"))
}
