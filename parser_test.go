package lua

import "testing"

func TestParseEmpty(t *testing.T) {
	chunk, err := Parse("t.lua", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chunk == nil || chunk.Block == nil {
		t.Fatal("expected non-nil chunk and block")
	}
	if len(chunk.Block.Statements) != 0 {
		t.Errorf("expected 0 statements, got %d", len(chunk.Block.Statements))
	}
}

func TestParseCommentsOnly(t *testing.T) {
	src := []byte("-- just a comment\n--[[ block comment ]]\n")
	chunk, err := Parse("t.lua", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chunk == nil || chunk.Block == nil {
		t.Fatal("expected non-nil chunk and block")
	}
}

func TestParsePropagatesLexerErrors(t *testing.T) {
	// Unterminated string should surface as a Parse error even though the
	// parser itself does not yet consume statements.
	_, err := Parse("t.lua", []byte(`"unterminated`))
	if err == nil {
		t.Fatal("expected error from unterminated string, got nil")
	}
}
