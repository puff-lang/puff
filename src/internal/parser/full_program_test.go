package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
)

func TestParseFullGrammarExampleGolden(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("testdata", "full_program.puff"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "full_program.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	result := parseTestSource("full_program.puff", string(input))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	wantText := strings.ReplaceAll(string(want), "\r\n", "\n")
	if got := renderFullProgram(result.File); got != wantText {
		t.Fatalf("unexpected full AST\nwant:\n%s\ngot:\n%s", wantText, got)
	}
}

func renderFullProgram(file *ast.File) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "requirements %d\n", len(file.Requirements))
	fmt.Fprintf(&builder, "declarations %d\n", len(file.Declarations))
	for _, declaration := range file.Declarations {
		switch node := declaration.(type) {
		case *ast.GlobalAssignment:
			if node.Public {
				builder.WriteString("pub ")
			}
			fmt.Fprintf(&builder, "global %s = %s\n", fullVariable(node.Target), fullExpression(node.Value))
		case *ast.FunctionDecl:
			if node.Public {
				builder.WriteString("pub ")
			}
			fmt.Fprintf(&builder, "fun %s(", node.Name.Name)
			for index, parameter := range node.Parameters {
				if index > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(parameter.Name.Name)
				if parameter.Type != nil {
					fmt.Fprintf(&builder, ": %s", renderType(parameter.Type))
				}
			}
			builder.WriteByte(')')
			if node.ReturnType != nil {
				fmt.Fprintf(&builder, " -> %s", renderType(node.ReturnType))
			}
			builder.WriteByte('\n')
			renderFullBlock(&builder, node.Body, "  ")
		case *ast.EventDecl:
			names := make([]string, len(node.Name))
			for index, name := range node.Name {
				names[index] = name.Name
			}
			fmt.Fprintf(&builder, "event %s\n", strings.Join(names, " "))
			renderFullBlock(&builder, node.Body, "  ")
		}
	}
	return builder.String()
}

func renderFullBlock(builder *strings.Builder, block ast.Block, indent string) {
	for _, statement := range block.Statements {
		switch node := statement.(type) {
		case *ast.AssignmentStmt:
			fmt.Fprintf(builder, "%sassign %s = %s\n", indent, fullVariable(node.Target), fullExpression(node.Value))
		case *ast.AddStmt:
			fmt.Fprintf(builder, "%sadd %s\n", indent, fullExpression(node.Value))
		case *ast.ReturnStmt:
			if node.Value == nil {
				fmt.Fprintf(builder, "%sreturn\n", indent)
			} else {
				fmt.Fprintf(builder, "%sreturn %s\n", indent, fullExpression(node.Value))
			}
		case *ast.StopStmt:
			fmt.Fprintf(builder, "%sstop\n", indent)
		case *ast.ExprStmt:
			fmt.Fprintf(builder, "%sexpr %s\n", indent, fullExpression(node.Expression))
		case *ast.EffectStmt:
			name := ""
			if len(node.Tokens) > 0 {
				name = node.Tokens[0].Lexeme
			}
			fmt.Fprintf(builder, "%seffect %s\n", indent, name)
		case *ast.IfStmt:
			fmt.Fprintf(builder, "%sif %s\n", indent, fullExpression(node.Condition))
			renderFullBlock(builder, node.Then, indent+"  ")
			for _, clause := range node.ElseIf {
				fmt.Fprintf(builder, "%selse if %s\n", indent, fullExpression(clause.Condition))
				renderFullBlock(builder, clause.Body, indent+"  ")
			}
			if node.Else != nil {
				fmt.Fprintf(builder, "%selse\n", indent)
				renderFullBlock(builder, *node.Else, indent+"  ")
			}
		case *ast.LoopTimesStmt:
			fmt.Fprintf(builder, "%sloop %s times\n", indent, fullExpression(node.Count))
			renderFullBlock(builder, node.Body, indent+"  ")
		case *ast.LoopRangeStmt:
			fmt.Fprintf(builder, "%sloop numbers from %s to %s\n", indent, fullExpression(node.Start), fullExpression(node.End))
			renderFullBlock(builder, node.Body, indent+"  ")
		case *ast.LoopPlayersStmt:
			fmt.Fprintf(builder, "%sloop players\n", indent)
			renderFullBlock(builder, node.Body, indent+"  ")
		case *ast.LoopEntitiesStmt:
			fmt.Fprintf(builder, "%sloop entities in radius %s around %s\n", indent, fullExpression(node.Radius), fullExpression(node.Around))
			renderFullBlock(builder, node.Body, indent+"  ")
		}
	}
}

func fullExpression(expression ast.Expression) string {
	switch node := expression.(type) {
	case *ast.NilLiteral:
		return "nil"
	case *ast.BoolLiteral:
		return strconv.FormatBool(node.Value)
	case *ast.IntLiteral:
		return strconv.FormatInt(node.Value, 10)
	case *ast.FloatLiteral:
		return strconv.FormatFloat(node.Value, 'g', -1, 64)
	case *ast.StringExpr:
		return strconv.Quote(stringValue(node))
	case *ast.VariableExpr:
		return fullVariable(node)
	case *ast.CallExpr:
		parts := make([]string, len(node.Callee.Parts))
		for index, part := range node.Callee.Parts {
			parts[index] = part.Name
		}
		name := strings.Join(parts, ".")
		if !node.ExplicitParens {
			return name
		}
		arguments := make([]string, len(node.Arguments))
		for index, argument := range node.Arguments {
			arguments[index] = fullExpression(argument)
		}
		return name + "(" + strings.Join(arguments, ", ") + ")"
	case *ast.UnaryExpr:
		return "(" + operatorShape(node.Operator) + " " + fullExpression(node.Operand) + ")"
	case *ast.BinaryExpr:
		return "(" + operatorShape(node.Operator) + " " + fullExpression(node.Left) + " " + fullExpression(node.Right) + ")"
	case *ast.GroupExpr:
		return "(group " + fullExpression(node.Expression) + ")"
	case *ast.RangeExpr:
		return "(range " + fullExpression(node.Start) + " " + fullExpression(node.End) + ")"
	case *ast.PatternExpr:
		parts := make([]string, len(node.Tokens))
		for index, tok := range node.Tokens {
			parts[index] = tok.Lexeme
		}
		return "pattern(" + strings.Join(parts, " ") + ")"
	default:
		return fmt.Sprintf("%T", expression)
	}
}

func fullVariable(variable *ast.VariableExpr) string {
	var builder strings.Builder
	if variable.Qualifier != nil {
		builder.WriteString(variable.Qualifier.Name)
		builder.WriteByte('.')
	}
	builder.WriteByte('$')
	if variable.Local {
		builder.WriteByte('_')
	}
	builder.WriteString(variable.Name.Name)
	for _, access := range variable.Accesses {
		switch node := access.(type) {
		case *ast.FieldAccess:
			builder.WriteByte('.')
			builder.WriteString(node.Field.Name)
		case *ast.EmptyIndexAccess:
			builder.WriteString("[]")
		case *ast.IndexAccess:
			builder.WriteByte('[')
			builder.WriteString(fullExpression(node.Index))
			builder.WriteByte(']')
		}
	}
	return builder.String()
}
