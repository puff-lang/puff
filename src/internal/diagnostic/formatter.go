package diagnostic

import (
	"fmt"
	"strings"
)

func FormatDiagnostic(diagnostic Diagnostic, source string) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "%s[%s]: %s\n", formatSeverity(diagnostic.Severity), diagnostic.Code, diagnostic.Message)
	writeLocation(&builder, diagnostic.File, diagnostic.Span)
	writeSourceLine(&builder, diagnostic.Span, source)

	if diagnostic.Hint != "" {
		builder.WriteString("   |\n")
		fmt.Fprintf(&builder, "   = hint: %s\n", diagnostic.Hint)
	}

	for _, note := range diagnostic.Notes {
		fmt.Fprintf(&builder, "   = note: %s\n", note.Message)
		if note.Span != nil {
			writeLocation(&builder, note.File, *note.Span)
			writeSourceLine(&builder, *note.Span, source)
		}
	}

	return builder.String()
}

func formatSeverity(severity Severity) string {
	switch severity {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return strings.ToLower(string(severity))
	}
}

func writeLocation(builder *strings.Builder, file string, span Span) {
	if file == "" {
		file = "<unknown>"
	}

	fmt.Fprintf(builder, "  --> %s:%d:%d\n", file, span.StartLine, span.StartColumn)
}

func writeSourceLine(builder *strings.Builder, span Span, source string) {
	line, ok := sourceLine(source, span.StartLine)
	if !ok {
		return
	}

	lineNumber := fmt.Sprintf("%d", span.StartLine)
	fmt.Fprintf(builder, "   |\n")
	fmt.Fprintf(builder, "%s | %s\n", leftPad(lineNumber, 2), line)
	fmt.Fprintf(builder, "   | %s%s\n", strings.Repeat(" ", max(span.StartColumn-1, 0)), caretRange(span))
}

func sourceLine(source string, lineNumber int) (string, bool) {
	if source == "" || lineNumber < 1 {
		return "", false
	}

	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	if lineNumber > len(lines) {
		return "", false
	}

	return lines[lineNumber-1], true
}

func caretRange(span Span) string {
	width := span.EndColumn - span.StartColumn
	if width < 1 {
		width = 1
	}

	return strings.Repeat("^", width)
}

func leftPad(value string, width int) string {
	if len(value) >= width {
		return value
	}

	return strings.Repeat(" ", width-len(value)) + value
}
