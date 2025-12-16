package main

import (
	"testing"
)

func TestAnalyzeLineMatch(t *testing.T) {
	tests := []struct {
		name                    string
		sourceText              string
		targetText              string
		previousFeatures        *LineFeatures
		lastProcessedWasChinese bool
		expectedLineType        LineType
		expectedShouldAdvance   bool
		expectedFollowPrevStyle bool
		expectedLinesMatch      bool
		expectedAddEmptyLine    bool
	}{
		{
			name:                    "English to English match",
			sourceText:              "Heritage Baptist Church Morning Service",
			targetText:              "Heritage Baptist Church Morning Service",
			previousFeatures:        nil,
			lastProcessedWasChinese: false,
			expectedLineType:        LineTypeEnglish,
			expectedShouldAdvance:   false,
			expectedFollowPrevStyle: false,
			expectedLinesMatch:      true,
			expectedAddEmptyLine:    false,
		},
		{
			name:                    "English to Chinese translation",
			sourceText:              "Heritage Baptist Church Morning Service",
			targetText:              "遗产浸信会早晨崇拜",
			previousFeatures:        &LineFeatures{Text: "test"},
			lastProcessedWasChinese: false,
			expectedLineType:        LineTypeChinese,
			expectedShouldAdvance:   false,
			expectedFollowPrevStyle: true,
			expectedLinesMatch:      false,
			expectedAddEmptyLine:    true,
		},
		{
			name:                    "Mixed content line (Chinese with English)",
			sourceText:              "Pastor Alan Fong",
			targetText:              "牧师 Alan Fong",
			previousFeatures:        &LineFeatures{Text: "test"},
			lastProcessedWasChinese: false,
			expectedLineType:        LineTypeMixed,
			expectedShouldAdvance:   false,
			expectedFollowPrevStyle: true, // Mixed content should follow previous style
			expectedLinesMatch:      false,
			expectedAddEmptyLine:    true, // Mixed content should add empty line
		},
		{
			name:                    "Source advance after Chinese line",
			sourceText:              "Title: Jesus Christ, the Same",
			targetText:              "Title: Jesus Christ, the Same",
			previousFeatures:        nil,
			lastProcessedWasChinese: true,
			expectedLineType:        LineTypeEnglish,
			expectedShouldAdvance:   true,
			expectedFollowPrevStyle: false,
			expectedLinesMatch:      true,
			expectedAddEmptyLine:    false,
		},
		{
			name:                    "Empty line",
			sourceText:              "",
			targetText:              "   ",
			previousFeatures:        nil,
			lastProcessedWasChinese: false,
			expectedLineType:        LineTypeEmpty,
			expectedShouldAdvance:   false,
			expectedFollowPrevStyle: false,
			expectedLinesMatch:      true,
			expectedAddEmptyLine:    false,
		},
		{
			name:                    "Key mismatch with punctuation",
			sourceText:              "Title: \"Jesus Christ, the Same\"",
			targetText:              "Title: Jesus Christ, the Same",
			previousFeatures:        nil,
			lastProcessedWasChinese: false,
			expectedLineType:        LineTypeEnglish,
			expectedShouldAdvance:   false,
			expectedFollowPrevStyle: false,
			expectedLinesMatch:      false, // Should NOT match due to different punctuation
			expectedAddEmptyLine:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := AnalyzeLineMatch(tt.sourceText, tt.targetText, tt.previousFeatures, tt.lastProcessedWasChinese)

			if decision.LineType != tt.expectedLineType {
				t.Errorf("LineType = %v, want %v", decision.LineType, tt.expectedLineType)
			}

			if decision.ShouldAdvanceSource != tt.expectedShouldAdvance {
				t.Errorf("ShouldAdvanceSource = %v, want %v", decision.ShouldAdvanceSource, tt.expectedShouldAdvance)
			}

			if decision.ShouldFollowPrevStyle != tt.expectedFollowPrevStyle {
				t.Errorf("ShouldFollowPrevStyle = %v, want %v", decision.ShouldFollowPrevStyle, tt.expectedFollowPrevStyle)
			}

			if decision.LinesMatch != tt.expectedLinesMatch {
				t.Errorf("LinesMatch = %v, want %v", decision.LinesMatch, tt.expectedLinesMatch)
			}

			if decision.ShouldAddEmptyLine != tt.expectedAddEmptyLine {
				t.Errorf("ShouldAddEmptyLine = %v, want %v", decision.ShouldAddEmptyLine, tt.expectedAddEmptyLine)
			}
		})
	}
}

func TestClassifyLineType(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected LineType
	}{
		{"Pure English", "Heritage Baptist Church Morning Service", LineTypeEnglish},
		{"Pure Chinese", "遗产浸信会早晨崇拜", LineTypeChinese},
		{"Mixed content", "牧师 Alan Fong", LineTypeMixed},
		{"Empty string", "", LineTypeEmpty},
		{"Whitespace only", "   \t\n  ", LineTypeEmpty},
		{"Numbers and punctuation", "123. Title: Test", LineTypeEnglish},
		{"Chinese with punctuation", "标题：\"耶稣基督\"", LineTypeChinese},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyLineType(tt.text)
			if result != tt.expected {
				t.Errorf("classifyLineType(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestGenerateLineKey(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"Simple text", "Heritage Baptist Church", "heritagebaptistchurch"},
		{"With punctuation", "Title: \"Jesus Christ, the Same\"", "title:\"jesuschrist,thesame\""},
		{"With numbers", "1. First Point", "firstpoint"},
		{"With leading Chinese", "遗产 Heritage Baptist", "heritagebaptist"},
		{"No English letters", "遗产浸信会", ""},
		{"Mixed spacing", "Heritage   Baptist    Church", "heritagebaptistchurch"},
		{"Leading punctuation", "a. Point One", "pointone"},
		{"Leading number", "1. Point One", "pointone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateLineKey(tt.text)
			if result != tt.expected {
				t.Errorf("generateLineKey(%q) = %q, want %q", tt.text, result, tt.expected)
			}
		})
	}
}

func TestContainsChinese(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Pure English", "Heritage Baptist Church", false},
		{"Pure Chinese", "遗产浸信会", true},
		{"Mixed content", "牧师 Alan Fong", true},
		{"Empty string", "", false},
		{"Numbers only", "12345", false},
		{"Punctuation only", ".,!?", false},
		{"Chinese with English", "标题：Jesus Christ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsChinese(tt.text)
			if result != tt.expected {
				t.Errorf("containsChinese(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestStartsWithChinese(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Starts with Chinese", "遗产浸信会早晨崇拜", true},
		{"Starts with English", "Heritage Baptist Church", false},
		{"Starts with Chinese, has English", "牧师 Alan Fong", true},
		{"Starts with English, has Chinese", "Pastor 牧师", false},
		{"Empty string", "", false},
		{"Whitespace then Chinese", "  遗产浸信会", true},
		{"Whitespace then English", "  Heritage", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := startsWithChinese(tt.text)
			if result != tt.expected {
				t.Errorf("startsWithChinese(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestOneOff(t *testing.T) {
	// Test the specific case that's failing

	target := "牧师 Alan Fong"

	result := classifyLineType(target)
	t.Logf("Line type for '%s': %v", target, result)
}
