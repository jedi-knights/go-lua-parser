package lua

// Parser converts a token stream from a Lexer into an AST.
//
// The parser is under active development. Currently supported grammar:
//   - Empty programs
//   - Comment-only files
//
// Statement and expression parsing lands in follow-up releases; see the
// package README for status.
type Parser struct {
	lexer  *Lexer
	tok    Token // current token
	next   Token // one-token lookahead
	errors SyntaxErrors
}

// Parse parses a complete Lua source file into a Chunk. The returned error,
// if any, is a SyntaxErrors describing every issue found by the lexer and
// parser combined; the Chunk is always returned (possibly partial) so that
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
// tok is the current token; next is the one-token lookahead.
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

// parseBlock parses a sequence of statements. The current implementation
// accepts only empty blocks (EOF or immediately-closing tokens); statement
// parsing arrives in follow-up commits.
func (p *Parser) parseBlock() *Block {
	return &Block{Position: p.tok.Pos}
}
