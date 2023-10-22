package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/Xwudao/neter/pkg/json2go/stack"
	"github.com/Xwudao/neter/pkg/json2go/token"
)

type Scanner interface {
	More() bool
	Scan() (token.Token, string, error)
}

const eof = -1

type DefaultScanner struct {
	source string

	character     rune
	offset        int
	readingOffset int

	stack *stack.Stack[token.Token]
}

func New(s string) Scanner {
	scanner := new(DefaultScanner)
	scanner.source = s
	scanner.stack = stack.New[token.Token]()
	scanner.next()
	return scanner
}

func NewFromBytes(data []byte) Scanner {
	return New(string(data))
}

func (s *DefaultScanner) next() {
	if s.readingOffset >= len(s.source) {
		s.character = eof
		return
	}

	r, size := rune(s.source[s.readingOffset]), 1
	if r >= utf8.RuneSelf {
		r, size = utf8.DecodeRuneInString(s.source[s.readingOffset:])
	}

	s.character = r
	s.offset = s.readingOffset
	s.readingOffset += size
}

func (s *DefaultScanner) peek() byte {
	if s.readingOffset >= len(s.source) {
		return 0
	}
	return s.source[s.readingOffset]
}

func (s *DefaultScanner) skipWhitespace() {
	for s.character == ' ' || s.character == '\n' || s.character == '\r' || s.character == '\t' {
		s.next()
	}
}

func (s *DefaultScanner) scanString() (string, error) {
	s.next()
	start := s.offset

	for s.character != '"' {
		switch s.character {
		case eof:
			return "", fmt.Errorf("")
		case '\\':
			s.next()
			s.next()
		default:
			s.next()
		}
	}

	end := s.offset
	s.next()

	return s.source[start:end], nil
}

func (s *DefaultScanner) scan(target string) error {
	if s.source[s.offset:s.offset+len(target)] != target {
		return fmt.Errorf("fail to scan %s", target)
	}

	s.readingOffset += len(target)
	s.next()
	return nil
}

func (s *DefaultScanner) More() bool {
	s.skipWhitespace()
	if s.character == ',' {
		s.next()
	}
	s.skipWhitespace()

	return s.character != eof && s.character != '}' && s.character != ']'
}

func (s *DefaultScanner) Scan() (token.Token, string, error) {
scanAgain:
	s.skipWhitespace()

	switch s.character {
	case '{':
		s.stack.Push(token.LeftBrace)
		s.next()
		return token.LeftBrace, "", nil
	case ':':
		s.next()
		goto scanAgain
	case '}':
		if s.stack.IsEmpty() {
			return token.Null, "", fmt.Errorf("brackets do not match")
		} else {
			if s.stack.Top() == token.LeftBrace {
				s.stack.Pop()
			} else {
				return token.Illegal, "", fmt.Errorf("expecting '%s', got '%s", token.LeftBrace, s.stack.Top())
			}
		}
		s.next()
		return token.RightBrace, "", nil
	case '[':
		s.stack.Push(token.LeftBracket)
		s.next()
		return token.LeftBracket, "", nil
	case ']':
		if s.stack.IsEmpty() {
			return token.Illegal, "", fmt.Errorf("brackets do not match")
		} else {
			if s.stack.Top() == token.LeftBracket {
				s.stack.Pop()
			} else {
				return token.Illegal, "", fmt.Errorf("expecting '%s', got '%s", token.LeftBracket, s.stack.Top())
			}
		}
		s.next()
		return token.RightBracket, "", nil
	case '"':
		literal, err := s.scanString()
		return token.String, literal, err
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		t := token.Int
		s.next()
		for '0' <= s.character && s.character <= '9' {
			s.next()
		}
		if s.character == '.' {
			t = token.Float
			s.next()
			for '0' <= s.character && s.character <= '9' {
				s.next()
			}
		}
		if s.character == 'e' || s.character == 'E' {
			t = token.Float
			s.next()
			if s.character == '+' || s.character == '-' {
				s.next()
			}
			for '0' <= s.character && s.character <= '9' {
				s.next()
			}
		}
		return t, "", nil
	case 't':
		return token.Bool, "", s.scan("true")
	case 'f':
		return token.Bool, "", s.scan("false")
	case 'n':
		return token.Null, "", s.scan("null")
	case ',':
		s.next()
		goto scanAgain
	case eof:
		return token.EOF, "", nil
	default:
		return token.Illegal, "", fmt.Errorf("illegal character %c", s.character)
	}
}

type StandardScanner struct {
	*json.Decoder
}

func NewStandard(s string) Scanner {
	scanner := StandardScanner{
		Decoder: json.NewDecoder(bytes.NewBufferString(s)),
	}
	scanner.UseNumber()
	return scanner
}

func NewStandardFromBytes(data []byte) Scanner {
	scanner := StandardScanner{
		Decoder: json.NewDecoder(bytes.NewBuffer(data)),
	}
	scanner.UseNumber()
	return scanner
}

func (s StandardScanner) Scan() (token.Token, string, error) {
	t, err := s.Decoder.Token()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return token.EOF, "", nil
		}
		return token.Illegal, "", err
	}

	switch x := t.(type) {
	case json.Delim:
		switch x {
		case '{':
			return token.LeftBrace, "", nil
		case '}':
			return token.RightBrace, "", nil
		case '[':
			return token.LeftBracket, "", nil
		case ']':
			return token.RightBracket, "", nil
		default:
			return token.Illegal, "", fmt.Errorf("invalid delim %s", x)
		}
	case bool:
		return token.Bool, "", nil
	case json.Number:
		n := x.String()

		if n == "" {
			return token.Float, "", nil
		}
		if !(n[0] == '+' || n[0] == '-' || ('0' <= n[0] && n[0] <= '9')) {
			return token.Float, "", nil
		}
		for i := 1; i < len(n); i++ {
			if !('0' <= n[i] && n[i] <= '9') {
				return token.Float, "", nil
			}
		}

		return token.Int, "", nil
	case string:
		return token.String, "", nil
	case nil:
		return token.Null, "", nil
	default:
		return token.Illegal, "", fmt.Errorf("unexpected type")
	}
}
