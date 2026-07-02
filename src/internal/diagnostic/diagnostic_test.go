package diagnostic

import (
	"encoding/json"
	"testing"
)

func TestOKResultSerializesDocumentedJSONShape(t *testing.T) {
	result := OK()

	if !result.OK {
		t.Fatal("expected OK result")
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(result.Errors))
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(result.Warnings))
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("expected result to marshal, got %v", err)
	}

	expected := `{"ok":true,"errors":[],"warnings":[]}`
	if string(data) != expected {
		t.Fatalf("expected JSON %s, got %s", expected, data)
	}
}

func TestErrorResultSerializesDiagnosticFields(t *testing.T) {
	span := Span{
		StartLine:   7,
		StartColumn: 22,
		EndLine:     7,
		EndColumn:   28,
		StartOffset: 128,
		EndOffset:   134,
	}

	result := FromDiagnostics(Diagnostic{
		Code:     CodeUndefinedName,
		Phase:    PhaseSemantics,
		Severity: SeverityError,
		File:     "src/shop.puff",
		Message:  "Undefined name: player",
		Hint:     "The name \"player\" is only available inside events that inject a player.",
		Span:     span,
	})

	if result.OK {
		t.Fatal("expected result with errors to be not OK")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %d", len(result.Errors))
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(result.Warnings))
	}

	got := result.Errors[0]
	if got.Code != CodeUndefinedName {
		t.Fatalf("expected code %q, got %q", CodeUndefinedName, got.Code)
	}
	if got.Phase != PhaseSemantics {
		t.Fatalf("expected phase %q, got %q", PhaseSemantics, got.Phase)
	}
	if got.Severity != SeverityError {
		t.Fatalf("expected severity %q, got %q", SeverityError, got.Severity)
	}
	if got.File != "src/shop.puff" {
		t.Fatalf("expected file %q, got %q", "src/shop.puff", got.File)
	}
	if got.Message != "Undefined name: player" {
		t.Fatalf("expected message %q, got %q", "Undefined name: player", got.Message)
	}
	if got.Hint != "The name \"player\" is only available inside events that inject a player." {
		t.Fatalf("expected hint %q, got %q", "The name \"player\" is only available inside events that inject a player.", got.Hint)
	}
	if got.Span != span {
		t.Fatalf("expected span %+v, got %+v", span, got.Span)
	}

	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("expected diagnostic to marshal, got %v", err)
	}

	expected := `{"code":"UNDEFINED_NAME","phase":"SEMANTICS","severity":"ERROR","message":"Undefined name: player","hint":"The name \"player\" is only available inside events that inject a player.","file":"src/shop.puff","span":{"startLine":7,"startColumn":22,"endLine":7,"endColumn":28,"startOffset":128,"endOffset":134}}`
	if string(data) != expected {
		t.Fatalf("expected diagnostic JSON %s, got %s", expected, data)
	}
}

func TestWarningResultSeparatesWarningsFromErrors(t *testing.T) {
	warning := Diagnostic{
		Code:     Code("UNUSED_VARIABLE"),
		Phase:    PhaseSemantics,
		Severity: SeverityWarning,
		Message:  "Variable is never used: $_price",
		Span: Span{
			StartLine:   8,
			StartColumn: 4,
			EndLine:     8,
			EndColumn:   11,
			StartOffset: 64,
			EndOffset:   71,
		},
	}

	result := FromDiagnostics(warning)

	if !result.OK {
		t.Fatal("expected warning-only result to be OK")
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(result.Errors))
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity, got %q", result.Warnings[0].Severity)
	}
}

func TestInfoResultDoesNotEnterErrorsOrWarnings(t *testing.T) {
	info := Diagnostic{
		Code:     Code("BUILD_OUTPUT_READY"),
		Phase:    PhaseCodegen,
		Severity: SeverityInfo,
		Message:  "Datapack output is ready.",
		Span: Span{
			StartLine:   1,
			StartColumn: 1,
			EndLine:     1,
			EndColumn:   1,
			StartOffset: 0,
			EndOffset:   0,
		},
	}

	result := FromDiagnostics(info)

	if !result.OK {
		t.Fatal("expected info-only result to be OK")
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(result.Errors))
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(result.Warnings))
	}
}

func TestOptionalFieldsAreOmittedWhenEmpty(t *testing.T) {
	result := FromDiagnostics(Diagnostic{
		Code:     CodeInvalidCharacter,
		Phase:    PhaseLexer,
		Severity: SeverityError,
		Message:  "Invalid character: @",
		Span: Span{
			StartLine:   1,
			StartColumn: 14,
			EndLine:     1,
			EndColumn:   15,
			StartOffset: 13,
			EndOffset:   14,
		},
	})

	data, err := json.Marshal(result.Errors[0])
	if err != nil {
		t.Fatalf("expected diagnostic to marshal, got %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("expected diagnostic JSON to unmarshal, got %v", err)
	}

	for _, field := range []string{"hint", "file", "notes"} {
		if _, ok := fields[field]; ok {
			t.Fatalf("expected empty optional field %q to be omitted from %s", field, data)
		}
	}
}

func TestDiagnosticNotesSerializeMessageFileAndOptionalSpan(t *testing.T) {
	noteSpan := Span{
		StartLine:   3,
		StartColumn: 5,
		EndLine:     3,
		EndColumn:   10,
		StartOffset: 24,
		EndOffset:   29,
	}
	diagnostic := Diagnostic{
		Code:     Code("DUPLICATE_SYMBOL"),
		Phase:    PhaseSemantics,
		Severity: SeverityError,
		Message:  "Function already declared: setup",
		Span: Span{
			StartLine:   12,
			StartColumn: 5,
			EndLine:     12,
			EndColumn:   10,
			StartOffset: 96,
			EndOffset:   101,
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

	data, err := json.Marshal(diagnostic)
	if err != nil {
		t.Fatalf("expected diagnostic to marshal, got %v", err)
	}

	var got Diagnostic
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("expected diagnostic JSON to unmarshal, got %v", err)
	}

	if len(got.Notes) != 2 {
		t.Fatalf("expected two notes, got %d", len(got.Notes))
	}
	if got.Notes[0].Message != "previous declaration here" {
		t.Fatalf("expected first note message, got %q", got.Notes[0].Message)
	}
	if got.Notes[0].File != "src/main.puff" {
		t.Fatalf("expected first note file, got %q", got.Notes[0].File)
	}
	if got.Notes[0].Span == nil || *got.Notes[0].Span != noteSpan {
		t.Fatalf("expected first note span %+v, got %+v", noteSpan, got.Notes[0].Span)
	}
	if got.Notes[1].Span != nil {
		t.Fatalf("expected second note span to be nil, got %+v", got.Notes[1].Span)
	}
}

func TestInitialRequiredCodesAreDefined(t *testing.T) {
	required := []Code{
		CodeInvalidUTF8,
		CodeInvalidLineEnding,
		CodeInvalidCharacter,
		CodeInvalidNumber,
		CodeUnterminatedString,
		CodeInvalidEscapeSequence,
		CodeUnterminatedInterpolation,
		CodeUnescapedCloseBrace,
		CodeEmptyInterpolation,
		CodeUnknownMetadataKey,
		CodeDuplicateMetadataKey,
		CodeInvalidMetadataValue,
		CodeSyntaxError,
		CodeExpectedToken,
		CodeUnexpectedToken,
		CodeExpectedNewline,
		CodeExpectedEnd,
		CodeInvalidTopLevelStatement,
		CodeInvalidImport,
		CodeInvalidImportPrefix,
		CodeAmbiguousImport,
		CodeImportNotFound,
		CodeInvalidDependency,
		CodeUndefinedName,
		CodeUndefinedVariable,
		CodeUndefinedFunction,
		CodeUndefinedType,
		CodeTypeMismatch,
		CodeMissingArguments,
		CodeTooManyArguments,
		CodeInvalidArgumentType,
		CodeInvalidReturnOutsideFunction,
		CodeMissingReturnValue,
		CodeMissingReturn,
		CodeInvalidStopInReturningFunc,
		CodeInvalidPublicLocalVariable,
		CodeAssignToImportedPublicVar,
		CodeMissingLoadEvent,
		CodeMissingTickEvent,
		CodeUnknownEffectPattern,
		CodeUnknownExpressionPattern,
		CodeUnknownConditionPattern,
		CodeUnknownEventPattern,
		CodeAmbiguousPattern,
		CodeInvalidPatternPlaceholder,
		CodePatternTypeMismatch,
		CodeMissingPuffTOML,
		CodeInvalidConfig,
		CodeInvalidPackID,
		CodeInvalidMinecraftVersion,
		CodeLockfileError,
		CodeInvalidNamespace,
		CodeInvalidMinecraftResource,
		CodeUnsupportedMinecraftVersion,
		CodeCodegenError,
		CodeLibraryError,
		CodeValidatorError,
		CodeUnknownCommand,
		CodeInvalidArgument,
		CodeCommandFailed,
	}

	seen := map[Code]bool{}
	for _, code := range required {
		if code == "" {
			t.Fatal("expected required diagnostic code to be non-empty")
		}
		if seen[code] {
			t.Fatalf("expected diagnostic code %q to be unique", code)
		}
		seen[code] = true
	}
}

func TestPhasesAndSeveritiesUseDocumentedValues(t *testing.T) {
	phases := []Phase{
		PhaseLexer,
		PhaseParser,
		PhaseSemantics,
		PhasePattern,
		PhaseIR,
		PhaseCodegen,
		PhaseProject,
		PhaseCLI,
		PhaseLibrary,
	}
	expectedPhases := []string{
		"LEXER",
		"PARSER",
		"SEMANTICS",
		"PATTERN",
		"IR",
		"CODEGEN",
		"PROJECT",
		"CLI",
		"LIBRARY",
	}

	for i, phase := range phases {
		if string(phase) != expectedPhases[i] {
			t.Fatalf("expected phase %q, got %q", expectedPhases[i], phase)
		}
	}

	severities := []Severity{
		SeverityError,
		SeverityWarning,
		SeverityInfo,
	}
	expectedSeverities := []string{
		"ERROR",
		"WARNING",
		"INFO",
	}

	for i, severity := range severities {
		if string(severity) != expectedSeverities[i] {
			t.Fatalf("expected severity %q, got %q", expectedSeverities[i], severity)
		}
	}
}
