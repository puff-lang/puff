package compile

import (
	"ast"
	"fmt"
	"llvm.org/llvm/bindings/go/llvm"
	"token"
)

// IRBuilder := GlobalContext.NewBuilder()

//https://onedrive.live.com/redir?resid=E8C19698FE429F68!164&authkey=!AL5uGtAPIElr2L8&ithint=file%2cpptx

// func compileLet(node *ast.LetNode) llvm.Value {
// 	GlobalContext := llvm.GlobalContext()
// 	return llvm.ConstFloat(llvm.FloatType, node.val)
// }

// func compileFnExpr(node *ast.FnExprNode) llvm.Value {
// 	GlobalContext := llvm.GlobalContext()
// 	return llvm.ConstFloat(llvm.FloatType, node.val)
// }

func compileExpr(node interface{}, module llvm.Module) llvm.Value {
	switch n := node.(type) {
	// case *ast.LetNode:
	// 	return compileLet(node)
	// case *ast.FnExprNode:
	// 	retu	rn compileFnExpr(node)
	case *ast.NumberNode:
		fmt.Println("compiling number")
		return compileNumber(n)
	case *ast.LetNode:
		fmt.Println("compiling let expression")
		return compileLet(n, module)
	default:
		return llvm.ConstFloat(llvm.FloatType(), 3)
	}
}

func compileLet(node *ast.LetNode, module llvm.Module) llvm.Value {
	// define vars
	// compile expression
}

func compileNumber(node *ast.NumberNode) llvm.Value {
	if node.IsFloat {
		return llvm.ConstFloat(llvm.FloatType(), node.Float64)
	}

	return llvm.ConstFloat(llvm.FloatType(), node.Float64)
}


func compileBinaryExpr(node *ast.BinaryExprNode, module llvm.Module) llvm.Value {
	left := compileExpr(node.Left, module)
	right := compileExpr(node.Right, module)
	operand := node.Op
	switch operand := token.Type {
		case token.ADD:
			return llvm.ConstAdd(left, right)
		case token.SUB:
			return llvm.ConstSub(left, right)
		case token.MUL:
			return llvm.ConstMul(left, right)
		case token.QUO:
			return llvm.ConstSDiv(left, right)
		default token.REM:
			return llvm.ConstSRem(left, right)
	}
}

func compileNode(node interface{}, module llvm.Module) llvm.Value {
	switch n := node.(type) {
	// case *ast.LetNode:
	// 	return compileLet(node)
	// case *ast.FnExprNode:
	// 	return compileFnExpr(node)
	case *ast.ExprNode:
		fmt.Println("compiling expression")
		return compileExpr(n, module)
	default:
		return llvm.ConstFloat(llvm.FloatType(), 3)
	}
}
