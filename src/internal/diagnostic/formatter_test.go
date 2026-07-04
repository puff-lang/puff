package diagnostic

import "testing"

func TestFormatDiagnosticRendersSingleLineSpan(t *testing.T) {
	diagnostic := Diagnostic{
		Code:     CodeUndefinedName,
		Phase:    PhaseSemantics,
		Severity: SeverityError,
		File:     "src/main.puff",
		Message:  "Undefined name: player",
		Hint:     "The name \"player\" is only available inside events that inject a player.",
		Span: Span{
			StartLine:   4,
			StartColumn: 20,
			EndLine:     4,
			EndColumn:   26,
			StartOffset: 0,
			EndOffset:   0,
		},
	}
	source := "on load\n   send \"Hello\" to player\nend\n   send \"Hello\" to player\n"

	got := FormatDiagnostic(diagnostic, source)
	expected := "error[UNDEFINED_NAME]: Undefined name: player\n" +
		"  --> src/main.puff:4:20\n" +
		"   |\n" +
		" 4 |    send \"Hello\" to player\n" +
		"   |                    ^^^^^^\n" +
		"   |\n" +
		"   = hint: The name \"player\" is only available inside events that inject a player.\n"

	if got != expected {
		t.Fatalf("expected formatted diagnostic:\n%s\ngot:\n%s", expected, got)
	}
}

func TestFormatDiagnosticHandlesMissingFileAndSource(t *testing.T) {
	diagnostic := Diagnostic{
		Code:     CodeMissingPuffTOML,
		Phase:    PhaseProject,
		Severity: SeverityError,
		Message:  "Missing puff.toml.",
		Span: Span{
			StartLine:   1,
			StartColumn: 1,
			EndLine:     1,
			EndColumn:   1,
			StartOffset: 0,
			EndOffset:   0,
		},
	}

	got := FormatDiagnostic(diagnostic, "")
	expected := "error[MISSING_PUFF_TOML]: Missing puff.toml.\n" +
		"  --> <unknown>:1:1\n"

	if got != expected {
		t.Fatalf("expected formatted diagnostic:\n%s\ngot:\n%s", expected, got)
	}
}

func TestFormatDiagnosticRendersNotes(t *testing.T) {
	noteSpan := Span{
		StartLine:   3,
		StartColumn: 5,
		EndLine:     3,
		EndColumn:   10,
		StartOffset: 0,
		EndOffset:   0,
	}
	diagnostic := Diagnostic{
		Code:     Code("DUPLICATE_SYMBOL"),
		Phase:    PhaseSemantics,
		Severity: SeverityError,
		File:     "src/main.puff",
		Message:  "Function already declared: setup",
		Span: Span{
			StartLine:   12,
			StartColumn: 5,
			EndLine:     12,
			EndColumn:   10,
			StartOffset: 0,
			EndOffset:   0,
		},
		Notes: []Note{
			{
				Message: "previous declaration here",
				File:    "src/main.puff",
				Span:    &noteSpan,
			},
			{
				Message: "declaration must be unique in a module",
			},
		},
	}
	source := "line 1\nline 2\nfun setup\n"

	got := FormatDiagnostic(diagnostic, source)
	expected := "error[DUPLICATE_SYMBOL]: Function already declared: setup\n" +
		"  --> src/main.puff:12:5\n" +
		"   = note: previous declaration here\n" +
		"  --> src/main.puff:3:5\n" +
		"   |\n" +
		" 3 | fun setup\n" +
		"   |     ^^^^^\n" +
		"   = note: declaration must be unique in a module\n"

	if got != expected {
		t.Fatalf("expected formatted diagnostic:\n%s\ngot:\n%s", expected, got)
	}
}
