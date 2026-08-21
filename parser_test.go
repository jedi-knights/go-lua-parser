package lua_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	lua "github.com/jedi-knights/go-lua-parser"
)

// mustParse parses src and fails the test if lexing or parsing reports any
// error. It returns the resulting Chunk for further inspection.
func mustParse(t *testing.T, src string) *lua.Chunk {
	t.Helper()
	chunk, err := lua.Parse("t.lua", []byte(src))
	if err != nil {
		t.Fatalf("parse error: %v\nsource: %s", err, src)
	}
	return chunk
}

// singleStat asserts the chunk has exactly one statement and returns it.
func singleStat(t *testing.T, chunk *lua.Chunk) lua.Statement {
	t.Helper()
	if chunk.Block == nil {
		t.Fatal("nil block")
	}
	if len(chunk.Block.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(chunk.Block.Statements))
	}
	return chunk.Block.Statements[0]
}

// firstExpr extracts the first value expression from `local _ = <expr>` so
// tests can inspect expression parsing in isolation.
func firstExpr(t *testing.T, src string) lua.Expression {
	t.Helper()
	stat := singleStat(t, mustParse(t, "local _ = "+src))
	la, ok := stat.(*lua.LocalAssignStat)
	if !ok {
		t.Fatalf("expected LocalAssignStat, got %T", stat)
	}
	if len(la.Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(la.Values))
	}
	return la.Values[0]
}

// --- baseline -------------------------------------------------------------

func TestParseEmpty(t *testing.T) {
	chunk := mustParse(t, "")
	if len(chunk.Block.Statements) != 0 {
		t.Errorf("expected 0 statements, got %d", len(chunk.Block.Statements))
	}
}

func TestParseCommentsOnly(t *testing.T) {
	chunk := mustParse(t, "-- comment\n--[[ block ]]\n")
	if len(chunk.Block.Statements) != 0 {
		t.Errorf("expected 0 statements, got %d", len(chunk.Block.Statements))
	}
}

func TestParsePropagatesLexerErrors(t *testing.T) {
	_, err := lua.Parse("t.lua", []byte(`"unterminated`))
	if err == nil {
		t.Fatal("expected error from unterminated string")
	}
}

// --- local + assignment ---------------------------------------------------

func TestParseLocalSingle(t *testing.T) {
	stat := singleStat(t, mustParse(t, "local x = 1"))
	la, ok := stat.(*lua.LocalAssignStat)
	if !ok {
		t.Fatalf("got %T, want *LocalAssignStat", stat)
	}
	if len(la.Names) != 1 || la.Names[0].Name != "x" {
		t.Errorf("names = %v", la.Names)
	}
	if len(la.Values) != 1 {
		t.Fatalf("values = %d, want 1", len(la.Values))
	}
	if n, ok := la.Values[0].(*lua.NumberExpr); !ok || n.Text != "1" {
		t.Errorf("value = %#v, want NumberExpr{1}", la.Values[0])
	}
}

func TestParseLocalMulti(t *testing.T) {
	la := singleStat(t, mustParse(t, "local a, b, c = 1, 2, 3")).(*lua.LocalAssignStat)
	if len(la.Names) != 3 || len(la.Values) != 3 {
		t.Fatalf("names=%d values=%d, want 3/3", len(la.Names), len(la.Values))
	}
}

func TestParseLocalNoValues(t *testing.T) {
	la := singleStat(t, mustParse(t, "local x")).(*lua.LocalAssignStat)
	if len(la.Names) != 1 || len(la.Values) != 0 {
		t.Errorf("names=%d values=%d, want 1/0", len(la.Names), len(la.Values))
	}
}

func TestParseAssignment(t *testing.T) {
	as := singleStat(t, mustParse(t, "x = 1")).(*lua.AssignStat)
	if len(as.Targets) != 1 || len(as.Values) != 1 {
		t.Errorf("targets=%d values=%d, want 1/1", len(as.Targets), len(as.Values))
	}
	if id, ok := as.Targets[0].(*lua.Ident); !ok || id.Name != "x" {
		t.Errorf("target = %#v", as.Targets[0])
	}
}

func TestParseMultiAssign(t *testing.T) {
	as := singleStat(t, mustParse(t, "a, t[1], obj.field = 1, 2, 3")).(*lua.AssignStat)
	if len(as.Targets) != 3 || len(as.Values) != 3 {
		t.Fatalf("targets=%d values=%d, want 3/3", len(as.Targets), len(as.Values))
	}
	if _, ok := as.Targets[1].(*lua.IndexExpr); !ok {
		t.Errorf("target[1] = %T, want *IndexExpr", as.Targets[1])
	}
	if _, ok := as.Targets[2].(*lua.FieldExpr); !ok {
		t.Errorf("target[2] = %T, want *FieldExpr", as.Targets[2])
	}
}

// --- control flow ---------------------------------------------------------

func TestParseIfElseif(t *testing.T) {
	src := `if x then y = 1 elseif z then y = 2 else y = 3 end`
	stat := singleStat(t, mustParse(t, src)).(*lua.IfStat)
	if len(stat.ElseIfs) != 1 {
		t.Errorf("elseifs = %d, want 1", len(stat.ElseIfs))
	}
	if stat.Else == nil {
		t.Error("expected else block")
	}
}

func TestParseWhile(t *testing.T) {
	w := singleStat(t, mustParse(t, "while x < 10 do x = x + 1 end")).(*lua.WhileStat)
	if _, ok := w.Cond.(*lua.BinaryExpr); !ok {
		t.Errorf("cond = %T, want *BinaryExpr", w.Cond)
	}
	if len(w.Body.Statements) != 1 {
		t.Errorf("body statements = %d, want 1", len(w.Body.Statements))
	}
}

func TestParseRepeat(t *testing.T) {
	r := singleStat(t, mustParse(t, "repeat x = x - 1 until x == 0")).(*lua.RepeatStat)
	if _, ok := r.Cond.(*lua.BinaryExpr); !ok {
		t.Errorf("cond = %T", r.Cond)
	}
}

func TestParseNumericFor(t *testing.T) {
	f := singleStat(t, mustParse(t, "for i = 1, 10, 2 do print(i) end")).(*lua.NumericForStat)
	if f.Name.Name != "i" {
		t.Errorf("name = %q", f.Name.Name)
	}
	if f.Step == nil {
		t.Error("step should be non-nil")
	}
}

func TestParseNumericForNoStep(t *testing.T) {
	f := singleStat(t, mustParse(t, "for i = 1, 10 do end")).(*lua.NumericForStat)
	if f.Step != nil {
		t.Errorf("step = %#v, want nil", f.Step)
	}
}

func TestParseGenericFor(t *testing.T) {
	f := singleStat(t, mustParse(t, "for k, v in pairs(t) do end")).(*lua.GenericForStat)
	if len(f.Names) != 2 {
		t.Errorf("names = %d, want 2", len(f.Names))
	}
	if len(f.Values) != 1 {
		t.Errorf("values = %d, want 1", len(f.Values))
	}
}

func TestParseDoBlock(t *testing.T) {
	d := singleStat(t, mustParse(t, "do local x = 1 end")).(*lua.DoStat)
	if len(d.Body.Statements) != 1 {
		t.Errorf("body statements = %d, want 1", len(d.Body.Statements))
	}
}

func TestParseBreakGotoLabel(t *testing.T) {
	src := "::top:: for i = 1, 10 do if i > 5 then break end goto top end"
	chunk := mustParse(t, src)
	if len(chunk.Block.Statements) != 2 {
		t.Fatalf("stmts = %d, want 2 (label, for)", len(chunk.Block.Statements))
	}
	if _, ok := chunk.Block.Statements[0].(*lua.LabelStat); !ok {
		t.Errorf("stmt[0] = %T, want *LabelStat", chunk.Block.Statements[0])
	}
}

// --- function forms -------------------------------------------------------

func TestParseFuncDeclSimple(t *testing.T) {
	fd := singleStat(t, mustParse(t, "function foo() return 1 end")).(*lua.FuncDeclStat)
	if fd.Name.Root.Name != "foo" {
		t.Errorf("root = %q", fd.Name.Root.Name)
	}
	if fd.Body.Body.Return == nil {
		t.Error("expected return in body")
	}
}

func TestParseFuncDeclDotted(t *testing.T) {
	fd := singleStat(t, mustParse(t, "function a.b.c() end")).(*lua.FuncDeclStat)
	if len(fd.Name.Dots) != 2 {
		t.Errorf("dots = %d, want 2", len(fd.Name.Dots))
	}
	if fd.Name.Method != nil {
		t.Errorf("method = %v, want nil", fd.Name.Method)
	}
}

func TestParseFuncDeclMethod(t *testing.T) {
	fd := singleStat(t, mustParse(t, "function obj:greet(msg) end")).(*lua.FuncDeclStat)
	if fd.Name.Method == nil || fd.Name.Method.Name != "greet" {
		t.Errorf("method = %v", fd.Name.Method)
	}
	if len(fd.Body.Params) != 1 || fd.Body.Params[0].Name != "msg" {
		t.Errorf("params = %v", fd.Body.Params)
	}
}

func TestParseLocalFunction(t *testing.T) {
	lf := singleStat(t, mustParse(t, "local function f(a, b, ...) return a end")).(*lua.LocalFuncStat)
	if lf.Name.Name != "f" {
		t.Errorf("name = %q", lf.Name.Name)
	}
	if len(lf.Body.Params) != 2 || !lf.Body.IsVararg {
		t.Errorf("params = %v, vararg = %v", lf.Body.Params, lf.Body.IsVararg)
	}
}

func TestParseCallStatement(t *testing.T) {
	cs := singleStat(t, mustParse(t, "print(1, 2, 3)")).(*lua.CallStat)
	if _, ok := cs.Call.(*lua.CallExpr); !ok {
		t.Errorf("call = %T", cs.Call)
	}
}

func TestParseMethodCall(t *testing.T) {
	cs := singleStat(t, mustParse(t, "obj:greet(msg)")).(*lua.CallStat)
	mc, ok := cs.Call.(*lua.MethodCallExpr)
	if !ok {
		t.Fatalf("call = %T, want *MethodCallExpr", cs.Call)
	}
	if mc.Method != "greet" {
		t.Errorf("method = %q", mc.Method)
	}
}

func TestParseReturnEmpty(t *testing.T) {
	chunk := mustParse(t, "return")
	if chunk.Block.Return == nil || len(chunk.Block.Return.Values) != 0 {
		t.Errorf("expected empty return, got %+v", chunk.Block.Return)
	}
}

func TestParseReturnValues(t *testing.T) {
	chunk := mustParse(t, "return 1, 2, x")
	if len(chunk.Block.Return.Values) != 3 {
		t.Errorf("values = %d, want 3", len(chunk.Block.Return.Values))
	}
}

// --- prefix / chained access ----------------------------------------------

func TestParseFieldChain(t *testing.T) {
	e := firstExpr(t, "a.b.c")
	f, ok := e.(*lua.FieldExpr)
	if !ok {
		t.Fatalf("outer = %T", e)
	}
	if f.Name != "c" {
		t.Errorf("outer field = %q", f.Name)
	}
	inner, ok := f.Object.(*lua.FieldExpr)
	if !ok || inner.Name != "b" {
		t.Errorf("inner = %#v", f.Object)
	}
}

func TestParseIndexChain(t *testing.T) {
	e := firstExpr(t, "a[1][2]")
	outer, ok := e.(*lua.IndexExpr)
	if !ok {
		t.Fatalf("outer = %T", e)
	}
	if _, ok := outer.Object.(*lua.IndexExpr); !ok {
		t.Errorf("inner = %T, want *IndexExpr", outer.Object)
	}
}

func TestParseCallChain(t *testing.T) {
	e := firstExpr(t, "f(1)(2)(3)")
	c, ok := e.(*lua.CallExpr)
	if !ok {
		t.Fatalf("outer = %T", e)
	}
	if _, ok := c.Fn.(*lua.CallExpr); !ok {
		t.Errorf("inner call = %T", c.Fn)
	}
}

func TestParseParenthesizedPrefix(t *testing.T) {
	// `(f)()` — the () around f still returns a callable.
	e := firstExpr(t, "(f)()")
	if _, ok := e.(*lua.CallExpr); !ok {
		t.Errorf("expr = %T, want *CallExpr", e)
	}
}

// --- call argument forms --------------------------------------------------

func TestParseCallWithStringSugar(t *testing.T) {
	e := firstExpr(t, `require "mod"`)
	c := e.(*lua.CallExpr)
	if len(c.Args) != 1 {
		t.Fatalf("args = %d, want 1", len(c.Args))
	}
	if s, ok := c.Args[0].(*lua.StringExpr); !ok || s.Value != "mod" {
		t.Errorf("arg = %#v", c.Args[0])
	}
}

func TestParseCallWithTableSugar(t *testing.T) {
	e := firstExpr(t, `f {1, 2, 3}`)
	c := e.(*lua.CallExpr)
	if len(c.Args) != 1 {
		t.Fatalf("args = %d, want 1", len(c.Args))
	}
	if _, ok := c.Args[0].(*lua.TableExpr); !ok {
		t.Errorf("arg = %T, want *TableExpr", c.Args[0])
	}
}

// --- table constructor ----------------------------------------------------

func TestParseTablePositional(t *testing.T) {
	e := firstExpr(t, "{10, 20, 30}")
	tab := e.(*lua.TableExpr)
	if len(tab.Fields) != 3 {
		t.Fatalf("fields = %d, want 3", len(tab.Fields))
	}
	for i, f := range tab.Fields {
		if f.Key != nil || f.KeyName != "" {
			t.Errorf("field %d unexpectedly keyed: %+v", i, f)
		}
	}
}

func TestParseTableKeyed(t *testing.T) {
	e := firstExpr(t, `{name = "a", age = 42, [1] = "first"}`)
	tab := e.(*lua.TableExpr)
	if len(tab.Fields) != 3 {
		t.Fatalf("fields = %d, want 3", len(tab.Fields))
	}
	if tab.Fields[0].KeyName != "name" {
		t.Errorf("field[0] KeyName = %q", tab.Fields[0].KeyName)
	}
	if tab.Fields[2].Key == nil {
		t.Errorf("field[2] should have expression key")
	}
}

func TestParseTableSemicolonSeparator(t *testing.T) {
	e := firstExpr(t, "{1; 2; 3}")
	tab := e.(*lua.TableExpr)
	if len(tab.Fields) != 3 {
		t.Errorf("fields = %d, want 3", len(tab.Fields))
	}
}

func TestParseTableTrailingSeparator(t *testing.T) {
	e := firstExpr(t, "{1, 2,}")
	tab := e.(*lua.TableExpr)
	if len(tab.Fields) != 2 {
		t.Errorf("fields = %d, want 2", len(tab.Fields))
	}
}

// --- expressions: precedence and associativity ----------------------------

func TestParsePrecedenceMultThenAdd(t *testing.T) {
	// 1 + 2 * 3 should parse as 1 + (2 * 3)
	e := firstExpr(t, "1 + 2 * 3").(*lua.BinaryExpr)
	if e.Op != "+" {
		t.Errorf("outer op = %q, want +", e.Op)
	}
	right, ok := e.Right.(*lua.BinaryExpr)
	if !ok || right.Op != "*" {
		t.Errorf("right = %#v", e.Right)
	}
}

func TestParsePrecedenceAndOr(t *testing.T) {
	// a or b and c should parse as a or (b and c)
	e := firstExpr(t, "a or b and c").(*lua.BinaryExpr)
	if e.Op != "or" {
		t.Errorf("outer op = %q, want or", e.Op)
	}
	if right, ok := e.Right.(*lua.BinaryExpr); !ok || right.Op != "and" {
		t.Errorf("right = %#v", e.Right)
	}
}

func TestParseConcatRightAssoc(t *testing.T) {
	// a..b..c should parse as a..(b..c)
	e := firstExpr(t, "a..b..c").(*lua.BinaryExpr)
	if e.Op != ".." {
		t.Errorf("outer op = %q", e.Op)
	}
	right, ok := e.Right.(*lua.BinaryExpr)
	if !ok || right.Op != ".." {
		t.Fatalf("right = %#v", e.Right)
	}
	if id, ok := e.Left.(*lua.Ident); !ok || id.Name != "a" {
		t.Errorf("left = %#v", e.Left)
	}
}

func TestParseCaretRightAssoc(t *testing.T) {
	// 2^3^2 should parse as 2^(3^2)
	e := firstExpr(t, "2^3^2").(*lua.BinaryExpr)
	if e.Op != "^" {
		t.Errorf("outer op = %q", e.Op)
	}
	if right, ok := e.Right.(*lua.BinaryExpr); !ok || right.Op != "^" {
		t.Errorf("right = %#v", e.Right)
	}
}

func TestParseUnaryLowerThanCaret(t *testing.T) {
	// -2^2 should parse as -(2^2)
	e := firstExpr(t, "-2^2").(*lua.UnaryExpr)
	if e.Op != "-" {
		t.Errorf("op = %q", e.Op)
	}
	if b, ok := e.Operand.(*lua.BinaryExpr); !ok || b.Op != "^" {
		t.Errorf("operand = %#v", e.Operand)
	}
}

func TestParseUnaryChain(t *testing.T) {
	e := firstExpr(t, "not not x").(*lua.UnaryExpr)
	if e.Op != "not" {
		t.Errorf("outer op = %q", e.Op)
	}
	if inner, ok := e.Operand.(*lua.UnaryExpr); !ok || inner.Op != "not" {
		t.Errorf("inner = %#v", e.Operand)
	}
}

func TestParseLength(t *testing.T) {
	e := firstExpr(t, "#t").(*lua.UnaryExpr)
	if e.Op != "#" {
		t.Errorf("op = %q", e.Op)
	}
}

func TestParseIntegerDivision(t *testing.T) {
	e := firstExpr(t, "a // b").(*lua.BinaryExpr)
	if e.Op != "//" {
		t.Errorf("op = %q, want //", e.Op)
	}
}

// --- literals -------------------------------------------------------------

func TestParseLiterals(t *testing.T) {
	cases := []struct {
		src  string
		want string // type name
	}{
		{"nil", "*lua.NilExpr"},
		{"true", "*lua.TrueExpr"},
		{"false", "*lua.FalseExpr"},
		{"42", "*lua.NumberExpr"},
		{`"hi"`, "*lua.StringExpr"},
	}
	for _, c := range cases {
		e := firstExpr(t, c.src)
		got := typeName(e)
		if got != c.want {
			t.Errorf("%q: got %s, want %s", c.src, got, c.want)
		}
	}
}

func TestParseVarargExpr(t *testing.T) {
	// `...` is legal inside a vararg function body.
	stat := singleStat(t, mustParse(t, "local function f(...) return ... end")).(*lua.LocalFuncStat)
	if stat.Body.Body.Return == nil {
		t.Fatal("expected return")
	}
	if _, ok := stat.Body.Body.Return.Values[0].(*lua.VarargExpr); !ok {
		t.Errorf("return value = %T", stat.Body.Body.Return.Values[0])
	}
}

// --- call arg sugars ------------------------------------------------------

func TestParseCallWithLongStringSugar(t *testing.T) {
	// Long-bracket string as a call argument (no parens): `require [[mod]]`.
	// The lexer emits TokenString for both short and long strings, so this
	// exercises the sugar path end-to-end.
	e := firstExpr(t, "require [[mod]]")
	c, ok := e.(*lua.CallExpr)
	if !ok {
		t.Fatalf("expr = %T, want *CallExpr", e)
	}
	if len(c.Args) != 1 {
		t.Fatalf("args = %d, want 1", len(c.Args))
	}
	s, ok := c.Args[0].(*lua.StringExpr)
	if !ok {
		t.Fatalf("arg = %T, want *StringExpr", c.Args[0])
	}
	if s.Value != "mod" {
		t.Errorf("arg value = %q, want %q", s.Value, "mod")
	}
}

// --- error cases ----------------------------------------------------------

func TestParseErrorMissingEnd(t *testing.T) {
	_, err := lua.Parse("t.lua", []byte("do local x = 1"))
	if err == nil {
		t.Fatal("expected error for unterminated do-block")
	}
}

func TestParseErrorBadStatement(t *testing.T) {
	// Bare `1 = 2` is not a valid assignment target.
	_, err := lua.Parse("t.lua", []byte("1 = 2"))
	if err == nil {
		t.Fatal("expected error for invalid assignment target")
	}
}

// --- error recovery -------------------------------------------------------

func TestParseRecoveryAfterBadStatement(t *testing.T) {
	// First statement is nonsense (two identifiers, no operator). The parser
	// should record an error, sync to the next statement boundary (`local`),
	// and still return the local assignment as a valid AST node.
	src := "xx yy = 1\nlocal z = 2"
	chunk, err := lua.Parse("t.lua", []byte(src))
	if err == nil {
		t.Fatal("expected an error from bad first statement")
	}
	if chunk == nil || chunk.Block == nil {
		t.Fatal("expected non-nil chunk and block despite errors")
	}
	found := false
	for _, s := range chunk.Block.Statements {
		if la, ok := s.(*lua.LocalAssignStat); ok {
			if len(la.Names) == 1 && la.Names[0].Name == "z" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected recovery to yield `local z = 2`, got %d statements", len(chunk.Block.Statements))
	}
}

func TestParseRecoveryDoesNotCascade(t *testing.T) {
	// Two independent bad statements should produce roughly two errors —
	// not a runaway cascade. We check that the count is bounded, not exact,
	// because expect() calls can inflate the count slightly.
	src := "xx yy\nlocal z = 2\naa bb"
	_, err := lua.Parse("t.lua", []byte(src))
	if err == nil {
		t.Fatal("expected errors")
	}
	errs, ok := err.(lua.SyntaxErrors)
	if !ok {
		t.Fatalf("err = %T, want SyntaxErrors", err)
	}
	if len(errs) > 6 {
		t.Errorf("cascade suspected: %d errors from 2 bad statements: %v", len(errs), errs)
	}
}

func TestParseRecoveryOnLexerErrorTerminates(t *testing.T) {
	// Arrange — a source with an invalid byte at statement position. Before
	// TokenError was a sync point, parseBlock would drive sync() through
	// every subsequent character before reaching EOF.
	src := []byte("local x = 1\n\x00\x00\x00\nlocal y = 2")

	// Act — should return promptly, not hang.
	done := make(chan struct{})
	var chunk *lua.Chunk
	var err error
	go func() {
		chunk, err = lua.Parse("t.lua", src)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Parse did not terminate on invalid bytes")
	}

	// Assert — errors surfaced, chunk still returned, both good statements
	// visible after recovery.
	if err == nil {
		t.Fatal("expected errors on invalid bytes")
	}
	if chunk == nil || chunk.Block == nil {
		t.Fatal("expected a partial chunk")
	}
	if got := len(chunk.Block.Statements); got < 2 {
		t.Errorf("expected both good statements to survive recovery, got %d: %+v", got, chunk.Block.Statements)
	}
}

func TestParseBoundsParamList(t *testing.T) {
	// Arrange — a function decl with > MaxListItems parameters.
	var b strings.Builder
	b.WriteString("local f = function(")
	for i := 0; i < lua.MaxListItems+50; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "p%d", i)
	}
	b.WriteString(") end")

	// Act
	chunk, err := lua.Parse("t.lua", []byte(b.String()))

	// Assert — error mentions the cap; params are truncated to the cap.
	if err == nil {
		t.Fatal("expected an error for oversized param list")
	}
	if !strings.Contains(err.Error(), "parameter list exceeds") {
		t.Errorf("expected 'parameter list exceeds' in error, got: %v", err)
	}
	assign, ok := chunk.Block.Statements[0].(*lua.LocalAssignStat)
	if !ok {
		t.Fatalf("unexpected first stmt: %T", chunk.Block.Statements[0])
	}
	fn := assign.Values[0].(*lua.FunctionExpr)
	if len(fn.Params) > lua.MaxListItems {
		t.Errorf("params not capped: got %d, want <= %d", len(fn.Params), lua.MaxListItems)
	}
}

func TestParseBoundsExprList(t *testing.T) {
	// Arrange — a return statement with > MaxListItems expressions.
	var b strings.Builder
	b.WriteString("return ")
	for i := 0; i < lua.MaxListItems+50; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("1")
	}

	// Act
	_, err := lua.Parse("t.lua", []byte(b.String()))

	// Assert
	if err == nil || !strings.Contains(err.Error(), "expression list exceeds") {
		t.Errorf("expected 'expression list exceeds' error, got: %v", err)
	}
}

func TestParseBoundsTableConstructor(t *testing.T) {
	// Arrange — a table literal with > MaxListItems fields.
	var b strings.Builder
	b.WriteString("local t = {")
	for i := 0; i < lua.MaxListItems+50; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("1")
	}
	b.WriteString("}")

	// Act
	_, err := lua.Parse("t.lua", []byte(b.String()))

	// Assert
	if err == nil || !strings.Contains(err.Error(), "table constructor exceeds") {
		t.Errorf("expected 'table constructor exceeds' error, got: %v", err)
	}
}

func TestParseBoundsFuncNameDots(t *testing.T) {
	// Arrange — a function declaration with > MaxListItems dotted components.
	var b strings.Builder
	b.WriteString("function a")
	for i := 0; i < lua.MaxListItems+50; i++ {
		b.WriteString(".b")
	}
	b.WriteString("() end")

	// Act
	_, err := lua.Parse("t.lua", []byte(b.String()))

	// Assert
	if err == nil || !strings.Contains(err.Error(), "function name exceeds") {
		t.Errorf("expected 'function name exceeds' error, got: %v", err)
	}
}

func TestParseBoundsElseIfChain(t *testing.T) {
	// Arrange — an if with > MaxListItems elseif clauses.
	var b strings.Builder
	b.WriteString("if x == 0 then")
	for i := 0; i < lua.MaxListItems+50; i++ {
		b.WriteString(" elseif x == 1 then")
	}
	b.WriteString(" end")

	// Act
	_, err := lua.Parse("t.lua", []byte(b.String()))

	// Assert
	if err == nil || !strings.Contains(err.Error(), "elseif chain exceeds") {
		t.Errorf("expected 'elseif chain exceeds' error, got: %v", err)
	}
}

func TestParseAssignTargetsNilGuarded(t *testing.T) {
	// Arrange — `a, 1 = x` triggers parsePrefixExp to record an error and
	// return nil for the second target. Without the nil-guard, that nil
	// would end up in Targets and panic on Walk / Pos().
	src := []byte("a, 1 = x")

	// Act
	chunk, _ := lua.Parse("t.lua", src)

	// Assert — the produced AST must have no nil elements in any
	// AssignStat.Targets slice. Walk the tree; a bad append would panic
	// in Walk before we get here.
	if chunk == nil || chunk.Block == nil || len(chunk.Block.Statements) == 0 {
		t.Fatal("expected a partial chunk")
	}
	assign, ok := chunk.Block.Statements[0].(*lua.AssignStat)
	if !ok {
		t.Fatalf("first stmt: got %T, want *lua.AssignStat", chunk.Block.Statements[0])
	}
	for i, tgt := range assign.Targets {
		if tgt == nil {
			t.Errorf("Targets[%d] is nil — nil-guard regressed", i)
		}
	}
	// Walk must not panic.
	lua.Walk(&noopVisitor{}, chunk)
}

type noopVisitor struct{}

func (noopVisitor) Visit(lua.Node) lua.Visitor { return noopVisitor{} }

// --- helpers --------------------------------------------------------------

func typeName(v any) string {
	if v == nil {
		return "<nil>"
	}
	// Reflection-free type name: use %T formatting.
	return sprintType(v)
}

// sprintType uses fmt indirectly via a small map so we avoid importing fmt
// in this test file (each test that needs formatting uses t.Errorf).
func sprintType(v any) string {
	switch v.(type) {
	case *lua.NilExpr:
		return "*lua.NilExpr"
	case *lua.TrueExpr:
		return "*lua.TrueExpr"
	case *lua.FalseExpr:
		return "*lua.FalseExpr"
	case *lua.NumberExpr:
		return "*lua.NumberExpr"
	case *lua.StringExpr:
		return "*lua.StringExpr"
	case *lua.VarargExpr:
		return "*lua.VarargExpr"
	case *lua.Ident:
		return "*lua.Ident"
	case *lua.BinaryExpr:
		return "*lua.BinaryExpr"
	case *lua.UnaryExpr:
		return "*lua.UnaryExpr"
	case *lua.CallExpr:
		return "*lua.CallExpr"
	case *lua.TableExpr:
		return "*lua.TableExpr"
	default:
		return "unknown"
	}
}
