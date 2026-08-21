package lua

import (
	"fmt"
	"strings"
)

// SyntaxError describes a single lex or parse error at a specific position.
type SyntaxError struct {
	Pos     Position
	Message string
}

// Error renders the error in "pos: message" form.
func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}

// SyntaxErrors aggregates multiple syntax errors from a single lex or parse.
// It satisfies the error interface so callers can return it directly.
type SyntaxErrors []*SyntaxError

// Error joins the individual error messages with newlines.
func (es SyntaxErrors) Error() string {
	msgs := make([]string, len(es))
	for i, e := range es {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "\n")
}
