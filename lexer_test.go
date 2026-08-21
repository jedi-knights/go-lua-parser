package lua_test

import (
	"testing"

	lua "github.com/jedi-knights/go-lua-parser"
)

func TestLexerKeywords(t *testing.T) {
	src := "and break do else elseif end false for function goto if in local nil not or repeat return then true until while"
	want := []lua.TokenKind{
		lua.TokenAnd, lua.TokenBreak, lua.TokenDo, lua.TokenElse, lua.TokenElseif, lua.TokenEnd,
		lua.TokenFalse, lua.TokenFor, lua.TokenFunction, lua.TokenGoto, lua.TokenIf, lua.TokenIn,
		lua.TokenLocal, lua.TokenNil, lua.TokenNot, lua.TokenOr, lua.TokenRepeat, lua.TokenReturn,
		lua.TokenThen, lua.TokenTrue, lua.TokenUntil, lua.TokenWhile,
	}
	l := lua.NewLexer("t.lua", []byte(src))
	for i, w := range want {
		got := l.Next()
		if got.Kind != w {
			t.Errorf("token %d: got %s, want %s", i, got.Kind, w)
		}
	}
	if got := l.Next(); got.Kind != lua.TokenEOF {
		t.Errorf("trailing token: got %s (%q), want EOF", got.Kind, got.Value)
	}
}

func TestLexerIdentifiers(t *testing.T) {
	l := lua.NewLexer("t.lua", []byte("foo_bar _x123 y"))
	want := []string{"foo_bar", "_x123", "y"}
	for i, w := range want {
		got := l.Next()
		if got.Kind != lua.TokenIdent {
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
		l := lua.NewLexer("t.lua", []byte(c.src))
		got := l.Next()
		if got.Kind != lua.TokenNumber {
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
		l := lua.NewLexer("t.lua", []byte(c.src))
		got := l.Next()
		if got.Kind != lua.TokenString {
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
		l := lua.NewLexer("t.lua", []byte(c.src))
		got := l.Next()
		if got.Kind != lua.TokenString {
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
	want := []lua.TokenKind{
		lua.TokenPlus, lua.TokenMinus, lua.TokenStar, lua.TokenSlash, lua.TokenDoubleSlash,
		lua.TokenPercent, lua.TokenCaret, lua.TokenHash,
		lua.TokenEq, lua.TokenNeq, lua.TokenLeq, lua.TokenGeq, lua.TokenLt, lua.TokenGt, lua.TokenAssign,
		lua.TokenLParen, lua.TokenRParen, lua.TokenLBrace, lua.TokenRBrace,
		lua.TokenLBracket, lua.TokenRBracket, lua.TokenSemicolon, lua.TokenColon, lua.TokenDoubleColon,
		lua.TokenComma, lua.TokenDot, lua.TokenConcat, lua.TokenEllipsis,
	}
	l := lua.NewLexer("t.lua", []byte(src))
	for i, w := range want {
		got := l.Next()
		if got.Kind != w {
			t.Errorf("token %d: got %s (%q), want %s", i, got.Kind, got.Value, w)
		}
	}
}

func TestLexerSkipsComments(t *testing.T) {
	src := "-- line comment\nlocal x --[[ inline long ]] = 1\n--[==[ nested [[ ]] still comment ]==]\nend"
	want := []lua.TokenKind{lua.TokenLocal, lua.TokenIdent, lua.TokenAssign, lua.TokenNumber, lua.TokenEnd, lua.TokenEOF}
	l := lua.NewLexer("t.lua", []byte(src))
	for i, w := range want {
		got := l.Next()
		if got.Kind != w {
			t.Fatalf("token %d: got %s (%q), want %s", i, got.Kind, got.Value, w)
		}
	}
}

func TestLexerPositions(t *testing.T) {
	l := lua.NewLexer("t.lua", []byte("local\n  x"))
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
	l := lua.NewLexer("t.lua", []byte(`"one\z    two"`))
	got := l.Next()
	if got.Kind != lua.TokenString {
		t.Fatalf("kind = %s, want STRING", got.Kind)
	}
	if got.Value != "onetwo" {
		t.Errorf("value = %q, want %q", got.Value, "onetwo")
	}
}

func TestLexerUnterminatedString(t *testing.T) {
	l := lua.NewLexer("t.lua", []byte(`"never ends`))
	got := l.Next()
	if got.Kind != lua.TokenError {
		t.Errorf("kind = %s, want ERROR", got.Kind)
	}
	if len(l.Errors()) == 0 {
		t.Error("expected at least one collected error")
	}
}
