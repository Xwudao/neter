package token

import "strconv"

type Token int8

const (
	Illegal      Token = iota
	LeftBrace          // '{'
	RightBrace         // '}'
	LeftBracket        // '['
	RightBracket       // ']'
	String
	Int
	Float
	Bool
	Null
	EOF
)

var tokens = [...]string{
	Illegal:      "illegal",
	LeftBrace:    "{",
	RightBrace:   "}",
	LeftBracket:  "[",
	RightBracket: "]",
	String:       "string",
	Int:          "int",
	Float:        "float",
	Bool:         "bool",
	Null:         "null",
	EOF:          "EOF",
}

func (token Token) String() string {
	if 0 <= token && token < Token(len(tokens)) && tokens[token] != "" {
		return tokens[token]
	}
	return "token(" + strconv.Itoa(int(token)) + ")"
}
