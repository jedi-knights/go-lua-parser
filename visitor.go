package lua

// Visitor is used with Walk to traverse an AST. Visit is called for each
// node encountered; if it returns a non-nil Visitor, Walk descends into
// the node's children using the returned Visitor. If Visit returns nil,
// the node's children are skipped. Walk calls v.Visit(nil) after all
// children of a node have been visited, so a Visitor can maintain scope.
// The return value of Visit(nil) is ignored.
type Visitor interface {
	Visit(node Node) Visitor
}

// Walk traverses an AST rooted at node in depth-first order, calling
// v.Visit on each node.
func Walk(v Visitor, node Node) {
	if node == nil {
		return
	}
	if v = v.Visit(node); v == nil {
		return
	}
	walkChildren(v, node)
	v.Visit(nil)
}

// walkChildren dispatches to the per-category walker. Container nodes
// (Chunk, Block, FuncName) are handled directly; Statement and Expression
// nodes are forwarded to the per-interface walkers.
//
// Adding a new AST node type: add a case here if the node is neither a
// Statement nor an Expression (like FuncName), or add a case in
// walkStatementChildren / walkExpressionChildren otherwise. The default
// branch of the type switches in the per-interface walkers documents that
// leaf nodes are intentional — anything with children must have an
// explicit case.
func walkChildren(v Visitor, node Node) {
	switch n := node.(type) {
	case *Chunk:
		Walk(v, n.Block)
		return
	case *Block:
		for _, s := range n.Statements {
			Walk(v, s)
		}
		if n.Return != nil {
			Walk(v, n.Return)
		}
		return
	case *FuncName:
		Walk(v, n.Root)
		for _, d := range n.Dots {
			Walk(v, d)
		}
		if n.Method != nil {
			Walk(v, n.Method)
		}
		return
	case *ElseIf:
		Walk(v, n.Cond)
		Walk(v, n.Body)
		return
	}
	if stmt, ok := node.(Statement); ok {
		walkStatementChildren(v, stmt)
		return
	}
	if expr, ok := node.(Expression); ok {
		walkExpressionChildren(v, expr)
	}
}

func walkStatementChildren(v Visitor, stmt Statement) {
	switch n := stmt.(type) {
	case *AssignStat:
		walkExprs(v, n.Targets)
		walkExprs(v, n.Values)
	case *LocalAssignStat:
		walkIdents(v, n.Names)
		walkExprs(v, n.Values)
	case *CallStat:
		Walk(v, n.Call)
	case *DoStat:
		Walk(v, n.Body)
	case *WhileStat:
		Walk(v, n.Cond)
		Walk(v, n.Body)
	case *RepeatStat:
		Walk(v, n.Body)
		Walk(v, n.Cond)
	case *IfStat:
		walkIfStat(v, n)
	case *NumericForStat:
		walkNumericFor(v, n)
	case *GenericForStat:
		walkIdents(v, n.Names)
		walkExprs(v, n.Values)
		Walk(v, n.Body)
	case *FuncDeclStat:
		Walk(v, n.Name)
		Walk(v, n.Body)
	case *LocalFuncStat:
		Walk(v, n.Name)
		Walk(v, n.Body)
	case *ReturnStat:
		walkExprs(v, n.Values)
	default:
		// Leaf statements (BreakStat, GotoStat, LabelStat) have no
		// children. Any new Statement with children needs an explicit
		// case above.
	}
}

func walkExpressionChildren(v Visitor, expr Expression) {
	switch n := expr.(type) {
	case *IndexExpr:
		Walk(v, n.Object)
		Walk(v, n.Index)
	case *FieldExpr:
		Walk(v, n.Object)
	case *CallExpr:
		Walk(v, n.Fn)
		walkExprs(v, n.Args)
	case *MethodCallExpr:
		Walk(v, n.Object)
		walkExprs(v, n.Args)
	case *BinaryExpr:
		Walk(v, n.Left)
		Walk(v, n.Right)
	case *UnaryExpr:
		Walk(v, n.Operand)
	case *FunctionExpr:
		walkIdents(v, n.Params)
		Walk(v, n.Body)
	case *TableExpr:
		walkTableFields(v, n.Fields)
	default:
		// Leaf expressions (NilExpr, TrueExpr, FalseExpr, VarargExpr,
		// NumberExpr, StringExpr, Ident) have no children. Any new
		// Expression with children needs an explicit case above.
	}
}

func walkIfStat(v Visitor, n *IfStat) {
	Walk(v, n.Cond)
	Walk(v, n.Then)
	for _, ei := range n.ElseIfs {
		// Walk *ElseIf as a node (see the *ElseIf case in walkChildren)
		// so visitors can observe the clause boundary and its position,
		// not just its cond and body fields in the flat stream.
		Walk(v, ei)
	}
	if n.Else != nil {
		Walk(v, n.Else)
	}
}

func walkNumericFor(v Visitor, n *NumericForStat) {
	Walk(v, n.Name)
	Walk(v, n.Start)
	Walk(v, n.Stop)
	if n.Step != nil {
		Walk(v, n.Step)
	}
	Walk(v, n.Body)
}

func walkExprs(v Visitor, exprs []Expression) {
	for _, e := range exprs {
		Walk(v, e)
	}
}

func walkIdents(v Visitor, idents []*Ident) {
	for _, i := range idents {
		Walk(v, i)
	}
}

func walkTableFields(v Visitor, fields []TableField) {
	for _, f := range fields {
		if f.Key != nil {
			Walk(v, f.Key)
		}
		if f.Value != nil {
			Walk(v, f.Value)
		}
	}
}
