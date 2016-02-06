package compile

import (
	"ast"
	"fmt"
	"llvm.org/llvm/bindings/go/llvm"
)

// IRBuilder := GlobalContext.NewBuilder()

// func compileLet(node *ast.LetNode) llvm.Value {
// 	GlobalContext := llvm.GlobalContext()
// 	return llvm.ConstFloat(llvm.FloatType, node.val)
// }

// func compileFnExpr(node *ast.FnExprNode) llvm.Value {
// 	GlobalContext := llvm.GlobalContext()
// 	return llvm.ConstFloat(llvm.FloatType, node.val)
// }

func compileExpr(node *ast.ExprNode, module llvm.Module) llvm.Value {
	return compileNode(node.Node, module)
}

func compileNumber(node *ast.NumberNode) llvm.Value {
	if node.IsFloat {
		return llvm.ConstFloat(llvm.FloatType(), node.Float64)
	}

	return llvm.ConstFloat(llvm.FloatType(), node.Float64)
}

func compileNode(node interface{}, module llvm.Module) llvm.Value {
	switch n := node.(type) {
	// case *ast.LetNode:
	// 	return compileLet(node)
	// case *ast.FnExprNode:
	// 	return compileFnExpr(node)
	case *ast.NumberNode:
		fmt.Println("compiling number")
		return compileNumber(n)
	case *ast.ExprNode:
		fmt.Println("compiling expression")
		return compileExpr(n, module)
	default:
		return llvm.ConstFloat(llvm.FloatType(), 3)
	}

}
