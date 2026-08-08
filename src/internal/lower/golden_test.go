package lower

import (
	"testing"
)

func TestLowerTargetProgramUsesDistinctASTAndIRGoldens(t *testing.T) {
	input := readTestdata(t, "target.puff")
	wantAST := readTestdata(t, "target.ast.golden")
	wantIR := readTestdata(t, "target.ir.golden")

	syntax := parseAST(t, "main.puff", input)
	gotAST := renderAST("main.puff", syntax)
	if gotAST != wantAST {
		t.Fatalf("unexpected target AST\nwant:\n%s\ngot:\n%s", wantAST, gotAST)
	}

	result := Lower(checkedProject(t, map[string]string{"main.puff": input}))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower target: %#v", result.Diagnostics)
	}
	gotIR := renderIR(result.Project)
	if gotIR != wantIR {
		t.Fatalf("unexpected target IR\nwant:\n%s\ngot:\n%s", wantIR, gotIR)
	}
	if gotAST == gotIR || wantAST == wantIR {
		t.Fatal("AST and IR goldens must represent distinct compiler layers")
	}
}
