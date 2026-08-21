package lua

// parseStatement dispatches to the parser for the current statement kind.
// It returns nil on unrecognized input; the caller (parseBlock) handles
// recovery by advancing past the offending token.
func (p *Parser) parseStatement() Statement {
	switch p.tok.Kind {
	case TokenDo:
		return p.parseDo()
	case TokenWhile:
		return p.parseWhile()
	case TokenRepeat:
		return p.parseRepeat()
	case TokenIf:
		return p.parseIf()
	case TokenFor:
		return p.parseFor()
	case TokenFunction:
		return p.parseFuncDecl()
	case TokenLocal:
		return p.parseLocal()
	case TokenBreak:
		return p.parseBreak()
	case TokenGoto:
		return p.parseGoto()
	case TokenDoubleColon:
		return p.parseLabel()
	}
	return p.parseAssignOrCall()
}

// parseReturn parses `return [explist] [';']`. The block loop calls this
// only when the current token is TokenReturn; it does not itself terminate
// the block — the caller does.
func (p *Parser) parseReturn() *ReturnStat {
	tok := p.expect(TokenReturn)
	r := &ReturnStat{Position: tok.Pos}
	if !p.atBlockEnd() && !p.check(TokenSemicolon) {
		r.Values = p.parseExprList()
	}
	p.consume(TokenSemicolon)
	return r
}

// parseDo parses `do block end`.
func (p *Parser) parseDo() *DoStat {
	tok := p.expect(TokenDo)
	body := p.parseBlock()
	p.expect(TokenEnd)
	return &DoStat{Position: tok.Pos, Body: body}
}

// parseWhile parses `while exp do block end`.
func (p *Parser) parseWhile() *WhileStat {
	tok := p.expect(TokenWhile)
	cond := p.parseExpr()
	p.expect(TokenDo)
	body := p.parseBlock()
	p.expect(TokenEnd)
	return &WhileStat{Position: tok.Pos, Cond: cond, Body: body}
}

// parseRepeat parses `repeat block until exp`.
func (p *Parser) parseRepeat() *RepeatStat {
	tok := p.expect(TokenRepeat)
	body := p.parseBlock()
	p.expect(TokenUntil)
	cond := p.parseExpr()
	return &RepeatStat{Position: tok.Pos, Body: body, Cond: cond}
}

// parseIf parses `if exp then block {elseif exp then block} [else block] end`.
func (p *Parser) parseIf() *IfStat {
	tok := p.expect(TokenIf)
	stat := &IfStat{Position: tok.Pos}
	stat.Cond = p.parseExpr()
	p.expect(TokenThen)
	stat.Then = p.parseBlock()
	for p.check(TokenElseif) {
		stat.ElseIfs = append(stat.ElseIfs, p.parseElseIf())
	}
	if p.consume(TokenElse) {
		stat.Else = p.parseBlock()
	}
	p.expect(TokenEnd)
	return stat
}

// parseElseIf parses one `elseif exp then block` clause.
func (p *Parser) parseElseIf() *ElseIf {
	tok := p.expect(TokenElseif)
	cond := p.parseExpr()
	p.expect(TokenThen)
	body := p.parseBlock()
	return &ElseIf{Position: tok.Pos, Cond: cond, Body: body}
}

// parseFor dispatches to numeric or generic for based on lookahead.
//
//	for Name '=' exp ',' exp [',' exp] do block end
//	for namelist in explist do block end
func (p *Parser) parseFor() Statement {
	tok := p.expect(TokenFor)
	first := p.expect(TokenIdent)
	if p.check(TokenAssign) {
		return p.parseNumericFor(tok.Pos, first)
	}
	return p.parseGenericFor(tok.Pos, first)
}

func (p *Parser) parseNumericFor(pos Position, first Token) *NumericForStat {
	p.expect(TokenAssign)
	start := p.parseExpr()
	p.expect(TokenComma)
	stop := p.parseExpr()
	var step Expression
	if p.consume(TokenComma) {
		step = p.parseExpr()
	}
	p.expect(TokenDo)
	body := p.parseBlock()
	p.expect(TokenEnd)
	return &NumericForStat{
		Position: pos,
		Name:     &Ident{Position: first.Pos, Name: first.Value},
		Start:    start,
		Stop:     stop,
		Step:     step,
		Body:     body,
	}
}

func (p *Parser) parseGenericFor(pos Position, first Token) *GenericForStat {
	stat := &GenericForStat{
		Position: pos,
		Names:    []*Ident{{Position: first.Pos, Name: first.Value}},
	}
	for p.consume(TokenComma) {
		name := p.expect(TokenIdent)
		stat.Names = append(stat.Names, &Ident{Position: name.Pos, Name: name.Value})
		if len(stat.Names) >= MaxListItems {
			p.errorAt(name.Pos, "generic-for name list exceeds %d items", MaxListItems)
			break
		}
	}
	p.expect(TokenIn)
	stat.Values = p.parseExprList()
	p.expect(TokenDo)
	stat.Body = p.parseBlock()
	p.expect(TokenEnd)
	return stat
}

// parseFuncDecl parses `function funcname funcbody`.
func (p *Parser) parseFuncDecl() *FuncDeclStat {
	tok := p.expect(TokenFunction)
	name := p.parseFuncName()
	body := p.parseFunctionBody(tok.Pos)
	return &FuncDeclStat{Position: tok.Pos, Name: name, Body: body}
}

// parseFuncName parses `Name {'.' Name} [':' Name]`.
func (p *Parser) parseFuncName() *FuncName {
	root := p.expect(TokenIdent)
	name := &FuncName{
		Position: root.Pos,
		Root:     &Ident{Position: root.Pos, Name: root.Value},
	}
	for p.consume(TokenDot) {
		n := p.expect(TokenIdent)
		name.Dots = append(name.Dots, &Ident{Position: n.Pos, Name: n.Value})
	}
	if p.consume(TokenColon) {
		n := p.expect(TokenIdent)
		name.Method = &Ident{Position: n.Pos, Name: n.Value}
	}
	return name
}

// parseLocal parses `local function Name funcbody` or `local namelist ['=' explist]`.
func (p *Parser) parseLocal() Statement {
	tok := p.expect(TokenLocal)
	if p.check(TokenFunction) {
		return p.parseLocalFunc(tok.Pos)
	}
	return p.parseLocalAssign(tok.Pos)
}

func (p *Parser) parseLocalFunc(pos Position) *LocalFuncStat {
	p.expect(TokenFunction)
	name := p.expect(TokenIdent)
	body := p.parseFunctionBody(pos)
	return &LocalFuncStat{
		Position: pos,
		Name:     &Ident{Position: name.Pos, Name: name.Value},
		Body:     body,
	}
}

func (p *Parser) parseLocalAssign(pos Position) *LocalAssignStat {
	stat := &LocalAssignStat{Position: pos}
	first := p.expect(TokenIdent)
	stat.Names = append(stat.Names, &Ident{Position: first.Pos, Name: first.Value})
	for p.consume(TokenComma) {
		n := p.expect(TokenIdent)
		stat.Names = append(stat.Names, &Ident{Position: n.Pos, Name: n.Value})
		if len(stat.Names) >= MaxListItems {
			p.errorAt(n.Pos, "local name list exceeds %d items", MaxListItems)
			break
		}
	}
	if p.consume(TokenAssign) {
		stat.Values = p.parseExprList()
	}
	return stat
}

func (p *Parser) parseBreak() *BreakStat {
	tok := p.expect(TokenBreak)
	return &BreakStat{Position: tok.Pos}
}

func (p *Parser) parseGoto() *GotoStat {
	tok := p.expect(TokenGoto)
	name := p.expect(TokenIdent)
	return &GotoStat{Position: tok.Pos, Label: name.Value}
}

func (p *Parser) parseLabel() *LabelStat {
	tok := p.expect(TokenDoubleColon)
	name := p.expect(TokenIdent)
	p.expect(TokenDoubleColon)
	return &LabelStat{Position: tok.Pos, Name: name.Value}
}

// parseAssignOrCall parses `varlist '=' explist` or a bare function call
// used as a statement.
func (p *Parser) parseAssignOrCall() Statement {
	pos := p.tok.Pos
	first := p.parsePrefixExp()
	if first == nil {
		return nil
	}
	if p.at(TokenComma, TokenAssign) {
		return p.parseAssignRest(pos, first)
	}
	switch first.(type) {
	case *CallExpr, *MethodCallExpr:
		return &CallStat{Position: pos, Call: first}
	}
	p.errorAt(pos, "syntax error: expected assignment or function call")
	return nil
}

func (p *Parser) parseAssignRest(pos Position, first Expression) *AssignStat {
	targets := []Expression{first}
	for p.consume(TokenComma) {
		targets = append(targets, p.parsePrefixExp())
		if len(targets) >= MaxListItems {
			p.errorAt(p.tok.Pos, "assignment target list exceeds %d items", MaxListItems)
			break
		}
	}
	p.expect(TokenAssign)
	values := p.parseExprList()
	return &AssignStat{Position: pos, Targets: targets, Values: values}
}

// parseExprList parses `exp {',' exp}`. Bounded by MaxListItems to prevent
// unbounded allocation on pathological user input.
func (p *Parser) parseExprList() []Expression {
	exprs := []Expression{p.parseExpr()}
	for p.consume(TokenComma) {
		exprs = append(exprs, p.parseExpr())
		if len(exprs) >= MaxListItems {
			p.errorAt(p.tok.Pos, "expression list exceeds %d items", MaxListItems)
			return exprs
		}
	}
	return exprs
}
