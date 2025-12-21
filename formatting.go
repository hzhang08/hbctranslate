package main

import (
	"fmt"
	"strings"

	"google.golang.org/api/docs/v1"
)

// extractFirst10LinesWithFormatting processes the document content and returns the first 10 lines with formatting details
func extractFirst10LinesWithFormatting(doc *docs.Document) string {
	if doc.Body == nil || len(doc.Body.Content) == 0 {
		return ""
	}

	var result strings.Builder
	lineCount := 0
	maxLines := 100 // Changed from 10 to 100

	// Iterate through document content to find paragraphs
	for _, element := range doc.Body.Content {
		if element.Paragraph != nil && len(element.Paragraph.Elements) > 0 && lineCount < maxLines {
			// Check if paragraph has text content
			var textContent strings.Builder
			var firstTextRun *docs.TextRun

			for _, paragraphElement := range element.Paragraph.Elements {
				if paragraphElement.TextRun != nil && strings.TrimSpace(paragraphElement.TextRun.Content) != "" {
					textContent.WriteString(paragraphElement.TextRun.Content)
					if firstTextRun == nil {
						firstTextRun = paragraphElement.TextRun
					}
				}
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

					// Add line header
					result.WriteString(fmt.Sprintf("=== LINE %d ===\n", lineCount))
					result.WriteString(fmt.Sprintf("Text: %s\n", trimmedLine))

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
							result.WriteString(fmt.Sprintf("First Line Indent Unit: %s\n", style.IndentFirstLine.Unit))

						}
						if style.IndentStart != nil {
							result.WriteString(fmt.Sprintf("Left Indent: %.1f pt\n", style.IndentStart.Magnitude))
							result.WriteString(fmt.Sprintf("Left Indent Unit: %s\n", style.IndentStart.Unit))

						}
						if style.IndentEnd != nil {
							result.WriteString(fmt.Sprintf("Right Indent: %.1f pt\n", style.IndentEnd.Magnitude))
							result.WriteString(fmt.Sprintf("Right Indent Unit: %s\n", style.IndentEnd.Unit))

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

					result.WriteString("\n") // Add spacing between lines

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

			for _, paragraphElement := range element.Paragraph.Elements {
				if paragraphElement.TextRun != nil && strings.TrimSpace(paragraphElement.TextRun.Content) != "" {
					textContent.WriteString(paragraphElement.TextRun.Content)
					if firstTextRun == nil {
						firstTextRun = paragraphElement.TextRun
					}
				}
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
