package lua

// Visitor is used with Walk to traverse an AST. Visit is called for each
// node encountered; if it returns a non-nil Visitor, Walk descends into
// the node's children using the returned Visitor. If Visit returns nil,
// the node's children are skipped. Walk calls v.Visit(nil) after all
// children of a node have been visited, so a Visitor can maintain scope.
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

// walkChildren dispatches to the per-category walker. The Statement /
// Expression interface checks come first so that node types added later
// pick up traversal automatically as long as they satisfy the interface.
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
		walkExprs(v, n.Values)
		Walk(v, n.Body)
	case *FuncDeclStat:
		Walk(v, n.Body)
	case *LocalFuncStat:
		Walk(v, n.Body)
	case *ReturnStat:
		walkExprs(v, n.Values)
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
		Walk(v, n.Body)
	case *TableExpr:
		walkTableFields(v, n.Fields)
	}
}

func walkIfStat(v Visitor, n *IfStat) {
	Walk(v, n.Cond)
	Walk(v, n.Then)
	for _, ei := range n.ElseIfs {
		Walk(v, ei.Cond)
		Walk(v, ei.Body)
	}
	if n.Else != nil {
		Walk(v, n.Else)
	}
}

func walkNumericFor(v Visitor, n *NumericForStat) {
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
