package lower

import (
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/ir"
)

func TestLowerPropagatesUnknownEffectDiagnosticWithoutInvalidCommand(t *testing.T) {
	const input = `on load
   explode player
end
`
	project := checkedProject(t, map[string]string{"main.puff": input})
	event := project.Modules[0].Syntax.Declarations[0].(*ast.EventDecl)
	effect := event.Body.Statements[0].(*ast.EffectStmt)

	result := Lower(project)
	want := diagnostic.Diagnostic{
		Code:     diagnostic.CodeUnknownEffectPattern,
		Phase:    diagnostic.PhasePattern,
		Severity: diagnostic.SeverityError,
		Message:  "Unknown effect pattern.",
		Hint:     "Check the syntax or require a library that registers this effect.",
		File:     "main.puff",
		Span:     effect.Span(),
	}
	assertOnlyDiagnostic(t, result.Diagnostics, want)
	if countEffects(result.Project) != 0 {
		t.Fatalf("unknown effect emitted a command: %#v", result.Project)
	}
	if result.Project != nil {
		for _, tag := range result.Project.Tags {
			if tag.Name != "load" {
				t.Errorf("unknown effect emitted invalid tag %#v", tag)
			}
		}
	}
}

func TestLowerPropagatesUnknownEventDiagnosticWithoutHandlerOrTag(t *testing.T) {
	const input = `on scoreboard update
end
`
	project := checkedProject(t, map[string]string{"main.puff": input})
	event := project.Modules[0].Syntax.Declarations[0].(*ast.EventDecl)
	nameSpan := ast.JoinSpans(event.Name[0].Span(), event.Name[len(event.Name)-1].Span())

	result := Lower(project)
	want := diagnostic.Diagnostic{
		Code:     diagnostic.CodeUnknownEventPattern,
		Phase:    diagnostic.PhasePattern,
		Severity: diagnostic.SeverityError,
		Message:  "Unknown event pattern: scoreboard update",
		Hint:     "Require the library that registers this event.",
		File:     "main.puff",
		Span:     nameSpan,
	}
	assertOnlyDiagnostic(t, result.Diagnostics, want)
	if result.Project != nil && (len(result.Project.Tags) != 0 || len(result.Project.Functions) != 0) {
		t.Fatalf("unknown event emitted handler or tag: %#v", result.Project)
	}
}

func TestLowerReportsUnsupportedStatementInsteadOfIgnoringIt(t *testing.T) {
	const input = `on load
   stop
end
`
	project := checkedProject(t, map[string]string{"main.puff": input})
	event := project.Modules[0].Syntax.Declarations[0].(*ast.EventDecl)
	stop := event.Body.Statements[0].(*ast.StopStmt)

	result := Lower(project)
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one unsupported-node diagnostic", result.Diagnostics)
	}
	issue := result.Diagnostics[0]
	if issue.Code != diagnostic.CodeUnsupportedASTNode {
		t.Errorf("Code = %q, want %q", issue.Code, diagnostic.CodeUnsupportedASTNode)
	}
	if issue.Phase != diagnostic.PhaseIR {
		t.Errorf("Phase = %q, want %q", issue.Phase, diagnostic.PhaseIR)
	}
	if issue.Severity != diagnostic.SeverityError {
		t.Errorf("Severity = %q, want %q", issue.Severity, diagnostic.SeverityError)
	}
	if issue.File != "main.puff" {
		t.Errorf("File = %q, want main.puff", issue.File)
	}
	if issue.Span != stop.Span() {
		t.Errorf("Span = %+v, want %+v", issue.Span, stop.Span())
	}
	if countEffects(result.Project) != 0 {
		t.Fatalf("unsupported statement emitted a command: %#v", result.Project)
	}
}

func assertOnlyDiagnostic(t *testing.T, got []diagnostic.Diagnostic, want diagnostic.Diagnostic) {
	t.Helper()

	if len(got) != 1 {
		t.Fatalf("diagnostics = %#v, want exactly %#v", got, want)
	}
	if !reflect.DeepEqual(got[0], want) {
		t.Errorf("diagnostic = %#v, want %#v", got[0], want)
	}
}

func countEffects(project *ir.Project) int {
	if project == nil {
		return 0
	}
	count := 0
	for _, function := range project.Functions {
		for _, command := range function.Commands {
			if _, ok := command.(*ir.Effect); ok {
				count++
			}
		}
	}
	return count
}
