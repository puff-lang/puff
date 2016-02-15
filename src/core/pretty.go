package core

func PpExpr(expr interface{}) string {
	switch e := expr.(type) {
	case ENum:
		return e.Text
	case EVar:
		return string(e)
	case EAp:
		return PpExpr(e.Left) + "(" + PpAExpr(e.Body) + ")"
	default:
		return ""
	}
}

func PpAExpr(expr Expr) string {
	if IsAtomicExpr(expr) {
		return PpExpr(expr)
	} else {
		return "(" + PpExpr(expr) + ")"
	}
}
