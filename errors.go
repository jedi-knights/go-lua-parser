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

// Error joins the individual error messages with newlines. An empty
// SyntaxErrors is not a valid error value (Parse's finalize() returns
// untyped nil in that case, never an empty slice) — but if a caller
// constructs one manually and asks for its message, return a placeholder
// rather than the empty string that would violate the error interface's
// non-empty contract.
func (es SyntaxErrors) Error() string {
	if len(es) == 0 {
		return "<no syntax errors>"
	}
	msgs := make([]string, len(es))
	for i, e := range es {
		if e == nil {
			// Defensive: Parse never inserts a nil into SyntaxErrors,
			// but a caller constructing SyntaxErrors{nil} manually
			// should not panic on Error().
			msgs[i] = "<nil syntax error>"
			continue
		}
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "\n")
}
