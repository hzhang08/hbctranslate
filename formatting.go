package main

import (
	"fmt"
	"strings"

	"google.golang.org/api/docs/v1"
)

// formatLineFeatures converts LineFeatures to a formatted string
func formatLineFeatures(features *LineFeatures, lineNumber int) string {
	var result strings.Builder

	// Add line header
	result.WriteString(fmt.Sprintf("=== LINE %d ===\n", lineNumber))
	result.WriteString(fmt.Sprintf("Text: %s\n", features.Text))

	// Alignment
	result.WriteString(fmt.Sprintf("Alignment: %s\n", features.Alignment))

	// Indentation
	if features.FirstLineIndent != nil {
		result.WriteString(fmt.Sprintf("First Line Indent: %.1f pt\n", *features.FirstLineIndent))
	}
	if features.LeftIndent != nil {
		result.WriteString(fmt.Sprintf("Left Indent: %.1f pt\n", *features.LeftIndent))
	}
	if features.RightIndent != nil {
		result.WriteString(fmt.Sprintf("Right Indent: %.1f pt\n", *features.RightIndent))
	}

	// Bullet points
	if features.HasBullet {
		result.WriteString("Bullet: Yes\n")
		if features.ListId != "" {
			result.WriteString(fmt.Sprintf("List ID: %s\n", features.ListId))
		}
		result.WriteString(fmt.Sprintf("Nesting Level: %d\n", features.NestingLevel))
	} else {
		result.WriteString("Bullet: No\n")
	}

	// Leading tabs
	result.WriteString(fmt.Sprintf("Leading Tabs: %d\n", features.LeadingTabs))

	// Font family
	if features.FontFamily != "" {
		result.WriteString(fmt.Sprintf("Font: %s\n", features.FontFamily))
	}

	// Font size
	if features.FontSize != nil {
		result.WriteString(fmt.Sprintf("Font Size: %.1f pt\n", *features.FontSize))
	}

	// Bold
	if features.Bold {
		result.WriteString("Bold: Yes\n")
	} else {
		result.WriteString("Bold: No\n")
	}

	// Italic
	if features.Italic {
		result.WriteString("Italic: Yes\n")
	} else {
		result.WriteString("Italic: No\n")
	}

	// Underline
	if features.Underline {
		result.WriteString("Underline: Yes\n")
	} else {
		result.WriteString("Underline: No\n")
	}

	// Text color
	if features.TextColor != nil {
		result.WriteString(fmt.Sprintf("Text Color: RGB(%.0f, %.0f, %.0f)\n",
			features.TextColor.Red, features.TextColor.Green, features.TextColor.Blue))
	}

	result.WriteString("\n")
	return result.String()
}

// extractFirst100LinesWithFormatting processes the document content and returns the first 100 lines with formatting details
func extractFirst100LinesWithFormatting(doc *docs.Document) string {
	if doc.Body == nil || len(doc.Body.Content) == 0 {
		return ""
	}

	var result strings.Builder
	lineCount := 0
	maxLines := 100

	// Iterate through document content to find paragraphs
	for _, element := range doc.Body.Content {
		if element.Paragraph != nil && len(element.Paragraph.Elements) > 0 && lineCount < maxLines {
			// Check if paragraph has text content
			var textContent strings.Builder
			var firstTextRun *docs.TextRun
			tabsFromSkippedRuns := 0

			for _, paragraphElement := range element.Paragraph.Elements {
				if paragraphElement.TextRun == nil {
					continue
				}

				content := paragraphElement.TextRun.Content
				if strings.TrimSpace(content) == "" {
					if firstTextRun == nil {
						tabsFromSkippedRuns += strings.Count(content, "\t")
					}
					continue
				}

				textContent.WriteString(content)
				if firstTextRun == nil {
					firstTextRun = paragraphElement.TextRun
				}
			}

			if firstTextRun != nil && tabsFromSkippedRuns > 0 {
				firstTextRunWithTabs := *firstTextRun
				firstTextRunWithTabs.Content = strings.Repeat("\t", tabsFromSkippedRuns) + firstTextRunWithTabs.Content
				firstTextRun = &firstTextRunWithTabs
			}

			paragraphText := strings.TrimSpace(textContent.String())
			if paragraphText == "" {
				continue
			}

			// Split paragraph into lines
			lines := strings.Split(paragraphText, "\n")
			for _, line := range lines {
				trimmedLine := strings.TrimSpace(line)
				if trimmedLine != "" && lineCount < maxLines {
					lineCount++

					// Use extractLineFeatures to get formatting
					features := extractLineFeatures(element, firstTextRun, trimmedLine)
					result.WriteString(formatLineFeatures(features, lineCount))

					if lineCount >= maxLines {
						break
					}
				}
			}
		}
	}

	if lineCount == 0 {
		return ""
	}

	return strings.TrimSpace(result.String())
}

// extractFirstLineWithFormatting processes the document content and returns the first line with formatting details (legacy function)
func extractFirstLineWithFormatting(doc *docs.Document) string {
	if doc.Body == nil || len(doc.Body.Content) == 0 {
		return ""
	}

	// Find the first paragraph with text content
	for _, element := range doc.Body.Content {
		if element.Paragraph != nil && len(element.Paragraph.Elements) > 0 {
			// Check if paragraph has text content
			var textContent strings.Builder
			var firstTextRun *docs.TextRun
			tabsFromSkippedRuns := 0

			for _, paragraphElement := range element.Paragraph.Elements {
				if paragraphElement.TextRun == nil {
					continue
				}

				content := paragraphElement.TextRun.Content
				if strings.TrimSpace(content) == "" {
					if firstTextRun == nil {
						tabsFromSkippedRuns += strings.Count(content, "\t")
					}
					continue
				}

				textContent.WriteString(content)
				if firstTextRun == nil {
					firstTextRun = paragraphElement.TextRun
				}
			}

			if firstTextRun != nil && tabsFromSkippedRuns > 0 {
				firstTextRunWithTabs := *firstTextRun
				firstTextRunWithTabs.Content = strings.Repeat("\t", tabsFromSkippedRuns) + firstTextRunWithTabs.Content
				firstTextRun = &firstTextRunWithTabs
			}

			firstLineText := strings.TrimSpace(textContent.String())
			if firstLineText == "" {
				continue
			}

			// Extract the first line only
			lines := strings.Split(firstLineText, "\n")
			actualFirstLine := ""
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					actualFirstLine = trimmed
					break
				}
			}

			if actualFirstLine == "" {
				continue
			}

			// Build formatting information
			var result strings.Builder
			result.WriteString(fmt.Sprintf("Text: %s\n", actualFirstLine))

			// Extract paragraph style information
			if element.Paragraph.ParagraphStyle != nil {
				style := element.Paragraph.ParagraphStyle

				// Alignment
				if style.Alignment != "" {
					result.WriteString(fmt.Sprintf("Alignment: %s\n", style.Alignment))
				} else {
					result.WriteString("Alignment: START\n")
				}

				// Indentation
				if style.IndentFirstLine != nil {
					result.WriteString(fmt.Sprintf("First Line Indent: %.1f pt\n", style.IndentFirstLine.Magnitude))
				}
				if style.IndentStart != nil {
					result.WriteString(fmt.Sprintf("Left Indent: %.1f pt\n", style.IndentStart.Magnitude))
				}
				if style.IndentEnd != nil {
					result.WriteString(fmt.Sprintf("Right Indent: %.1f pt\n", style.IndentEnd.Magnitude))
				}

				// Bullet points
				if element.Paragraph.Bullet != nil {
					result.WriteString("Bullet: Yes\n")
					if element.Paragraph.Bullet.ListId != "" {
						result.WriteString(fmt.Sprintf("List ID: %s\n", element.Paragraph.Bullet.ListId))
					}
					result.WriteString(fmt.Sprintf("Nesting Level: %d\n", element.Paragraph.Bullet.NestingLevel))
				} else {
					result.WriteString("Bullet: No\n")
				}
			}

			// Extract text formatting from first text run
			if firstTextRun != nil && firstTextRun.TextStyle != nil {
				textStyle := firstTextRun.TextStyle

				// Font family
				if textStyle.WeightedFontFamily != nil && textStyle.WeightedFontFamily.FontFamily != "" {
					result.WriteString(fmt.Sprintf("Font: %s\n", textStyle.WeightedFontFamily.FontFamily))
				}

				// Font size
				if textStyle.FontSize != nil {
					result.WriteString(fmt.Sprintf("Font Size: %.1f pt\n", textStyle.FontSize.Magnitude))
				}

				// Bold
				if textStyle.Bold {
					result.WriteString("Bold: Yes\n")
				} else {
					result.WriteString("Bold: No\n")
				}

				// Italic
				if textStyle.Italic {
					result.WriteString("Italic: Yes\n")
				} else {
					result.WriteString("Italic: No\n")
				}

				// Underline
				if textStyle.Underline {
					result.WriteString("Underline: Yes\n")
				} else {
					result.WriteString("Underline: No\n")
				}

				// Text color
				if textStyle.ForegroundColor != nil && textStyle.ForegroundColor.Color != nil {
					if textStyle.ForegroundColor.Color.RgbColor != nil {
						rgb := textStyle.ForegroundColor.Color.RgbColor
						result.WriteString(fmt.Sprintf("Text Color: RGB(%.0f, %.0f, %.0f)\n",
							rgb.Red*255, rgb.Green*255, rgb.Blue*255))
					}
				}
			}

			return strings.TrimSpace(result.String())
		}
	}

	return ""
}

// extractFirstLine processes the document content and returns the first line of text (legacy function)
func extractFirstLine(doc *docs.Document) string {
	if doc.Body == nil || len(doc.Body.Content) == 0 {
		return ""
	}

	var allText strings.Builder

	// Iterate through document content
	for _, element := range doc.Body.Content {
		if element.Paragraph != nil {
			// Process paragraph elements
			for _, paragraphElement := range element.Paragraph.Elements {
				if paragraphElement.TextRun != nil {
					allText.WriteString(paragraphElement.TextRun.Content)
				}
			}
		}
	}

	// Get the full text and extract first line
	fullText := strings.TrimSpace(allText.String())
	if fullText == "" {
		return ""
	}

	// Split by newlines and return the first non-empty line
	lines := strings.Split(fullText, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			return trimmedLine
		}
	}

	return ""
}
