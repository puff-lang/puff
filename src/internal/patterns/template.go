package patterns

import (
	"fmt"
	"strings"

	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

type templatePart struct {
	literal     string
	placeholder string
}

func compileTemplate(syntax string) ([]templatePart, error) {
	words := strings.Fields(syntax)
	if len(words) == 0 {
		return nil, fmt.Errorf("pattern syntax cannot be empty")
	}

	parts := make([]templatePart, 0, len(words))
	placeholders := make(map[string]struct{})
	for _, word := range words {
		if !strings.Contains(word, "%") {
			parts = append(parts, templatePart{literal: word})
			continue
		}

		if len(word) < 3 || word[0] != '%' || word[len(word)-1] != '%' || strings.Count(word, "%") != 2 {
			return nil, fmt.Errorf("invalid placeholder %q", word)
		}
		name := word[1 : len(word)-1]
		if _, exists := placeholders[name]; exists {
			return nil, fmt.Errorf("duplicate placeholder %q", name)
		}
		if len(parts) > 0 && parts[len(parts)-1].placeholder != "" {
			return nil, fmt.Errorf("adjacent placeholders %q and %q require a literal delimiter", parts[len(parts)-1].placeholder, name)
		}
		placeholders[name] = struct{}{}
		parts = append(parts, templatePart{placeholder: name})
	}
	return parts, nil
}

func matchTemplate(parts []templatePart, tokens []token.Token) []Captures {
	matches := make([]Captures, 0, 2)
	matchFrom(parts, tokens, structuralTokens(tokens), 0, 0, Captures{}, &matches)
	return matches
}

func matchFrom(parts []templatePart, tokens []token.Token, structural []bool, partIndex, tokenIndex int, captures Captures, matches *[]Captures) {
	if len(*matches) == 2 {
		return
	}
	if partIndex == len(parts) {
		if tokenIndex == len(tokens) {
			*matches = append(*matches, captures)
		}
		return
	}
	if tokenIndex == len(tokens) {
		return
	}

	part := parts[partIndex]
	if part.placeholder == "" {
		if !structural[tokenIndex] || tokens[tokenIndex].Lexeme != part.literal {
			return
		}
		matchFrom(parts, tokens, structural, partIndex+1, tokenIndex+1, captures, matches)
		return
	}

	remainingParts := len(parts) - partIndex - 1
	lastCaptureEnd := len(tokens) - remainingParts
	for captureEnd := tokenIndex + 1; captureEnd <= lastCaptureEnd; captureEnd++ {
		nextCaptures := cloneCaptures(captures)
		captured := append([]token.Token(nil), tokens[tokenIndex:captureEnd]...)
		nextCaptures[part.placeholder] = Capture{
			Tokens: captured,
			Span:   captureSpan(captured),
		}
		matchFrom(parts, tokens, structural, partIndex+1, captureEnd, nextCaptures, matches)
	}
}

func cloneCaptures(captures Captures) Captures {
	cloned := make(Captures, len(captures)+1)
	for name, captured := range captures {
		cloned[name] = captured
	}
	return cloned
}

func structuralTokens(tokens []token.Token) []bool {
	structural := make([]bool, len(tokens))
	stringDepth := 0
	groupDepth := 0
	for index, current := range tokens {
		structural[index] = stringDepth == 0 && groupDepth == 0

		switch current.Type {
		case token.StringStart:
			stringDepth++
		case token.StringEnd:
			if stringDepth > 0 {
				stringDepth--
			}
		case token.LParen, token.LBracket, token.LBrace:
			if stringDepth == 0 {
				groupDepth++
			}
		case token.RParen, token.RBracket, token.RBrace:
			if stringDepth == 0 && groupDepth > 0 {
				groupDepth--
			}
		}
	}
	return structural
}

func captureSpan(tokens []token.Token) diagnostic.Span {
	first := tokens[0]
	last := tokens[len(tokens)-1]
	endLine, endColumn := tokenEnd(last)
	return diagnostic.Span{
		StartLine:   first.Line,
		StartColumn: first.Column,
		EndLine:     endLine,
		EndColumn:   endColumn,
		StartOffset: first.StartOffset,
		EndOffset:   last.EndOffset,
	}
}

func tokenEnd(current token.Token) (int, int) {
	line := current.Line
	column := current.Column + current.EndOffset - current.StartOffset
	if lastNewline := strings.LastIndexByte(current.Lexeme, '\n'); lastNewline >= 0 {
		line += strings.Count(current.Lexeme, "\n")
		column = len(current.Lexeme) - lastNewline
	}
	return line, column
}
