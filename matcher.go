package main

import (
	"strings"
)

// LineType represents the type of content in a line
type LineType int

const (
	LineTypeEnglish LineType = iota
	LineTypeChinese
	LineTypeMixed
	LineTypeEmpty
)

// MatchDecision contains all decisions about how to handle a line
type MatchDecision struct {
	ShouldAddEmptyLine    bool
	ShouldFollowPrevStyle bool
	ShouldAdvanceSource   bool
	LineType              LineType
	LinesMatch            bool
}

// containsChinese checks if text contains Chinese characters
func containsChinese(text string) bool {
	for _, r := range text {
		// Check for CJK Unified Ideographs (most common Chinese characters)
		// Unicode range U+4E00–U+9FFF covers most Chinese characters
		if r >= '\u4e00' && r <= '\u9fff' {
			return true
		}
	}
	return false
}

// startsWithChinese checks if the first non-whitespace character in text is Chinese
func startsWithChinese(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return false
	}

	// Get the first rune (character)
	for _, r := range text {
		// Check for CJK Unified Ideographs (most common Chinese characters)
		// Unicode range U+4E00–U+9FFF covers most Chinese characters
		if r >= '\u4e00' && r <= '\u9fff' {
			return true
		}
		// If we encounter a non-whitespace character that's not Chinese, return false
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return false
}

// generateLineKey creates a normalized key from line text by removing all spaces
// and ignoring leading non-English characters, only including letters starting from first English letter
func generateLineKey(text string) string {
	// Find the first English letter
	firstEnglishIndex := -1
	for i, r := range text {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			firstEnglishIndex = i
			break
		}
	}

	// If no English letters found, return empty string
	if firstEnglishIndex == -1 {
		return ""
	}

	// Extract text starting from first English letter
	englishPortion := text[firstEnglishIndex:]

	// Remove all whitespace but keep all other characters (letters, numbers, punctuation)
	var result strings.Builder
	for _, r := range englishPortion {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			result.WriteRune(r)
		}
	}

	key := strings.ToLower(result.String())

	// Remove single character + dot patterns from the beginning (e.g., "a.", "1.", "b.", "2.")
	if len(key) >= 2 && key[1] == '.' {
		key = key[2:]
	}

	return key
}

// classifyLineType determines the type of content in a line
func classifyLineType(text string) LineType {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 {
		return LineTypeEmpty
	}

	hasChinese := containsChinese(text)
	hasEnglish := generateLineKey(text) != ""

	if hasChinese && hasEnglish {
		return LineTypeMixed
	} else if hasChinese {
		return LineTypeChinese
	} else if hasEnglish {
		return LineTypeEnglish
	}

	return LineTypeEmpty
}

// shouldAddEmptyLine determines if an empty line should be added after the current line
func shouldAddEmptyLine(lineText string, previousLineText string) bool {
	// Add empty line after Chinese and mixed content lines that don't already have spacing
	lineType := classifyLineType(lineText)
	if lineType == LineTypeChinese || lineType == LineTypeMixed {
		return true
	}
	return false
}

// shouldFollowPreviousLineStyle determines if the current line should follow the previous line's formatting
func shouldFollowPreviousLineStyle(lineText string, previousFeatures *LineFeatures) bool {
	// Both Chinese and mixed content lines should follow the formatting of the previous English line
	lineType := classifyLineType(lineText)
	if (lineType == LineTypeChinese || lineType == LineTypeMixed) && previousFeatures != nil {
		return true
	}
	return false
}

// shouldAdvanceSourceCursor determines if the source cursor should advance to the next line
func shouldAdvanceSourceCursor(lastProcessedWasChinese bool) bool {
	// Advance source cursor only if the last processed line was Chinese
	// This is because Chinese lines don't have corresponding source lines
	return lastProcessedWasChinese
}

// linesMatch checks if two line keys represent matching content
func linesMatch(sourceKey, targetKey string) bool {
	return sourceKey == targetKey
}

// AnalyzeLineMatch performs comprehensive analysis of line matching and formatting decisions
func AnalyzeLineMatch(sourceText, targetText string, previousFeatures *LineFeatures, lastProcessedWasChinese bool) *MatchDecision {
	targetLineType := classifyLineType(targetText)
	sourceKey := generateLineKey(sourceText)
	targetKey := generateLineKey(targetText)

	decision := &MatchDecision{
		LineType:              targetLineType,
		ShouldAddEmptyLine:    shouldAddEmptyLine(targetText, ""), // Previous line text not needed for current logic
		ShouldFollowPrevStyle: shouldFollowPreviousLineStyle(targetText, previousFeatures),
		ShouldAdvanceSource:   shouldAdvanceSourceCursor(lastProcessedWasChinese),
		LinesMatch:            linesMatch(sourceKey, targetKey),
	}

	return decision
}

// GetLineMatchingStrategy returns a strategy for handling the current line based on its type
func GetLineMatchingStrategy(targetText string) string {
	lineType := classifyLineType(targetText)

	switch lineType {
	case LineTypeChinese:
		return "chinese_translation"
	case LineTypeEnglish:
		return "english_match"
	case LineTypeMixed:
		return "mixed_content"
	case LineTypeEmpty:
		return "empty_line"
	default:
		return "unknown"
	}
}
