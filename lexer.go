package lua

import (
	"fmt"
	"strings"
)

// Lexer produces a stream of Tokens from Lua source.
//
// The Lexer accepts the LuaJIT superset of Lua 5.1: `goto`/`::label::`, the
// `\z` string escape, and `//` integer division. Long-bracket strings and
// comments with level markers (`[==[ ... ]==]`) are supported.
type Lexer struct {
	filename string
	src      []byte
	offset   int
	line     int
	col      int
	errors   SyntaxErrors
}

// NewLexer returns a Lexer over the given source. The filename is used only
// for error messages and has no effect on lexing.
func NewLexer(filename string, src []byte) *Lexer {
	return &Lexer{
		filename: filename,
		src:      src,
		line:     1,
		col:      1,
	}
}

// Errors returns any syntax errors accumulated during lexing. It is safe to
// call at any point.
func (l *Lexer) Errors() SyntaxErrors {
	return l.errors
}

// Next returns the next Token. At end of input, Next returns a Token with
// Kind == TokenEOF; further calls continue to return EOF.
func (l *Lexer) Next() Token {
	l.skipWhitespaceAndComments()
	if l.eof() {
		return l.makeToken(TokenEOF, "")
	}
	pos := l.position()
	c := l.peek()
	switch {
	case isIdentStart(c):
		return l.scanIdentifier(pos)
	case isDigit(c):
		return l.scanNumber(pos)
	case c == '"' || c == '\'':
		return l.scanShortString(pos, c)
	case c == '[':
		if level, ok := l.tryLongBracket(); ok {
			return l.scanLongString(pos, level)
		}
	}
	return l.scanPunct(pos)
}

// --- source cursor --------------------------------------------------------

func (l *Lexer) eof() bool { return l.offset >= len(l.src) }

func (l *Lexer) peek() byte {
	if l.eof() {
		return 0
	}
	return l.src[l.offset]
}

func (l *Lexer) peekAt(n int) byte {
	if l.offset+n >= len(l.src) {
		return 0
	}
	return l.src[l.offset+n]
}

func (l *Lexer) advance() byte {
	if l.eof() {
		return 0
	}
	c := l.src[l.offset]
	l.offset++
	if c == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return c
}

func (l *Lexer) position() Position {
	return Position{Filename: l.filename, Line: l.line, Column: l.col, Offset: l.offset}
}

func (l *Lexer) makeToken(kind TokenKind, value string) Token {
	return Token{Kind: kind, Value: value, Pos: l.position()}
}

func (l *Lexer) errorf(pos Position, format string, args ...any) Token {
	msg := fmt.Sprintf(format, args...)
	l.errors = append(l.errors, &SyntaxError{Pos: pos, Message: msg})
	return Token{Kind: TokenError, Value: msg, Pos: pos}
}

// --- whitespace and comments ----------------------------------------------

// Bounded by len(l.src): every iteration either advances the offset by
// one (isSpace branch) or calls skipComment which advances at least two
// bytes. Terminates when the offset reaches EOF or hits a non-skippable
// byte.
func (l *Lexer) skipWhitespaceAndComments() {
	for !l.eof() {
		c := l.peek()
		switch {
		case isSpace(c):
			l.advance()
		case c == '-' && l.peekAt(1) == '-':
			l.skipComment()
		default:
			return
		}
	}
}

// skipComment consumes a `--` comment. Long comments (`--[=*[ ... ]=*]`) are
// handled explicitly; short comments run to end of line.
func (l *Lexer) skipComment() {
	l.advance() // -
	l.advance() // -
	if l.peek() == '[' {
		if level, ok := l.tryLongBracket(); ok {
			l.consumeLongBracketBody(level)
			return
		}
	}
	for !l.eof() && l.peek() != '\n' {
		l.advance()
	}
}

// tryLongBracket attempts to consume an opening long bracket `[=*[` and
// returns the level (number of `=` signs). On failure the lexer state is
// unchanged.
func (l *Lexer) tryLongBracket() (int, bool) {
	if l.peek() != '[' {
		return 0, false
	}
	level := 0
	for l.peekAt(1+level) == '=' {
		level++
	}
	if l.peekAt(1+level) != '[' {
		return 0, false
	}
	l.advance() // [
	for i := 0; i < level; i++ {
		l.advance() // =
	}
	l.advance() // [
	// Lua convention: a newline immediately after the opener is discarded.
	if l.peek() == '\n' {
		l.advance()
	}
	return level, true
}

// consumeLongBracketBody reads until the matching `]=*]` and discards the
// body. Used for long comments.
func (l *Lexer) consumeLongBracketBody(level int) {
	for !l.eof() {
		if l.peek() != ']' {
			l.advance()
			continue
		}
		if l.matchLongBracketClose(level) {
			return
		}
		l.advance()
	}
}

// matchLongBracketClose reports whether the current position is a valid
// closing long bracket for the given level and, if so, consumes it.
func (l *Lexer) matchLongBracketClose(level int) bool {
	if l.peek() != ']' {
		return false
	}
	for i := 0; i < level; i++ {
		if l.peekAt(1+i) != '=' {
			return false
		}
	}
	if l.peekAt(1+level) != ']' {
		return false
	}
	l.advance() // ]
	for i := 0; i < level; i++ {
		l.advance() // =
	}
	l.advance() // ]
	return true
}

// --- token scanners -------------------------------------------------------

func (l *Lexer) scanIdentifier(pos Position) Token {
	start := l.offset
	for !l.eof() && isIdentPart(l.peek()) {
		l.advance()
	}
	value := string(l.src[start:l.offset])
	if kind, ok := keywords[value]; ok {
		return Token{Kind: kind, Value: value, Pos: pos}
	}
	return Token{Kind: TokenIdent, Value: value, Pos: pos}
}

// scanNumber scans a numeric literal: decimal int, hex int (0x prefix), or
// decimal float with optional fraction and [eE][+-]?digits exponent.
func (l *Lexer) scanNumber(pos Position) Token {
	start := l.offset
	if l.peek() == '0' && (l.peekAt(1) == 'x' || l.peekAt(1) == 'X') {
		l.advance()
		l.advance()
		for !l.eof() && isHexDigit(l.peek()) {
			l.advance()
		}
		return Token{Kind: TokenNumber, Value: string(l.src[start:l.offset]), Pos: pos}
	}
	for !l.eof() && isDigit(l.peek()) {
		l.advance()
	}
	if l.peek() == '.' && isDigit(l.peekAt(1)) {
		l.advance()
		for !l.eof() && isDigit(l.peek()) {
			l.advance()
		}
	}
	l.scanExponent()
	return Token{Kind: TokenNumber, Value: string(l.src[start:l.offset]), Pos: pos}
}

// scanExponent consumes an [eE][+-]?digits exponent if present.
func (l *Lexer) scanExponent() {
	if l.peek() != 'e' && l.peek() != 'E' {
		return
	}
	l.advance()
	if l.peek() == '+' || l.peek() == '-' {
		l.advance()
	}
	for !l.eof() && isDigit(l.peek()) {
		l.advance()
	}
}

// scanShortString scans a single- or double-quoted string literal.
func (l *Lexer) scanShortString(pos Position, quote byte) Token {
	l.advance() // opening quote
	var b strings.Builder
	for !l.eof() {
		c := l.peek()
		switch c {
		case quote:
			l.advance()
			return Token{Kind: TokenString, Value: b.String(), Pos: pos}
		case '\n':
			return l.errorf(pos, "unterminated string literal")
		case '\\':
			if err := l.scanEscape(&b); err != nil {
				return l.errorf(pos, "%s", err.Error())
			}
		default:
			b.WriteByte(c)
			l.advance()
		}
	}
	return l.errorf(pos, "unterminated string literal")
}

// simpleEscapes maps single-character escape sequences to their byte value.
var simpleEscapes = map[byte]byte{
	'a': 0x07, 'b': 0x08, 'f': 0x0C, 'n': '\n',
	'r': '\r', 't': '\t', 'v': 0x0B,
	'\\': '\\', '"': '"', '\'': '\'',
}

// scanEscape handles the character after `\` in a short string. The
// leading `\` is consumed on entry.
func (l *Lexer) scanEscape(b *strings.Builder) error {
	l.advance() // consume '\'
	if l.eof() {
		return fmt.Errorf("unterminated escape sequence")
	}
	c := l.advance()
	if v, ok := simpleEscapes[c]; ok {
		b.WriteByte(v)
		return nil
	}
	switch c {
	case '\n':
		b.WriteByte('\n')
		return nil
	case 'z':
		for !l.eof() && isSpace(l.peek()) {
			l.advance()
		}
		return nil
	}
	if isDigit(c) {
		l.scanDecimalEscape(b, c)
		return nil
	}
	return fmt.Errorf("invalid escape sequence \\%c", c)
}

// scanDecimalEscape consumes up to two more decimal digits following the
// initial digit for a \ddd escape sequence.
func (l *Lexer) scanDecimalEscape(b *strings.Builder, first byte) {
	value := int(first - '0')
	for i := 0; i < 2; i++ {
		if !isDigit(l.peek()) {
			break
		}
		value = value*10 + int(l.advance()-'0')
	}
	if value > 255 {
		l.errors = append(l.errors, &SyntaxError{
			Pos:     l.position(),
			Message: fmt.Sprintf("decimal escape \\%d out of range", value),
		})
		return
	}
	b.WriteByte(byte(value))
}

// scanLongString scans a long-bracket string. The opening bracket has
// already been consumed; level identifies its `=` count.
func (l *Lexer) scanLongString(pos Position, level int) Token {
	start := l.offset
	for !l.eof() {
		if l.peek() != ']' {
			l.advance()
			continue
		}
		end := l.offset
		if l.matchLongBracketClose(level) {
			return Token{Kind: TokenString, Value: string(l.src[start:end]), Pos: pos}
		}
		l.advance()
	}
	return l.errorf(pos, "unterminated long string")
}

// singleCharPunct maps single-byte punctuation to its TokenKind. Multi-byte
// forms (== ~= <= >= // :: .. ...) are handled explicitly in scanPunct.
var singleCharPunct = map[byte]TokenKind{
	'+': TokenPlus, '-': TokenMinus, '*': TokenStar,
	'%': TokenPercent, '^': TokenCaret, '#': TokenHash,
	'(': TokenLParen, ')': TokenRParen,
	'{': TokenLBrace, '}': TokenRBrace,
	'[': TokenLBracket, ']': TokenRBracket,
	';': TokenSemicolon, ',': TokenComma,
}

// scanPunct scans operator and punctuation tokens. The first byte of the
// operator is consumed on entry.
func (l *Lexer) scanPunct(pos Position) Token {
	c := l.advance()
	switch c {
	case '/':
		if l.peek() == '/' {
			l.advance()
			return Token{Kind: TokenDoubleSlash, Value: "//", Pos: pos}
		}
		return Token{Kind: TokenSlash, Value: "/", Pos: pos}
	case '=':
		if l.peek() == '=' {
			l.advance()
			return Token{Kind: TokenEq, Value: "==", Pos: pos}
		}
		return Token{Kind: TokenAssign, Value: "=", Pos: pos}
	case '~':
		if l.peek() == '=' {
			l.advance()
			return Token{Kind: TokenNeq, Value: "~=", Pos: pos}
		}
		return l.errorf(pos, "unexpected character '~'")
	case '<':
		if l.peek() == '=' {
			l.advance()
			return Token{Kind: TokenLeq, Value: "<=", Pos: pos}
		}
		return Token{Kind: TokenLt, Value: "<", Pos: pos}
	case '>':
		if l.peek() == '=' {
			l.advance()
			return Token{Kind: TokenGeq, Value: ">=", Pos: pos}
		}
		return Token{Kind: TokenGt, Value: ">", Pos: pos}
	case ':':
		if l.peek() == ':' {
			l.advance()
			return Token{Kind: TokenDoubleColon, Value: "::", Pos: pos}
		}
		return Token{Kind: TokenColon, Value: ":", Pos: pos}
	case '.':
		return l.scanDotOrConcatOrEllipsis(pos)
	}
	if kind, ok := singleCharPunct[c]; ok {
		return Token{Kind: kind, Value: string(c), Pos: pos}
	}
	return l.errorf(pos, "unexpected character %q", string(c))
}

// scanDotOrConcatOrEllipsis distinguishes `.`, `..`, `...`, and a fractional
// number starting with `.` (e.g., `.25`). The leading `.` has already been
// consumed by scanPunct.
func (l *Lexer) scanDotOrConcatOrEllipsis(pos Position) Token {
	if l.peek() == '.' {
		l.advance()
		if l.peek() == '.' {
			l.advance()
			return Token{Kind: TokenEllipsis, Value: "...", Pos: pos}
		}
		return Token{Kind: TokenConcat, Value: "..", Pos: pos}
	}
	if isDigit(l.peek()) {
		start := l.offset - 1
		for !l.eof() && isDigit(l.peek()) {
			l.advance()
		}
		l.scanExponent()
		return Token{Kind: TokenNumber, Value: string(l.src[start:l.offset]), Pos: pos}
	}
	return Token{Kind: TokenDot, Value: ".", Pos: pos}
}

// --- character classifiers ------------------------------------------------

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || isDigit(c)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}
