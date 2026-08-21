package lua

import (
	"fmt"
	"testing"
)

// recorder collects node types in the order Walk visits them. Nil visits
// (emitted when Walk finishes a subtree) are ignored so the sequence
// captures only the tree shape.
type recorder struct {
	visited []string
}

func (r *recorder) Visit(node Node) Visitor {
	if node == nil {
		return r
	}
	r.visited = append(r.visited, kindName(node))
	return r
}

func kindName(n Node) string {
	switch n.(type) {
	case *Chunk:
		return "Chunk"
	case *Block:
		return "Block"
	case *AssignStat:
		return "AssignStat"
	case *LocalAssignStat:
		return "LocalAssignStat"
	case *IfStat:
		return "IfStat"
	case *NumericForStat:
		return "NumericForStat"
	case *ReturnStat:
		return "ReturnStat"
	case *BinaryExpr:
		return "BinaryExpr"
	case *NumberExpr:
		return "NumberExpr"
	case *Ident:
		return "Ident"
	case *NilExpr:
		return "NilExpr"
	case *TableExpr:
		return "TableExpr"
	default:
		return "Other"
	}
}

func TestWalkChunkAndBlock(t *testing.T) {
	pos := Position{Filename: "t.lua", Line: 1, Column: 1}
	chunk := &Chunk{
		Filename: "t.lua",
		Block: &Block{
			Position: pos,
			Statements: []Statement{
				&LocalAssignStat{
					Position: pos,
					Names:    []*Ident{{Position: pos, Name: "x"}},
					Values:   []Expression{&NumberExpr{Position: pos, Text: "1"}},
				},
			},
			Return: &ReturnStat{
				Position: pos,
				Values:   []Expression{&Ident{Position: pos, Name: "x"}},
			},
		},
	}
	r := &recorder{}
	Walk(r, chunk)
	got := r.visited
	// LocalAssignStat's declared names are walked before its values, so the
	// first Ident is the binding `x` and the second is the reference in the
	// return statement.
	want := []string{"Chunk", "Block", "LocalAssignStat", "Ident", "NumberExpr", "ReturnStat", "Ident"}
	if len(got) != len(want) {
		t.Fatalf("visited %d nodes, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("node %d: got %s, want %s", i, got[i], w)
		}
	}
}

func TestWalkFuncDeclNameAndParams(t *testing.T) {
	// Arrange — `function a.b:c(x, y) end`
	pos := Position{Filename: "t.lua"}
	stmt := &FuncDeclStat{
		Position: pos,
		Name: &FuncName{
			Position: pos,
			Root:     &Ident{Position: pos, Name: "a"},
			Dots:     []*Ident{{Position: pos, Name: "b"}},
			Method:   &Ident{Position: pos, Name: "c"},
		},
		Body: &FunctionExpr{
			Position: pos,
			Params: []*Ident{
				{Position: pos, Name: "x"},
				{Position: pos, Name: "y"},
			},
			Body: &Block{Position: pos},
		},
	}

	// Act
	names := &identNameCollector{}
	Walk(names, stmt)

	// Assert — declared function name identifiers AND parameter names
	// must all appear; before this fix they were silently dropped.
	want := []string{"a", "b", "c", "x", "y"}
	if len(names.got) != len(want) {
		t.Fatalf("collected %v, want %v", names.got, want)
	}
	for i, w := range want {
		if names.got[i] != w {
			t.Errorf("ident %d: got %q, want %q", i, names.got[i], w)
		}
	}
}

func TestWalkGenericForNames(t *testing.T) {
	// Arrange — `for k, v in pairs(t) do end`
	pos := Position{Filename: "t.lua"}
	stmt := &GenericForStat{
		Position: pos,
		Names: []*Ident{
			{Position: pos, Name: "k"},
			{Position: pos, Name: "v"},
		},
		Values: []Expression{&Ident{Position: pos, Name: "t"}},
		Body:   &Block{Position: pos},
	}

	// Act
	names := &identNameCollector{}
	Walk(names, stmt)

	// Assert — loop-declared names appear before the iterable expression.
	want := []string{"k", "v", "t"}
	if len(names.got) != len(want) {
		t.Fatalf("collected %v, want %v", names.got, want)
	}
	for i, w := range want {
		if names.got[i] != w {
			t.Errorf("ident %d: got %q, want %q", i, names.got[i], w)
		}
	}
}

// identNameCollector records the Name of every *Ident node visited, in walk
// order. Used to verify that Walk descends into binding sites that were
// previously dropped (function-name chains, parameter lists, loop names).
type identNameCollector struct {
	got []string
}

func (c *identNameCollector) Visit(node Node) Visitor {
	if id, ok := node.(*Ident); ok {
		c.got = append(c.got, id.Name)
	}
	return c
}

// TestWalkExercisesAllNodeTypes parses a small but node-diverse Lua program
// and walks the resulting AST, asserting that every walker branch runs at
// least once. The point is coverage of walkStatementChildren /
// walkExpressionChildren / helpers, not the specific visit order.
func TestWalkExercisesAllNodeTypes(t *testing.T) {
	// Arrange — a program that touches: local assign, numeric for,
	// generic for, repeat, while, if/elseif/else, method call,
	// function decl with dotted name, local func, table with keyed and
	// positional fields, binary / unary / index / field expressions,
	// return, break, goto/label.
	src := []byte(`
local x, y = 1, 2
local function inc(n) return n + 1 end
function m.helper:call(a, b) return a * b end
for i = 1, 10, 2 do x = x + i end
for k, v in pairs(x) do print(k, v) end
repeat y = y - 1 until y == 0
while x > 0 do x = x - 1; break end
local t = { 10, 20, key = "v", [x] = "expr" }
if x == 0 then
    print("zero")
elseif x < 0 then
    print("neg")
else
    print("pos")
end
local a = -x
local b = t[1]
local c = t.key
local d = t:concat(",")
::retry::
goto retry
return t
`)

	chunk, err := Parse("all.lua", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Act
	counter := &nodeKindCounter{seen: map[string]int{}}
	Walk(counter, chunk)

	// Assert — a representative subset must appear. Missing any one of
	// these means the corresponding walker case did not run.
	want := []string{
		"*lua.LocalAssignStat",
		"*lua.LocalFuncStat",
		"*lua.FuncDeclStat",
		"*lua.FuncName",
		"*lua.NumericForStat",
		"*lua.GenericForStat",
		"*lua.RepeatStat",
		"*lua.WhileStat",
		"*lua.IfStat",
		"*lua.BinaryExpr",
		"*lua.UnaryExpr",
		"*lua.IndexExpr",
		"*lua.FieldExpr",
		"*lua.CallExpr",
		"*lua.MethodCallExpr",
		"*lua.TableExpr",
		"*lua.FunctionExpr",
		"*lua.ReturnStat",
		"*lua.BreakStat",
		"*lua.GotoStat",
		"*lua.LabelStat",
	}
	for _, w := range want {
		if counter.seen[w] == 0 {
			t.Errorf("Walk never visited %s (walker case is unreachable)", w)
		}
	}
}

// nodeKindCounter counts occurrences of each concrete node type visited.
type nodeKindCounter struct {
	seen map[string]int
}

func (c *nodeKindCounter) Visit(node Node) Visitor {
	if node == nil {
		return c
	}
	c.seen[fmt.Sprintf("%T", node)]++
	return c
}

func TestWalkIfAndBinary(t *testing.T) {
	pos := Position{Filename: "t.lua", Line: 1, Column: 1}
	ifStat := &IfStat{
		Position: pos,
		Cond: &BinaryExpr{
			Position: pos,
			Op:       "<",
			Left:     &Ident{Position: pos, Name: "n"},
			Right:    &NumberExpr{Position: pos, Text: "10"},
		},
		Then: &Block{Position: pos},
	}
	r := &recorder{}
	Walk(r, ifStat)
	// If-then subtree: IfStat, BinaryExpr, Ident, NumberExpr, Block.
	want := []string{"IfStat", "BinaryExpr", "Ident", "NumberExpr", "Block"}
	if len(r.visited) != len(want) {
		t.Fatalf("visited = %v, want %v", r.visited, want)
	}
	for i, w := range want {
		if r.visited[i] != w {
			t.Errorf("node %d: got %s, want %s", i, r.visited[i], w)
		}
	}
}

func TestWalkNilNode(t *testing.T) {
	r := &recorder{}
	Walk(r, nil)
	if len(r.visited) != 0 {
		t.Errorf("walking nil should visit nothing, got %v", r.visited)
	}
}

// stopVisitor returns nil after the first visit, so Walk should not descend.
type stopVisitor struct{ count int }

func (s *stopVisitor) Visit(node Node) Visitor {
	if node == nil {
		return s
	}
	s.count++
	return nil
}

func TestWalkStopsWhenVisitReturnsNil(t *testing.T) {
	pos := Position{Filename: "t.lua"}
	chunk := &Chunk{Filename: "t.lua", Block: &Block{Position: pos}}
	v := &stopVisitor{}
	Walk(v, chunk)
	if v.count != 1 {
		t.Errorf("visited %d nodes, want 1 (stopped at root)", v.count)
	}
}

func TestTokenKindString(t *testing.T) {
	if got := TokenLocal.String(); got != "local" {
		t.Errorf("TokenLocal.String() = %q, want %q", got, "local")
	}
	if got := TokenEq.String(); got != "==" {
		t.Errorf("TokenEq.String() = %q, want %q", got, "==")
	}
	if got := TokenKind(9999).String(); got == "" {
		t.Error("unknown TokenKind should produce a non-empty string")
	}
}

func TestPositionString(t *testing.T) {
	if got := (Position{Line: 3, Column: 5}).String(); got != "3:5" {
		t.Errorf("Position.String() = %q, want %q", got, "3:5")
	}
	if got := (Position{Filename: "x.lua", Line: 3, Column: 5}).String(); got != "x.lua:3:5" {
		t.Errorf("Position.String() = %q, want %q", got, "x.lua:3:5")
	}
}
