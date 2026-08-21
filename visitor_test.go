package lua

import "testing"

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
	want := []string{"Chunk", "Block", "LocalAssignStat", "NumberExpr", "ReturnStat", "Ident"}
	if len(got) != len(want) {
		t.Fatalf("visited %d nodes, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("node %d: got %s, want %s", i, got[i], w)
		}
	}
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
