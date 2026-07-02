package diagnostic

type Code string

const (
	CodeInvalidUTF8                  Code = "INVALID_UTF8"
	CodeInvalidLineEnding            Code = "INVALID_LINE_ENDING"
	CodeInvalidCharacter             Code = "INVALID_CHARACTER"
	CodeInvalidNumber                Code = "INVALID_NUMBER"
	CodeUnterminatedString           Code = "UNTERMINATED_STRING"
	CodeInvalidEscapeSequence        Code = "INVALID_ESCAPE_SEQUENCE"
	CodeUnterminatedInterpolation    Code = "UNTERMINATED_INTERPOLATION"
	CodeUnescapedCloseBrace          Code = "UNESCAPED_CLOSE_BRACE"
	CodeEmptyInterpolation           Code = "EMPTY_INTERPOLATION"
	CodeUnknownMetadataKey           Code = "UNKNOWN_METADATA_KEY"
	CodeDuplicateMetadataKey         Code = "DUPLICATE_METADATA_KEY"
	CodeInvalidMetadataValue         Code = "INVALID_METADATA_VALUE"
	CodeSyntaxError                  Code = "SYNTAX_ERROR"
	CodeExpectedToken                Code = "EXPECTED_TOKEN"
	CodeUnexpectedToken              Code = "UNEXPECTED_TOKEN"
	CodeExpectedNewline              Code = "EXPECTED_NEWLINE"
	CodeExpectedEnd                  Code = "EXPECTED_END"
	CodeInvalidTopLevelStatement     Code = "INVALID_TOP_LEVEL_STATEMENT"
	CodeInvalidImport                Code = "INVALID_IMPORT"
	CodeInvalidImportPrefix          Code = "INVALID_IMPORT_PREFIX"
	CodeAmbiguousImport              Code = "AMBIGUOUS_IMPORT"
	CodeImportNotFound               Code = "IMPORT_NOT_FOUND"
	CodeInvalidDependency            Code = "INVALID_DEPENDENCY"
	CodeUndefinedName                Code = "UNDEFINED_NAME"
	CodeUndefinedVariable            Code = "UNDEFINED_VARIABLE"
	CodeUndefinedFunction            Code = "UNDEFINED_FUNCTION"
	CodeUndefinedType                Code = "UNDEFINED_TYPE"
	CodeTypeMismatch                 Code = "TYPE_MISMATCH"
	CodeMissingArguments             Code = "MISSING_ARGUMENTS"
	CodeTooManyArguments             Code = "TOO_MANY_ARGUMENTS"
	CodeInvalidArgumentType          Code = "INVALID_ARGUMENT_TYPE"
	CodeInvalidReturnOutsideFunction Code = "INVALID_RETURN_OUTSIDE_FUNCTION"
	CodeMissingReturnValue           Code = "MISSING_RETURN_VALUE"
	CodeMissingReturn                Code = "MISSING_RETURN"
	CodeInvalidStopInReturningFunc   Code = "INVALID_STOP_IN_RETURNING_FUNCTION"
	CodeInvalidPublicLocalVariable   Code = "INVALID_PUBLIC_LOCAL_VARIABLE"
	CodeAssignToImportedPublicVar    Code = "ASSIGN_TO_IMPORTED_PUBLIC_VARIABLE"
	CodeMissingLoadEvent             Code = "MISSING_LOAD_EVENT"
	CodeMissingTickEvent             Code = "MISSING_TICK_EVENT"
	CodeUnknownEffectPattern         Code = "UNKNOWN_EFFECT_PATTERN"
	CodeUnknownExpressionPattern     Code = "UNKNOWN_EXPRESSION_PATTERN"
	CodeUnknownConditionPattern      Code = "UNKNOWN_CONDITION_PATTERN"
	CodeUnknownEventPattern          Code = "UNKNOWN_EVENT_PATTERN"
	CodeAmbiguousPattern             Code = "AMBIGUOUS_PATTERN"
	CodeInvalidPatternPlaceholder    Code = "INVALID_PATTERN_PLACEHOLDER"
	CodePatternTypeMismatch          Code = "PATTERN_TYPE_MISMATCH"
	CodeMissingPuffTOML              Code = "MISSING_PUFF_TOML"
	CodeInvalidConfig                Code = "INVALID_CONFIG"
	CodeInvalidPackID                Code = "INVALID_PACK_ID"
	CodeInvalidMinecraftVersion      Code = "INVALID_MINECRAFT_VERSION"
	CodeLockfileError                Code = "LOCKFILE_ERROR"
	CodeInvalidNamespace             Code = "INVALID_NAMESPACE"
	CodeInvalidMinecraftResource     Code = "INVALID_MINECRAFT_RESOURCE"
	CodeUnsupportedMinecraftVersion  Code = "UNSUPPORTED_MINECRAFT_VERSION"
	CodeCodegenError                 Code = "CODEGEN_ERROR"
	CodeLibraryError                 Code = "LIBRARY_ERROR"
	CodeValidatorError               Code = "VALIDATOR_ERROR"
	CodeUnknownCommand               Code = "UNKNOWN_COMMAND"
	CodeInvalidArgument              Code = "INVALID_ARGUMENT"
	CodeCommandFailed                Code = "COMMAND_FAILED"
)

type Phase string

const (
	PhaseLexer     Phase = "LEXER"
	PhaseParser    Phase = "PARSER"
	PhaseSemantics Phase = "SEMANTICS"
	PhasePattern   Phase = "PATTERN"
	PhaseIR        Phase = "IR"
	PhaseCodegen   Phase = "CODEGEN"
	PhaseProject   Phase = "PROJECT"
	PhaseCLI       Phase = "CLI"
	PhaseLibrary   Phase = "LIBRARY"
)

type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
)

type Span struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
}

type Note struct {
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
	Span    *Span  `json:"span,omitempty"`
}

type Diagnostic struct {
	Code     Code     `json:"code"`
	Phase    Phase    `json:"phase"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Hint     string   `json:"hint,omitempty"`
	File     string   `json:"file,omitempty"`
	Span     Span     `json:"span"`
	Notes    []Note   `json:"notes,omitempty"`
}

type Result struct {
	OK       bool         `json:"ok"`
	Errors   []Diagnostic `json:"errors"`
	Warnings []Diagnostic `json:"warnings"`
}

func OK() Result {
	return Result{
		OK:       true,
		Errors:   []Diagnostic{},
		Warnings: []Diagnostic{},
	}
}

func FromDiagnostics(diagnostics ...Diagnostic) Result {
	result := Result{
		OK:       true,
		Errors:   []Diagnostic{},
		Warnings: []Diagnostic{},
	}

	for _, diagnostic := range diagnostics {
		switch diagnostic.Severity {
		case SeverityError:
			result.Errors = append(result.Errors, diagnostic)
			result.OK = false
		case SeverityWarning:
			result.Warnings = append(result.Warnings, diagnostic)
		}
	}

	return result
}
