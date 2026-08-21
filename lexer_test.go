package lua

import (
	"testing"
)

func TestLexerKeywords(t *testing.T) {
	src := "and break do else elseif end false for function goto if in local nil not or repeat return then true until while"
	want := []TokenKind{
		TokenAnd, TokenBreak, TokenDo, TokenElse, TokenElseif, TokenEnd,
		TokenFalse, TokenFor, TokenFunction, TokenGoto, TokenIf, TokenIn,
		TokenLocal, TokenNil, TokenNot, TokenOr, TokenRepeat, TokenReturn,
		TokenThen, TokenTrue, TokenUntil, TokenWhile,
	}
	l := NewLexer("t.lua", []byte(src))
	for i, w := range want {
		got := l.Next()
		if got.Kind != w {
			t.Errorf("token %d: got %s, want %s", i, got.Kind, w)
		}
	}
	if got := l.Next(); got.Kind != TokenEOF {
		t.Errorf("trailing token: got %s (%q), want EOF", got.Kind, got.Value)
	}
}

func TestLexerIdentifiers(t *testing.T) {
	l := NewLexer("t.lua", []byte("foo_bar _x123 y"))
	want := []string{"foo_bar", "_x123", "y"}
	for i, w := range want {
		got := l.Next()
		if got.Kind != TokenIdent {
			t.Fatalf("token %d: kind = %s, want IDENT", i, got.Kind)
		}
		if got.Value != w {
			t.Errorf("token %d: value = %q, want %q", i, got.Value, w)
		}
	}
}

func TestLexerNumbers(t *testing.T) {
	cases := []struct {
		src, want string
	}{
		{"0", "0"},
		{"42", "42"},
		{"3.14", "3.14"},
		{"1e10", "1e10"},
		{"1.5e-3", "1.5e-3"},
		{"0xFF", "0xFF"},
		{"0x10a", "0x10a"},
		{".25", ".25"},
		{".5e2", ".5e2"},
	}
	for _, c := range cases {
		l := NewLexer("t.lua", []byte(c.src))
		got := l.Next()
		if got.Kind != TokenNumber {
			t.Errorf("%q: kind = %s, want NUMBER", c.src, got.Kind)
			continue
		}
		if got.Value != c.want {
			t.Errorf("%q: value = %q, want %q", c.src, got.Value, c.want)
		}
	}
}

func TestLexerShortStrings(t *testing.T) {
	cases := []struct {
		src, want string
	}{
		{`"hello"`, "hello"},
		{`'world'`, "world"},
		{`"a\nb"`, "a\nb"},
		{`"tab\there"`, "tab\there"},
		{`"quote\""`, `quote"`},
		{`"\65"`, "A"},
		{`"\\"`, `\`},
	}
	for _, c := range cases {
		l := NewLexer("t.lua", []byte(c.src))
		got := l.Next()
		if got.Kind != TokenString {
			t.Errorf("%q: kind = %s, want STRING", c.src, got.Kind)
			continue
		}
		if got.Value != c.want {
			t.Errorf("%q: value = %q, want %q", c.src, got.Value, c.want)
		}
	}
}

func TestLexerLongStrings(t *testing.T) {
	cases := []struct {
		src, want string
	}{
		{"[[hello]]", "hello"},
		{"[==[with ] and ]] inside]==]", "with ] and ]] inside"},
		{"[[\nleading newline is stripped]]", "leading newline is stripped"},
	}
	for _, c := range cases {
		l := NewLexer("t.lua", []byte(c.src))
		got := l.Next()
		if got.Kind != TokenString {
			t.Errorf("%q: kind = %s, want STRING", c.src, got.Kind)
			continue
		}
		if got.Value != c.want {
			t.Errorf("%q: value = %q, want %q", c.src, got.Value, c.want)
		}
	}
}

func TestLexerOperators(t *testing.T) {
	src := "+ - * / // % ^ # == ~= <= >= < > = ( ) { } [ ] ; : :: , . .. ..."
	want := []TokenKind{
		TokenPlus, TokenMinus, TokenStar, TokenSlash, TokenDoubleSlash,
		TokenPercent, TokenCaret, TokenHash,
		TokenEq, TokenNeq, TokenLeq, TokenGeq, TokenLt, TokenGt, TokenAssign,
		TokenLParen, TokenRParen, TokenLBrace, TokenRBrace,
		TokenLBracket, TokenRBracket, TokenSemicolon, TokenColon, TokenDoubleColon,
		TokenComma, TokenDot, TokenConcat, TokenEllipsis,
	}
	l := NewLexer("t.lua", []byte(src))
	for i, w := range want {
		got := l.Next()
		if got.Kind != w {
			t.Errorf("token %d: got %s (%q), want %s", i, got.Kind, got.Value, w)
		}
	}
}

func TestLexerSkipsComments(t *testing.T) {
	src := "-- line comment\nlocal x --[[ inline long ]] = 1\n--[==[ nested [[ ]] still comment ]==]\nend"
	want := []TokenKind{TokenLocal, TokenIdent, TokenAssign, TokenNumber, TokenEnd, TokenEOF}
	l := NewLexer("t.lua", []byte(src))
	for i, w := range want {
		got := l.Next()
		if got.Kind != w {
			t.Fatalf("token %d: got %s (%q), want %s", i, got.Kind, got.Value, w)
		}
	}
}

func TestLexerPositions(t *testing.T) {
	l := NewLexer("t.lua", []byte("local\n  x"))
	first := l.Next()
	if first.Pos.Line != 1 || first.Pos.Column != 1 {
		t.Errorf("first token pos = %v, want 1:1", first.Pos)
	}
	second := l.Next()
	if second.Pos.Line != 2 || second.Pos.Column != 3 {
		t.Errorf("second token pos = %v, want 2:3", second.Pos)
	}
}

func TestLexerLuaJITZeroEscape(t *testing.T) {
	l := NewLexer("t.lua", []byte(`"one\z    two"`))
	got := l.Next()
	if got.Kind != TokenString {
		t.Fatalf("kind = %s, want STRING", got.Kind)
	}
	if got.Value != "onetwo" {
		t.Errorf("value = %q, want %q", got.Value, "onetwo")
	}
}

func TestLexerUnterminatedString(t *testing.T) {
	l := NewLexer("t.lua", []byte(`"never ends`))
	got := l.Next()
	if got.Kind != TokenError {
		t.Errorf("kind = %s, want ERROR", got.Kind)
	}
	if len(l.Errors()) == 0 {
		t.Error("expected at least one collected error")
	}
}
