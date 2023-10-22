package json2go

import (
	"bytes"

	"github.com/dave/jennifer/jen"
)

// RenderCode renders jen.Code.
func RenderCode(code jen.Code) (string, error) {
	buffer := new(bytes.Buffer)
	if err := jen.Add(code).Render(buffer); err != nil {
		return "", err
	}

	return buffer.String(), nil
}
