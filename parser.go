package lua

import "fmt"

// MaxListItems is the upper bound on the number of items the parser will
// accept in any comma-separated list — parameters, expression lists,
// local-assign names, generic-for names, assignment targets. Prevents an
// unbounded loop on user-supplied source per the algorithmic-complexity
// bounded-loops rule. Chosen to match Lua 5.1's MAXVARS (200) with a small
// buffer for the LuaJIT superset.
const MaxListItems = 250

// Parser converts a token stream from a Lexer into an AST.
//
// The parser accepts the LuaJIT superset of Lua 5.1: `goto`/`::label::`,
// the `\z` string escape, and `//` integer division. Semantic checks that
// are enforced by the reference Lua compiler but not by the grammar
// (e.g., that `break` appears only inside a loop, or that `return` is the
// last statement in its block) are not enforced here; the parser accepts
// them as parse-time valid and leaves such checks to callers.
type Parser struct {
	lexer  *Lexer
	tok    Token // current token
	next   Token // one-token lookahead
	errors SyntaxErrors
}

// Parse parses a complete Lua source file into a Chunk. The returned error,
// if any, is a SyntaxErrors describing every issue found by the lexer and
// parser combined. The Chunk is always returned (possibly partial) so that
// callers can produce diagnostics without a second parse.
func Parse(filename string, src []byte) (*Chunk, error) {
	p := newParser(filename, src)
	block := p.parseBlock()
	chunk := &Chunk{Filename: filename, Block: block}
	return chunk, p.finalize()
}

// newParser constructs a Parser and primes both tok and next.
func newParser(filename string, src []byte) *Parser {
	p := &Parser{lexer: NewLexer(filename, src)}
	p.advance()
	p.advance()
	return p
}

// advance moves the parser one token forward and returns the previous token.
func (p *Parser) advance() Token {
	prev := p.tok
	p.tok = p.next
	p.next = p.lexer.Next()
	return prev
}

// finalize merges lexer and parser errors into a single result.
func (p *Parser) finalize() error {
	errs := append(SyntaxErrors{}, p.lexer.Errors()...)
	errs = append(errs, p.errors...)
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// --- token helpers --------------------------------------------------------

// check reports whether the current token is of the given kind.
func (p *Parser) check(kind TokenKind) bool {
	return p.tok.Kind == kind
}

// at reports whether the current token is any of the given kinds.
func (p *Parser) at(kinds ...TokenKind) bool {
	for _, k := range kinds {
		if p.tok.Kind == k {
			return true
		}
	}
	return false
}

// consume advances past the current token if it matches kind, returning
// whether the match succeeded.
func (p *Parser) consume(kind TokenKind) bool {
	if p.tok.Kind != kind {
		return false
	}
	p.advance()
	return true
}

// expect advances past the current token if it matches, otherwise records
// a syntax error. The returned Token is either the consumed token or the
// current (unexpected) token so callers still get a valid Position.
func (p *Parser) expect(kind TokenKind) Token {
	if p.tok.Kind == kind {
		return p.advance()
	}
	p.errorAt(p.tok.Pos, "expected %s, got %s", kind, p.tok.Kind)
	return p.tok
}

// errorAt records a syntax error at the given position.
func (p *Parser) errorAt(pos Position, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	p.errors = append(p.errors, &SyntaxError{Pos: pos, Message: msg})
}

// atBlockEnd reports whether the current token terminates the enclosing
// block. These are the tokens the block's caller consumes.
func (p *Parser) atBlockEnd() bool {
	return p.at(TokenEOF, TokenEnd, TokenElse, TokenElseif, TokenUntil)
}

// atSyncPoint reports whether the current token is a plausible statement
// boundary — a block terminator, a semicolon, the start of a new
// statement, or a lexer error token. Used by sync() to recover after a
// parse error. TokenError is treated as a sync point so a run of invalid
// bytes doesn't force sync() to scan the entire tail of the input before
// reaching EOF; the caller reports it, records an error, and moves on.
func (p *Parser) atSyncPoint() bool {
	return p.at(
		TokenEOF, TokenError,
		TokenEnd, TokenElse, TokenElseif, TokenUntil,
		TokenSemicolon, TokenDo, TokenWhile, TokenRepeat,
		TokenIf, TokenFor, TokenFunction, TokenLocal,
		TokenBreak, TokenGoto, TokenDoubleColon, TokenReturn,
	)
}

// sync advances tokens until the current position is a statement boundary.
// This is textbook panic-mode recovery: after a parse error we skip the
// tail of the malformed construct so that subsequent errors describe
// independent problems rather than cascading from the same failure.
func (p *Parser) sync() {
	for !p.atSyncPoint() {
		p.advance()
	}
}

// parseBlock parses statements until a block terminator. On a failed
// statement parse it advances past the offending token (if no progress was
// made) and then syncs to the next statement boundary, so a single bad
// statement does not cause a cascade of errors on the rest of the block.
func (p *Parser) parseBlock() *Block {
	block := &Block{Position: p.tok.Pos}
	for !p.atBlockEnd() {
		if p.consume(TokenSemicolon) {
			continue
		}
		if p.check(TokenReturn) {
			block.Return = p.parseReturn()
			break
		}
		before := p.tok.Pos.Offset
		stat := p.parseStatement()
		if stat != nil {
			block.Statements = append(block.Statements, stat)
			continue
		}
		if p.tok.Pos.Offset == before {
			p.advance()
		}
		p.sync()
	}
	return block
}
