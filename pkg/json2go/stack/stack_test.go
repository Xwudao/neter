package stack

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func valid(s string) bool {
	stack := New[rune]()

	for _, c := range s {
		switch c {
		case '(', '[', '{', '<':
			stack.Push(c)
		case ')':
			if stack.IsEmpty() || stack.Pop() != '(' {
				return false
			}
		case ']':
			if stack.IsEmpty() || stack.Pop() != '[' {
				return false
			}
		case '}':
			if stack.IsEmpty() || stack.Pop() != '{' {
				return false
			}
		case '>':
			if stack.IsEmpty() || stack.Pop() != '<' {
				return false
			}
		}
	}

	return stack.IsEmpty()
}

func TestStack(t *testing.T) {
	assert.True(t, valid(""))
	assert.True(t, valid("( )"))
	assert.True(t, valid("[( )]"))
	assert.True(t, valid("[( )]{}"))
	assert.True(t, valid("<[( )]{}>"))

	assert.False(t, valid("("))
	assert.False(t, valid("([)]"))
	assert.False(t, valid("[{( )]"))
	assert.False(t, valid("{[( )]}>"))
}
