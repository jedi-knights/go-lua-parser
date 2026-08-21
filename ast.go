package lua

import "strconv"

// Node is the common interface for all AST nodes. Every node reports the
// position of its first token so callers can produce diagnostics.
type Node interface {
	Pos() Position
}

// Statement is implemented by nodes that appear at statement position.
type Statement interface {
	Node
	statementNode()
}

// Expression is implemented by nodes that appear at expression position.
type Expression interface {
	Node
	expressionNode()
}

// Chunk is the root of a Lua source file.
type Chunk struct {
	Filename string
	Block    *Block
}

// Pos returns the start of the file.
func (c *Chunk) Pos() Position {
	return Position{Filename: c.Filename, Line: 1, Column: 1}
}

// Block is a sequence of statements optionally terminated by a return.
type Block struct {
	Position   Position
	Statements []Statement
	Return     *ReturnStat // nil if the block does not end with `return`
}

// Pos returns the block's start position.
func (b *Block) Pos() Position { return b.Position }

// --- statements -----------------------------------------------------------

// AssignStat is `t1, t2 = e1, e2`. Each target must be an assignable
// expression: *Ident, *IndexExpr, or *FieldExpr.
type AssignStat struct {
	Position Position
	Targets  []Expression
	Values   []Expression
}

// LocalAssignStat is `local n1, n2 = e1, e2`. Values may be empty.
type LocalAssignStat struct {
	Position Position
	Names    []*Ident
	Values   []Expression
}

// CallStat wraps a function or method call used as a statement.
type CallStat struct {
	Position Position
	Call     Expression // *CallExpr or *MethodCallExpr
}

// DoStat is `do ... end`.
type DoStat struct {
	Position Position
	Body     *Block
}

// WhileStat is `while cond do ... end`.
type WhileStat struct {
	Position Position
	Cond     Expression
	Body     *Block
}

// RepeatStat is `repeat ... until cond`.
type RepeatStat struct {
	Position Position
	Body     *Block
	Cond     Expression
}

// IfStat is `if cond then ... elseif ... else ... end`.
type IfStat struct {
	Position Position
	Cond     Expression
	Then     *Block
	ElseIfs  []*ElseIf
	Else     *Block // nil if there is no else branch
}

// ElseIf is one `elseif` clause of an IfStat.
type ElseIf struct {
	Position Position
	Cond     Expression
	Body     *Block
}

// NumericForStat is `for name = start, stop [, step] do ... end`.
type NumericForStat struct {
	Position Position
	Name     *Ident
	Start    Expression
	Stop     Expression
	Step     Expression // nil if omitted
	Body     *Block
}

// GenericForStat is `for n1, n2 in e1, e2 do ... end`.
type GenericForStat struct {
	Position Position
	Names    []*Ident
	Values   []Expression
	Body     *Block
}

// FuncDeclStat is `function a.b.c:d(...) ... end`.
type FuncDeclStat struct {
	Position Position
	Name     *FuncName
	Body     *FunctionExpr
}

// LocalFuncStat is `local function name(...) ... end`.
type LocalFuncStat struct {
	Position Position
	Name     *Ident
	Body     *FunctionExpr
}

// ReturnStat is `return e1, e2`. Values may be empty.
type ReturnStat struct {
	Position Position
	Values   []Expression
}

// BreakStat is `break`.
type BreakStat struct {
	Position Position
}

// GotoStat is `goto label`.
type GotoStat struct {
	Position Position
	Label    string
}

// LabelStat is `::name::`.
type LabelStat struct {
	Position Position
	Name     string
}

// FuncName is the dotted/colon name used to declare a function, e.g.
// `a.b.c:d`. Method is non-nil iff the declaration used `:`.
type FuncName struct {
	Position Position
	Root     *Ident
	Dots     []*Ident
	Method   *Ident
}

// --- expressions ----------------------------------------------------------

// NilExpr is the `nil` literal.
type NilExpr struct{ Position Position }

// TrueExpr is the `true` literal.
type TrueExpr struct{ Position Position }

// FalseExpr is the `false` literal.
type FalseExpr struct{ Position Position }

// VarargExpr is the `...` expression usable inside a vararg function.
type VarargExpr struct{ Position Position }

// NumberExpr is a numeric literal. Text is the raw source; use Float or
// IsInteger to decode.
type NumberExpr struct {
	Position Position
	Text     string
}

// Float decodes the literal as a float64. Decimal integers, decimal floats
// with optional exponent, and hexadecimal integers (`0x`/`0X` prefix) are
// supported — the full Lua 5.1 / LuaJIT numeric grammar. Callers that need
// a specific integer type should check IsInteger first.
//
// Hex literals are parsed as unsigned so values above int64 max (e.g.
// `0xFFFFFFFFFFFFFFFF`) round-trip cleanly into float64 rather than
// erroring with an overflow that a caller cannot distinguish from a
// malformed literal.
func (n *NumberExpr) Float() (float64, error) {
	if isHexLiteral(n.Text) {
		v, err := strconv.ParseUint(n.Text, 0, 64)
		if err != nil {
			return 0, err
		}
		return float64(v), nil
	}
	return strconv.ParseFloat(n.Text, 64)
}

// IsInteger reports whether the literal has no fractional or exponent
// component and can therefore be represented as an integer without loss.
// Hex literals (`0xFF`, `0xE`) are always integers under Lua 5.1 rules —
// note that `E` inside a hex literal is a digit, not an exponent marker.
func (n *NumberExpr) IsInteger() bool {
	if isHexLiteral(n.Text) {
		return true
	}
	for i := 0; i < len(n.Text); i++ {
		c := n.Text[i]
		if c == '.' || c == 'e' || c == 'E' {
			return false
		}
	}
	return true
}

func isHexLiteral(text string) bool {
	return len(text) >= 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X')
}

// StringExpr is a string literal. Value is the decoded contents (escape
// sequences applied for short strings; long-string bodies verbatim).
type StringExpr struct {
	Position Position
	Value    string
}

// Ident is a variable or field name reference.
type Ident struct {
	Position Position
	Name     string
}

// IndexExpr is `object[index]`.
type IndexExpr struct {
	Position Position
	Object   Expression
	Index    Expression
}

// FieldExpr is `object.name`.
type FieldExpr struct {
	Position Position
	Object   Expression
	Name     string
}

// CallExpr is `fn(args)`.
type CallExpr struct {
	Position Position
	Fn       Expression
	Args     []Expression
}

// MethodCallExpr is `object:method(args)`.
type MethodCallExpr struct {
	Position Position
	Object   Expression
	Method   string
	Args     []Expression
}

// BinaryExpr is a binary operator application.
// Op holds the operator lexeme ("+", "==", "and", "..", etc.).
type BinaryExpr struct {
	Position Position
	Op       string
	Left     Expression
	Right    Expression
}

// UnaryExpr is a unary operator application.
// Op is one of "-", "not", "#".
type UnaryExpr struct {
	Position Position
	Op       string
	Operand  Expression
}

// FunctionExpr is a function literal body — the `(params) body end` portion
// of both anonymous functions and named function declarations.
type FunctionExpr struct {
	Position Position
	Params   []*Ident
	IsVararg bool
	Body     *Block
}

// TableExpr is a table constructor `{ ... }`.
type TableExpr struct {
	Position Position
	Fields   []TableField
}

// TableField is one entry in a table constructor. Exactly one of Key or
// KeyName is set for keyed entries; positional entries leave both zero.
type TableField struct {
	Position Position
	Key      Expression // set for `[expr] = value` form
	KeyName  string     // set for `name = value` form
	Value    Expression
}

// --- interface satisfaction -----------------------------------------------

func (n *AssignStat) statementNode()      {}
func (n *AssignStat) Pos() Position       { return n.Position }
func (n *LocalAssignStat) statementNode() {}
func (n *LocalAssignStat) Pos() Position  { return n.Position }
func (n *CallStat) statementNode()        {}
func (n *CallStat) Pos() Position         { return n.Position }
func (n *DoStat) statementNode()          {}
func (n *DoStat) Pos() Position           { return n.Position }
func (n *WhileStat) statementNode()       {}
func (n *WhileStat) Pos() Position        { return n.Position }
func (n *RepeatStat) statementNode()      {}
func (n *RepeatStat) Pos() Position       { return n.Position }
func (n *IfStat) statementNode()          {}
func (n *IfStat) Pos() Position           { return n.Position }
func (n *NumericForStat) statementNode()  {}
func (n *NumericForStat) Pos() Position   { return n.Position }
func (n *GenericForStat) statementNode()  {}
func (n *GenericForStat) Pos() Position   { return n.Position }
func (n *FuncDeclStat) statementNode()    {}
func (n *FuncDeclStat) Pos() Position     { return n.Position }
func (n *LocalFuncStat) statementNode()   {}
func (n *LocalFuncStat) Pos() Position    { return n.Position }
func (n *ReturnStat) statementNode()      {}
func (n *ReturnStat) Pos() Position       { return n.Position }
func (n *BreakStat) statementNode()       {}
func (n *BreakStat) Pos() Position        { return n.Position }
func (n *GotoStat) statementNode()        {}
func (n *GotoStat) Pos() Position         { return n.Position }
func (n *LabelStat) statementNode()       {}
func (n *LabelStat) Pos() Position        { return n.Position }
func (n *FuncName) Pos() Position         { return n.Position }

func (n *NilExpr) expressionNode()        {}
func (n *NilExpr) Pos() Position          { return n.Position }
func (n *TrueExpr) expressionNode()       {}
func (n *TrueExpr) Pos() Position         { return n.Position }
func (n *FalseExpr) expressionNode()      {}
func (n *FalseExpr) Pos() Position        { return n.Position }
func (n *VarargExpr) expressionNode()     {}
func (n *VarargExpr) Pos() Position       { return n.Position }
func (n *NumberExpr) expressionNode()     {}
func (n *NumberExpr) Pos() Position       { return n.Position }
func (n *StringExpr) expressionNode()     {}
func (n *StringExpr) Pos() Position       { return n.Position }
func (n *Ident) expressionNode()          {}
func (n *Ident) Pos() Position            { return n.Position }
func (n *IndexExpr) expressionNode()      {}
func (n *IndexExpr) Pos() Position        { return n.Position }
func (n *FieldExpr) expressionNode()      {}
func (n *FieldExpr) Pos() Position        { return n.Position }
func (n *CallExpr) expressionNode()       {}
func (n *CallExpr) Pos() Position         { return n.Position }
func (n *MethodCallExpr) expressionNode() {}
func (n *MethodCallExpr) Pos() Position   { return n.Position }
func (n *BinaryExpr) expressionNode()     {}
func (n *BinaryExpr) Pos() Position       { return n.Position }
func (n *UnaryExpr) expressionNode()      {}
func (n *UnaryExpr) Pos() Position        { return n.Position }
func (n *FunctionExpr) expressionNode()   {}
func (n *FunctionExpr) Pos() Position     { return n.Position }
func (n *TableExpr) expressionNode()      {}
func (n *TableExpr) Pos() Position        { return n.Position }
