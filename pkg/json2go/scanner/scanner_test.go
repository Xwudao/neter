package scanner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Xwudao/neter/pkg/json2go/token"
)

func getTokens(scanner Scanner) (tokens []token.Token) {
	t, _, err := scanner.Scan()
	if err != nil {
		return
	}

	for t != token.EOF {
		tokens = append(tokens, t)
		t, _, err = scanner.Scan()
		if err != nil {
			return
		}
	}

	return
}

type Tokens []token.Token

func (tokens Tokens) String() string {
	var b strings.Builder

	if len(tokens) != 0 {
		b.WriteString(tokens[0].String())
	}
	for i := 1; i < len(tokens); i++ {
		b.WriteByte(',')
		b.WriteByte(' ')
		b.WriteString(tokens[i].String())
	}

	return b.String()
}

func testScanner(t *testing.T, newScanner func(s string) Scanner) {
	json := `[
  {
    "string": "中文",
    "int": 123,
    "float": 1.0,
    "bool": false,
    "null": null,
    "array": [
      "\\\"", -1, 0, 1, 1.0, 1e3, 1e-3, true, false, null
    ]
  }
]`
	assert.Equal(
		t,
		Tokens([]token.Token{
			token.LeftBracket,
			token.LeftBrace,
			token.String, token.String,
			token.String, token.Int,
			token.String, token.Float,
			token.String, token.Bool,
			token.String, token.Null,
			token.String, token.LeftBracket,
			token.String, token.Int, token.Int, token.Int, token.Float, token.Float, token.Float, token.Bool, token.Bool, token.Null,
			token.RightBracket,
			token.RightBrace,
			token.RightBracket,
		}).String(),
		Tokens(getTokens(newScanner(json))).String(),
	)
}

func TestDefaultScanner(t *testing.T) {
	testScanner(t, New)
}

func TestStandardScanner(t *testing.T) {
	testScanner(t, NewStandard)
}
