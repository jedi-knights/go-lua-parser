package lua

// binaryOp describes precedence and associativity of a binary operator.
//
// The `prec` field is an internal encoding — NOT the 1–8 scale from the
// Lua 5.1 reference §2.5.6. Only relative ordering matters: higher
// numbers bind tighter, and `unaryPrec` sits between the comparison level
// and `^` on purpose so `-2^2` parses as `-(2^2)`. Do not "fix" these
// values to match the spec's integers — the parser tests (see
// TestParseUnaryLowerThanCaret) pin the current relationships.
//
// A right-associative operator recurses at the same precedence, a
// left-associative one at prec+1.
type binaryOp struct {
	prec       int
	rightAssoc bool
	op         string
}

// unaryPrec is the binding power of unary operators. It sits below `^`
// so that `-2^2` parses as `-(2^2)`.
const unaryPrec = 8

// binaryOps maps token kinds to their operator info. `//` (LuaJIT integer
// division) is grouped with `* / %` at the multiplicative level.
//
// Precedence level 4 is intentionally unused — the jump from 3 (comparisons)
// to 5 (concat) is a deliberate gap. Do not fill 4 without re-deriving the
// relationships pinned by TestParseUnaryLowerThanCaret and friends.
var binaryOps = map[TokenKind]binaryOp{
	TokenOr:          {1, false, "or"},
	TokenAnd:         {2, false, "and"},
	TokenLt:          {3, false, "<"},
	TokenGt:          {3, false, ">"},
	TokenLeq:         {3, false, "<="},
	TokenGeq:         {3, false, ">="},
	TokenNeq:         {3, false, "~="},
	TokenEq:          {3, false, "=="},
	TokenConcat:      {5, true, ".."},
	TokenPlus:        {6, false, "+"},
	TokenMinus:       {6, false, "-"},
	TokenStar:        {7, false, "*"},
	TokenSlash:       {7, false, "/"},
	TokenDoubleSlash: {7, false, "//"},
	TokenPercent:     {7, false, "%"},
	TokenCaret:       {11, true, "^"},
}

// parseExpr parses a single expression.
func (p *Parser) parseExpr() Expression {
	return p.parseExprPrec(0)
}

// parseExprPrec parses an expression whose top-level operator has binding
// power at least minPrec. This is textbook precedence climbing.
func (p *Parser) parseExprPrec(minPrec int) Expression {
	left := p.parseUnary()
	for {
		op, ok := binaryOps[p.tok.Kind]
		if !ok || op.prec < minPrec {
			return left
		}
		pos := p.tok.Pos
		p.advance()
		nextMin := op.prec + 1
		if op.rightAssoc {
			nextMin = op.prec
		}
		right := p.parseExprPrec(nextMin)
		left = &BinaryExpr{Position: pos, Op: op.op, Left: left, Right: right}
	}
}

// parseUnary parses zero or more unary operators followed by a primary.
// Unary applies at unaryPrec so that operators higher than unary (only `^`)
// bind tighter to the operand.
func (p *Parser) parseUnary() Expression {
	if op, ok := unaryOp(p.tok.Kind); ok {
		pos := p.tok.Pos
		p.advance()
		operand := p.parseExprPrec(unaryPrec)
		return &UnaryExpr{Position: pos, Op: op, Operand: operand}
	}
	return p.parsePrimary()
}

// unaryOp maps a token kind to its unary operator lexeme.
func unaryOp(k TokenKind) (string, bool) {
	switch k {
	case TokenMinus:
		return "-", true
	case TokenNot:
		return "not", true
	case TokenHash:
		return "#", true
	}
	return "", false
}

// parsePrimary parses a literal, table constructor, function expression,
// or prefix expression (identifier or parenthesized) with any chain of
// `.name`, `[expr]`, `:name(args)`, or `(args)` suffixes.
func (p *Parser) parsePrimary() Expression {
	if e, ok := p.tryAtomLiteral(); ok {
		return e
	}
	switch p.tok.Kind {
	case TokenFunction:
		return p.parseFunctionExpr()
	case TokenLBrace:
		return p.parseTableConstructor()
	}
	return p.parsePrefixExp()
}

// tryAtomLiteral consumes and returns a self-contained literal expression:
// nil, true, false, Number, String, or `...`.
func (p *Parser) tryAtomLiteral() (Expression, bool) {
	pos := p.tok.Pos
	switch p.tok.Kind {
	case TokenNil:
		p.advance()
		return &NilExpr{Position: pos}, true
	case TokenTrue:
		p.advance()
		return &TrueExpr{Position: pos}, true
	case TokenFalse:
		p.advance()
		return &FalseExpr{Position: pos}, true
	case TokenNumber:
		text := p.tok.Value
		p.advance()
		return &NumberExpr{Position: pos, Text: text}, true
	case TokenString:
		value := p.tok.Value
		p.advance()
		return &StringExpr{Position: pos, Value: value}, true
	case TokenEllipsis:
		p.advance()
		return &VarargExpr{Position: pos}, true
	}
	return nil, false
}

// parsePrefixExp parses an identifier or `(expr)` followed by any chain
// of `.`/`[`/`:`/call suffixes.
func (p *Parser) parsePrefixExp() Expression {
	expr := p.parsePrefixHead()
	if expr == nil {
		return nil
	}
	for {
		next := p.applyPrefixSuffix(expr)
		if next == nil {
			return expr
		}
		expr = next
	}
}

// parsePrefixHead parses the leading Name or `(expr)` of a prefix chain.
func (p *Parser) parsePrefixHead() Expression {
	switch p.tok.Kind {
	case TokenIdent:
		pos := p.tok.Pos
		name := p.tok.Value
		p.advance()
		return &Ident{Position: pos, Name: name}
	case TokenLParen:
		p.advance()
		expr := p.parseExpr()
		p.expect(TokenRParen)
		return expr
	}
	p.errorAt(p.tok.Pos, "expected identifier or '(', got %s", p.tok.Kind)
	return nil
}

// applyPrefixSuffix applies at most one prefix suffix to expr and returns
// the extended expression, or nil if no suffix is present.
func (p *Parser) applyPrefixSuffix(expr Expression) Expression {
	switch p.tok.Kind {
	case TokenDot:
		p.advance()
		name := p.expect(TokenIdent)
		return &FieldExpr{Position: expr.Pos(), Object: expr, Name: name.Value}
	case TokenLBracket:
		pos := p.tok.Pos
		p.advance()
		index := p.parseExpr()
		p.expect(TokenRBracket)
		return &IndexExpr{Position: pos, Object: expr, Index: index}
	case TokenColon:
		p.advance()
		method := p.expect(TokenIdent)
		args := p.parseCallArgs()
		return &MethodCallExpr{Position: expr.Pos(), Object: expr, Method: method.Value, Args: args}
	case TokenLParen, TokenLBrace, TokenString:
		args := p.parseCallArgs()
		return &CallExpr{Position: expr.Pos(), Fn: expr, Args: args}
	}
	return nil
}

// parseCallArgs parses one of the three Lua argument forms:
//
//	'(' [explist] ')' | tableconstructor | LiteralString
func (p *Parser) parseCallArgs() []Expression {
	switch p.tok.Kind {
	case TokenLParen:
		p.advance()
		if p.consume(TokenRParen) {
			return nil
		}
		args := p.parseExprList()
		p.expect(TokenRParen)
		return args
	case TokenLBrace:
		return []Expression{p.parseTableConstructor()}
	case TokenString:
		pos := p.tok.Pos
		value := p.tok.Value
		p.advance()
		return []Expression{&StringExpr{Position: pos, Value: value}}
	}
	p.errorAt(p.tok.Pos, "expected function call arguments, got %s", p.tok.Kind)
	return nil
}

// parseTableConstructor parses `{ [fieldlist] }`.
// fieldlist ::= field {fieldsep field} [fieldsep]
// fieldsep  ::= ',' | ';'
func (p *Parser) parseTableConstructor() *TableExpr {
	tok := p.expect(TokenLBrace)
	tab := &TableExpr{Position: tok.Pos}
	for !p.check(TokenRBrace) && !p.check(TokenEOF) {
		tab.Fields = append(tab.Fields, p.parseTableField())
		if len(tab.Fields) >= MaxListItems {
			p.errorAt(p.tok.Pos, "table constructor exceeds %d fields", MaxListItems)
			break
		}
		if !p.consume(TokenComma) && !p.consume(TokenSemicolon) {
			break
		}
	}
	p.expect(TokenRBrace)
	return tab
}

// parseTableField parses one entry of a table constructor.
// field ::= '[' exp ']' '=' exp | Name '=' exp | exp
func (p *Parser) parseTableField() TableField {
	pos := p.tok.Pos
	switch {
	case p.check(TokenLBracket):
		p.advance()
		key := p.parseExpr()
		p.expect(TokenRBracket)
		p.expect(TokenAssign)
		return TableField{Position: pos, Key: key, Value: p.parseExpr()}
	case p.check(TokenIdent) && p.next.Kind == TokenAssign:
		name := p.tok.Value
		p.advance() // name
		p.advance() // =
		return TableField{Position: pos, KeyName: name, Value: p.parseExpr()}
	}
	return TableField{Position: pos, Value: p.parseExpr()}
}

// parseFunctionExpr parses `function funcbody`.
func (p *Parser) parseFunctionExpr() *FunctionExpr {
	tok := p.expect(TokenFunction)
	return p.parseFunctionBody(tok.Pos)
}

// parseFunctionBody parses `'(' [parlist] ')' block end`. The caller
// supplies the position of the `function` keyword or declaration.
func (p *Parser) parseFunctionBody(pos Position) *FunctionExpr {
	p.expect(TokenLParen)
	fn := &FunctionExpr{Position: pos}
	if !p.check(TokenRParen) {
		p.parseParamList(fn)
	}
	p.expect(TokenRParen)
	fn.Body = p.parseBlock()
	p.expect(TokenEnd)
	return fn
}

// parseParamList parses `namelist [',' '...'] | '...'`. The list length
// is bounded by MaxListItems to prevent an unbounded allocation on
// pathological user input.
func (p *Parser) parseParamList(fn *FunctionExpr) {
	for {
		if p.check(TokenEllipsis) {
			fn.IsVararg = true
			p.advance()
			return
		}
		name := p.expect(TokenIdent)
		fn.Params = append(fn.Params, &Ident{Position: name.Pos, Name: name.Value})
		if len(fn.Params) >= MaxListItems {
			p.errorAt(name.Pos, "parameter list exceeds %d items", MaxListItems)
			return
		}
		if !p.consume(TokenComma) {
			return
		}
	}
}
