package sema

import (
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/source"
)

func TestCheckRequiredEventsFromMetadata(t *testing.T) {
	tests := []struct {
		name         string
		tags         string
		events       [][]string
		wantCodes    []diagnostic.Code
		wantMessages []string
		wantHints    []string
	}{
		{
			name:   "load present",
			tags:   "load",
			events: [][]string{{"load"}},
		},
		{
			name:   "tick present",
			tags:   "tick",
			events: [][]string{{"tick"}},
		},
		{
			name:   "both present",
			tags:   "load, tick",
			events: [][]string{{"load"}, {"tick"}},
		},
		{
			name:   "custom tags have no required event",
			tags:   "custom:event, minecraft:load",
			events: nil,
		},
		{
			name:         "load missing",
			tags:         "load",
			events:       [][]string{{"tick"}},
			wantCodes:    []diagnostic.Code{diagnostic.CodeMissingLoadEvent},
			wantMessages: []string{"Missing required event: on load"},
			wantHints:    []string{"Add an on load block or remove the load tag."},
		},
		{
			name:         "tick missing",
			tags:         "tick",
			events:       [][]string{{"load"}},
			wantCodes:    []diagnostic.Code{diagnostic.CodeMissingTickEvent},
			wantMessages: []string{"Missing required event: on tick"},
			wantHints:    []string{"Add an on tick block or remove the tick tag."},
		},
		{
			name:      "both missing",
			tags:      "load, tick",
			events:    nil,
			wantCodes: []diagnostic.Code{diagnostic.CodeMissingLoadEvent, diagnostic.CodeMissingTickEvent},
			wantMessages: []string{
				"Missing required event: on load",
				"Missing required event: on tick",
			},
			wantHints: []string{
				"Add an on load block or remove the load tag.",
				"Add an on tick block or remove the tick tag.",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata := efMetadata(test.tags, 1)
			declarations := make([]ast.Declaration, 0, len(test.events))
			for index, name := range test.events {
				declarations = append(declarations, efEvent(name, nil, 10+index))
			}
			project := efProject(efModule("events.puff", []ast.MetadataEntry{metadata}, declarations...))

			result := Check(project)

			want := make([]diagnostic.Diagnostic, len(test.wantCodes))
			for index, code := range test.wantCodes {
				want[index] = efDiagnostic(
					code,
					test.wantMessages[index],
					test.wantHints[index],
					"events.puff",
					metadata.Span(),
				)
			}
			efAssertDiagnostics(t, result.Diagnostics, want...)
		})
	}
}

func TestCheckRequiredEventsMatchExactNamesPerModule(t *testing.T) {
	t.Run("event names are exact and case sensitive", func(t *testing.T) {
		metadata := efMetadata("load", 2)
		project := efProject(efModule(
			"exact.puff",
			[]ast.MetadataEntry{metadata},
			efEvent([]string{"Load"}, nil, 10),
			efEvent([]string{"load", "extra"}, nil, 11),
		))

		result := Check(project)

		efAssertDiagnostics(t, result.Diagnostics, efDiagnostic(
			diagnostic.CodeMissingLoadEvent,
			"Missing required event: on load",
			"Add an on load block or remove the load tag.",
			"exact.puff",
			metadata.Span(),
		))
	})

	t.Run("another module cannot satisfy the requirement", func(t *testing.T) {
		metadata := efMetadata("load", 3)
		project := efProject(
			efModule("needs-load.puff", []ast.MetadataEntry{metadata}),
			efModule("has-load.puff", nil, efEvent([]string{"load"}, nil, 20)),
		)

		result := Check(project)

		efAssertDiagnostics(t, result.Diagnostics, efDiagnostic(
			diagnostic.CodeMissingLoadEvent,
			"Missing required event: on load",
			"Add an on load block or remove the load tag.",
			"needs-load.puff",
			metadata.Span(),
		))
	})
}

func TestCheckRejectsReturnInEventsIncludingNestedLoops(t *testing.T) {
	tests := []struct {
		name string
		body []ast.Statement
		span diagnostic.Span
	}{
		{
			name: "direct",
			body: []ast.Statement{efReturn(nil, 30)},
			span: efSpan(30),
		},
		{
			name: "inside loop",
			body: []ast.Statement{
				&ast.LoopTimesStmt{
					NodeBase: ast.NodeBase{SourceSpan: efSpan(31)},
					Count:    efInt(1, 32),
					Body:     efBlock(efReturn(nil, 33)),
				},
			},
			span: efSpan(33),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project := efProject(efModule(
				"event-return.puff",
				nil,
				efEvent([]string{"custom"}, test.body, 29),
			))

			result := Check(project)

			efAssertDiagnostics(t, result.Diagnostics, efDiagnostic(
				diagnostic.CodeInvalidReturnOutsideFunction,
				"return can only be used inside functions.",
				"Use stop to stop an event or execution block.",
				"event-return.puff",
				test.span,
			))
		})
	}
}

func TestCheckReturningFunctionFlow(t *testing.T) {
	intType := efType("int", 40)
	tests := []struct {
		name string
		body []ast.Statement
		want *diagnostic.Diagnostic
	}{
		{
			name: "bare return reports missing value without missing return cascade",
			body: []ast.Statement{efReturn(nil, 41)},
			want: efDiagnosticPointer(
				diagnostic.CodeMissingReturnValue,
				"Missing return value.",
				"Return a value compatible with int.",
				"flow.puff",
				efSpan(41),
			),
		},
		{
			name: "absent return",
			body: nil,
			want: efDiagnosticPointer(
				diagnostic.CodeMissingReturn,
				"Function calculate must return int in all paths.",
				"Add an else branch or a final return.",
				"flow.puff",
				efSpan(40),
			),
		},
		{
			name: "partial if",
			body: []ast.Statement{
				efIf(
					efBlock(efReturn(efInt(1, 43), 42)),
					nil,
					nil,
					42,
				),
			},
			want: efDiagnosticPointer(
				diagnostic.CodeMissingReturn,
				"Function calculate must return int in all paths.",
				"Add an else branch or a final return.",
				"flow.puff",
				efSpan(40),
			),
		},
		{
			name: "if else returns on all paths",
			body: []ast.Statement{
				efIf(
					efBlock(efReturn(efInt(1, 45), 44)),
					nil,
					efBlockPointer(efReturn(efInt(2, 47), 46)),
					44,
				),
			},
		},
		{
			name: "else if and else return on all paths",
			body: []ast.Statement{
				efIf(
					efBlock(efReturn(efInt(1, 49), 48)),
					[]ast.ElseIfClause{
						{
							NodeBase:  ast.NodeBase{SourceSpan: efSpan(50)},
							Condition: efBool(true, 50),
							Body:      efBlock(efReturn(efInt(2, 52), 51)),
						},
					},
					efBlockPointer(efReturn(efInt(3, 54), 53)),
					48,
				),
			},
		},
		{
			name: "nested if returns on all paths",
			body: []ast.Statement{
				efIf(
					efBlock(efIf(
						efBlock(efReturn(efInt(1, 57), 56)),
						nil,
						efBlockPointer(efReturn(efInt(2, 59), 58)),
						56,
					)),
					nil,
					efBlockPointer(efReturn(efInt(3, 61), 60)),
					55,
				),
			},
		},
		{
			name: "partial if followed by final return",
			body: []ast.Statement{
				efIf(
					efBlock(efReturn(efInt(1, 63), 62)),
					nil,
					nil,
					62,
				),
				efReturn(efInt(2, 65), 64),
			},
		},
		{
			name: "return only inside loop does not guarantee return",
			body: []ast.Statement{
				&ast.LoopTimesStmt{
					NodeBase: ast.NodeBase{SourceSpan: efSpan(66)},
					Count:    efInt(1, 66),
					Body:     efBlock(efReturn(efInt(1, 68), 67)),
				},
			},
			want: efDiagnosticPointer(
				diagnostic.CodeMissingReturn,
				"Function calculate must return int in all paths.",
				"Add an else branch or a final return.",
				"flow.puff",
				efSpan(40),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			function := efFunction("calculate", intType, test.body, 40)
			project := efProject(efModule("flow.puff", nil, function))

			result := Check(project)

			if test.want == nil {
				efAssertDiagnostics(t, result.Diagnostics)
				return
			}
			efAssertDiagnostics(t, result.Diagnostics, *test.want)
		})
	}
}

func TestCheckStopRules(t *testing.T) {
	t.Run("stop is valid in event", func(t *testing.T) {
		project := efProject(efModule(
			"stop-event.puff",
			nil,
			efEvent([]string{"custom"}, []ast.Statement{efStop(70)}, 69),
		))

		result := Check(project)

		efAssertDiagnostics(t, result.Diagnostics)
	})

	t.Run("stop is valid in untyped function", func(t *testing.T) {
		project := efProject(efModule(
			"stop-function.puff",
			nil,
			efFunction("setup", nil, []ast.Statement{efStop(72)}, 71),
		))

		result := Check(project)

		efAssertDiagnostics(t, result.Diagnostics)
	})

	t.Run("stop in typed function has no missing return cascade", func(t *testing.T) {
		project := efProject(efModule(
			"stop-returning.puff",
			nil,
			efFunction("calculate", efType("int", 73), []ast.Statement{efStop(74)}, 73),
		))

		result := Check(project)

		efAssertDiagnostics(t, result.Diagnostics, efDiagnostic(
			diagnostic.CodeInvalidStopInReturningFunc,
			"stop cannot replace a return value.",
			"Return a value compatible with int.",
			"stop-returning.puff",
			efSpan(74),
		))
	})
}

func TestCheckRejectsIncompatibleReturnType(t *testing.T) {
	value := &ast.StringExpr{
		NodeBase: ast.NodeBase{SourceSpan: efSpan(81)},
		Quote:    '"',
		Parts: []ast.StringPart{
			&ast.StringText{
				NodeBase: ast.NodeBase{SourceSpan: efSpan(81)},
				Raw:      "wrong",
				Value:    "wrong",
			},
		},
	}
	project := efProject(efModule(
		"return-type.puff",
		nil,
		efFunction(
			"calculate",
			efType("int", 80),
			[]ast.Statement{efReturn(value, 80)},
			80,
		),
	))

	result := Check(project)

	efAssertDiagnostics(t, result.Diagnostics, efDiagnostic(
		diagnostic.CodeTypeMismatch,
		"Type mismatch: cannot return string as int.",
		"Return a value compatible with int.",
		"return-type.puff",
		value.Span(),
	))
}

func efProject(modules ...*Module) *Project {
	return &Project{Root: "/project", Modules: modules}
}

func efModule(relPath string, metadata []ast.MetadataEntry, declarations ...ast.Declaration) *Module {
	return &Module{
		Source: source.NewFile("/project/src/"+relPath, relPath, ""),
		Syntax: &ast.File{
			Metadata:     metadata,
			Declarations: declarations,
		},
		Imports: map[string]*Import{},
	}
}

func efMetadata(tags string, seed int) ast.MetadataEntry {
	return ast.MetadataEntry{
		NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
		Key:      "tags",
		Value:    tags,
	}
}

func efEvent(name []string, statements []ast.Statement, seed int) *ast.EventDecl {
	identifiers := make([]ast.Identifier, len(name))
	for index, part := range name {
		identifiers[index] = ast.Identifier{
			NodeBase: ast.NodeBase{SourceSpan: efSpan(seed + index)},
			Name:     part,
		}
	}
	return &ast.EventDecl{
		NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
		Name:     identifiers,
		Body:     efBlock(statements...),
	}
}

func efFunction(name string, returnType *ast.TypeRef, statements []ast.Statement, seed int) *ast.FunctionDecl {
	return &ast.FunctionDecl{
		NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
		Name: ast.Identifier{
			NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
			Name:     name,
		},
		ReturnType: returnType,
		Body:       efBlock(statements...),
	}
}

func efType(name string, seed int) *ast.TypeRef {
	return &ast.TypeRef{
		NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
		Name: ast.Identifier{
			NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
			Name:     name,
		},
	}
}

func efBlock(statements ...ast.Statement) ast.Block {
	return ast.Block{Statements: statements}
}

func efBlockPointer(statements ...ast.Statement) *ast.Block {
	value := efBlock(statements...)
	return &value
}

func efIf(then ast.Block, elseIf []ast.ElseIfClause, elseBlock *ast.Block, seed int) *ast.IfStmt {
	return &ast.IfStmt{
		NodeBase:  ast.NodeBase{SourceSpan: efSpan(seed)},
		Condition: efBool(true, seed),
		Then:      then,
		ElseIf:    elseIf,
		Else:      elseBlock,
	}
}

func efReturn(value ast.Expression, seed int) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
		Value:    value,
	}
}

func efStop(seed int) *ast.StopStmt {
	return &ast.StopStmt{NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)}}
}

func efInt(value int64, seed int) *ast.IntLiteral {
	return &ast.IntLiteral{
		NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
		Value:    value,
	}
}

func efBool(value bool, seed int) *ast.BoolLiteral {
	return &ast.BoolLiteral{
		NodeBase: ast.NodeBase{SourceSpan: efSpan(seed)},
		Value:    value,
	}
}

func efSpan(seed int) diagnostic.Span {
	return diagnostic.Span{
		StartLine:   seed,
		StartColumn: 2,
		EndLine:     seed,
		EndColumn:   8,
		StartOffset: seed * 10,
		EndOffset:   seed*10 + 6,
	}
}

func efDiagnostic(
	code diagnostic.Code,
	message string,
	hint string,
	file string,
	span diagnostic.Span,
) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:     code,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  message,
		Hint:     hint,
		File:     file,
		Span:     span,
	}
}

func efDiagnosticPointer(
	code diagnostic.Code,
	message string,
	hint string,
	file string,
	span diagnostic.Span,
) *diagnostic.Diagnostic {
	value := efDiagnostic(code, message, hint, file, span)
	return &value
}

func efAssertDiagnostics(
	t *testing.T,
	got []diagnostic.Diagnostic,
	want ...diagnostic.Diagnostic,
) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected diagnostics:\ngot  %#v\nwant %#v", got, want)
	}
}
