package compile

import (
	"ast"
	"fmt"
	"token"
	"core"
)

func translateNumber(node *ast.NumberNode) (core.CoreExpr) {
	return core.ENum{node.IsInt, node.IsUint, node.IsFloat, node.Int64, node.Uint64, node.Float64, node.Text}	
}

func translateVariable(node *ast.VariableNode) (core.CoreExpr) {
	return core.EVar(node.Ident)
}

func translateIf(node *ast.IfNode) (core.CoreExpr) {
	// cond := translateExpr(node.Cond)
	// then := translateExpr(node.Then)
	// els := translateExpr(node.Else)

	return core.ScDefn{} //TODO: Incomplete fn require support of prelude in core-ast 
}

func translateBinaryExpr(node *ast.BinaryExprNode) (core.CoreExpr) {
	left := translateExpr(node.Left)
	right := translateExpr(node.Right)
	var oper string
	switch node.Op {
		case token.ADD:
			oper = "+"
		case token.SUB:
			oper = "-"
		case token.MUL:
			oper = "*"
		case token.QUO:
			oper = "/"
		case token.REM:
			oper = "%"
		case token.EQL:
			oper = "=="
		default:
			oper = ""
	}
	fmt.Println(oper)
	return core.EAp{core.EAp{core.EVar(oper), left}, right} // TODO: Incomplete fn require support of prelude in core-ast
}

func translateDefnNode(node *ast.DefnNode) core.Defn {
	expr := translateExpr(node.Expr)
	return core.Defn{core.Name(node.Var), expr}
}

func translateLet(node *ast.LetNode) (core.CoreExpr) {
	expr := translateExpr(node.Expr)
	defns := []core.Defn{}
	for _, defn := range node.Defns {
		defns = append(defns, translateDefnNode(defn))
	}
	return core.ELet{false, defns, expr}
}

func translateExpr(node interface{}) (core.CoreExpr) {
	switch n := node.(type) {
		case *ast.NumberNode:
			return translateNumber(n)
		case *ast.BinaryExprNode:
			return translateBinaryExpr(n)
		case *ast.LetNode:
			return translateLet(n)
		case *ast.IfNode:
			return translateIf(n)
		case *ast.VariableNode:
			return translateVariable(n)
		case *ast.FnExprNode:
			return translateFnExpr(n)
		default:
			return core.ENum{true, false, false, 4, 4, 4, "4"}
	}
}

func translateFnStatement(node *ast.FnExprNode)  (core.ScDefn) {
	params := []core.Name{}
	for _, param := range node.Params{
		params = append(params, core.Name(param))
	}
	return core.ScDefn{core.Name("f1"), params, translateExpr(node.Body)}
}

func translateFnExpr(node *ast.FnExprNode) (core.CoreExpr) {
	params := []core.Name{}
	for _, param := range node.Params{
		params = append(params, core.Name(param))
	}
	return core.ELam{params, translateExpr(node.Body)}
}

func translateNode(node interface{}) (core.CoreExpr) {
	switch n := node.(type) {
		case *ast.LetNode:
			return translateLet(n)
		case *ast.FnExprNode:
			return translateFnExpr(n)
		default:
			return translateExpr(n)
	}
}



