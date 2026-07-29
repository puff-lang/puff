package parser

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
)

func TestParseStatementsAndBranches(t *testing.T) {
	result := parseTestSource("statements.puff", `
fun example
$_price = 50
add 1 to $coins
if $coins >= 100
return true
else if $coins >= 50
return false
else
stop
end
shop.Run
send "Hello" to player
end
`)

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	body := result.File.Declarations[0].(*ast.FunctionDecl).Body
	if len(body.Statements) != 5 {
		t.Fatalf("expected five statements, got %#v", body.Statements)
	}
	assignment := body.Statements[0].(*ast.AssignmentStmt)
	if !assignment.Target.Local || assignment.Target.Name.Name != "price" {
		t.Fatalf("unexpected assignment: %#v", assignment)
	}
	if body.Statements[1].(*ast.AddStmt).Target.(*ast.VariableExpr).Name.Name != "coins" {
		t.Fatalf("unexpected add statement: %#v", body.Statements[1])
	}
	conditional := body.Statements[2].(*ast.IfStmt)
	if len(conditional.Then.Statements) != 1 || len(conditional.ElseIf) != 1 || len(conditional.Else.Statements) != 1 {
		t.Fatalf("unexpected conditional: %#v", conditional)
	}
	if _, ok := body.Statements[3].(*ast.ExprStmt); !ok {
		t.Fatalf("expected expression statement, got %T", body.Statements[3])
	}
	if _, ok := body.Statements[4].(*ast.EffectStmt); !ok {
		t.Fatalf("expected effect statement, got %T", body.Statements[4])
	}
}

func TestParseAllLoopForms(t *testing.T) {
	result := parseTestSource("loops.puff", `
on load
loop 3 times
stop
end
loop numbers from 10 to 1
stop
end
loop players
stop
end
loop entities in radius 10 around player
stop
end
end
`)

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	body := result.File.Declarations[0].(*ast.EventDecl).Body
	if len(body.Statements) != 4 {
		t.Fatalf("expected four loops, got %#v", body.Statements)
	}
	if body.Statements[0].(*ast.LoopTimesStmt).Count.(*ast.IntLiteral).Value != 3 {
		t.Fatalf("unexpected times loop: %#v", body.Statements[0])
	}
	rangeLoop := body.Statements[1].(*ast.LoopRangeStmt)
	if rangeLoop.Start.(*ast.IntLiteral).Value != 10 || rangeLoop.End.(*ast.IntLiteral).Value != 1 {
		t.Fatalf("unexpected range loop: %#v", rangeLoop)
	}
	if _, ok := body.Statements[2].(*ast.LoopPlayersStmt); !ok {
		t.Fatalf("expected players loop, got %T", body.Statements[2])
	}
	entities := body.Statements[3].(*ast.LoopEntitiesStmt)
	if entities.Radius.(*ast.IntLiteral).Value != 10 || expressionShape(entities.Around) != "player" {
		t.Fatalf("unexpected entities loop: %#v", entities)
	}
}

func TestParseAddPatternTargetAndCondition(t *testing.T) {
	result := parseTestSource("patterns.puff", `
on join
add amount to coins of target
if coins of player >= 100
stop
end
end
`)

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	body := result.File.Declarations[0].(*ast.EventDecl).Body
	add := body.Statements[0].(*ast.AddStmt)
	if _, ok := add.Value.(*ast.CallExpr); !ok {
		t.Fatalf("expected amount expression, got %T", add.Value)
	}
	if len(add.Target.(*ast.AccessExpr).Tokens) != 3 {
		t.Fatalf("unexpected access pattern: %#v", add.Target)
	}
	condition := body.Statements[1].(*ast.IfStmt).Condition.(*ast.BinaryExpr)
	if _, ok := condition.Left.(*ast.PatternExpr); !ok {
		t.Fatalf("expected pattern condition, got %T", condition.Left)
	}
}

func TestParseRejectsMissingLoopOperands(t *testing.T) {
	tests := []string{
		"on load\nloop times\nend\nend\n",
		"on load\nloop numbers from to 3\nend\nend\n",
		"on load\nloop entities in radius around player\nend\nend\n",
	}

	for _, input := range tests {
		result := parseTestSource("loop.puff", input)
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Message != `Expected "expression".` {
			t.Fatalf("source %q: expected missing expression diagnostic, got %#v", input, result.Diagnostics)
		}
	}
}
