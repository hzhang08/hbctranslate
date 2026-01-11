package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

// Configuration constants
const (
	// FormattingDelaySeconds configures the delay after each formatting application
	FormattingDelaySeconds = 1
)

// Data structures for dual-document synchronization

// RGBColor represents RGB color values
type RGBColor struct {
	Red   float64
	Green float64
	Blue  float64
}

// LineFeatures contains all formatting properties of a line
type LineFeatures struct {
	// Text properties
	Text string

	// Alignment
	Alignment string // START, CENTER, END, JUSTIFIED

	// Indentation
	FirstLineIndent *float64 // First line indent in points
	LeftIndent      *float64 // Left margin indent in points
	RightIndent     *float64 // Right margin indent in points

	// Font properties
	FontFamily string
	FontSize   *float64 // Font size in points

	// Text formatting
	Bold      bool
	Italic    bool
	Underline bool
	TextColor *RGBColor

	// List properties
	HasBullet    bool
	ListId       string
	NestingLevel int64

	// Tab properties
	LeadingTabs int // Number of leading tab characters
}

// LineInfo contains line content and associated formatting
type LineInfo struct {
	Text     string
	Element  *docs.StructuralElement
	TextRun  *docs.TextRun
	Features *LineFeatures
}

// DocumentCursor tracks position in a document
type DocumentCursor struct {
	Document     *docs.Document
	ElementIndex int
	LineIndex    int
	CurrentLine  *LineInfo
}

// SyncError represents synchronization errors
type SyncError struct {
	SourceLine int
	TargetLine int
	SourceKey  string
	TargetKey  string
	Message    string
}

func (e *SyncError) Error() string {
	return fmt.Sprintf("Sync error at source line %d, target line %d: %s (source key: %s, target key: %s)",
		e.SourceLine, e.TargetLine, e.Message, e.SourceKey, e.TargetKey)
}

// BatchUpdateManager manages Google Docs API batch updates
type BatchUpdateManager struct {
	Updates []docs.Request
	DocID   string
}

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "analyze":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go analyze <google-docs-url>")
			os.Exit(1)
		}
		analyzeDocument(os.Args[2])
	case "sync-format":
		fs := flag.NewFlagSet("sync-format", flag.ExitOnError)
		startLoop := fs.Int("start-loop", 1, "")
		_ = fs.Parse(os.Args[2:])
		args := fs.Args()
		if len(args) < 2 {
			fmt.Println("Usage: go run main.go sync-format [--start-loop N] <source-doc-url> <target-doc-url>")
			os.Exit(1)
		}
		if *startLoop < 1 {
			fmt.Println("Error: --start-loop must be >= 1")
			os.Exit(1)
		}
		syncDocumentFormatting(args[0], args[1], *startLoop)
	case "test-action":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go test-action <google-docs-url>")
			os.Exit(1)
		}
		testAction(os.Args[2])
	case "add-spacing":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go add-spacing <google-docs-url>")
			os.Exit(1)
		}
		addSpacingAfterChineseLines(os.Args[2])
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Println("Google Docs Tool")
	fmt.Println("Usage:")
	fmt.Println("  go run main.go analyze <google-docs-url>")
	fmt.Println("  go run main.go sync-format [--start-loop N] <source-doc-url> <target-doc-url>")
	fmt.Println("  go run main.go test-action <google-docs-url>")
	fmt.Println("  go run main.go add-spacing <google-docs-url>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  analyze      Analyze document formatting (first 100 lines)")
	fmt.Println("  sync-format  Synchronize formatting from source to target document")
	fmt.Println("  test-action  Test action for development purposes")
	fmt.Println("  add-spacing  Add empty line after lines starting with Chinese characters")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run main.go analyze \"https://docs.google.com/document/d/12sJRJ57pNy9zJ6YMD9_HRNuD_UPuWEWnjIBxvul8sRQ/edit\"")
	fmt.Println("  go run main.go sync-format \"<source-url>\" \"<target-url>\"")
	fmt.Println("  go run main.go sync-format --start-loop 200 \"<source-url>\" \"<target-url>\"")
	fmt.Println("  go run main.go test-action \"<document-url>\"")
	fmt.Println("  go run main.go add-spacing \"<document-url>\"")
}

func analyzeDocument(docURL string) {
	docID := extractDocumentID(docURL)
	if docID == "" {
		log.Fatal("Invalid Google Docs URL. Please provide a valid document URL.")
	}

	fmt.Printf("Document ID: %s\n", docID)

	firstLine, err := getFirstLineFromDoc(docID)
	if err != nil {
		log.Fatalf("Error reading document: %v", err)
	}

	fmt.Printf("First line: %s\n", firstLine)
}

// testAction updates a Google Doc by converting lines starting with '·' into proper bullet points
func testAction(docURL string) {
	docID := extractDocumentID(docURL)
	if docID == "" {
		log.Fatal("Invalid Google Docs URL. Please provide a valid document URL.")
	}

	fmt.Printf("Document ID: %s\n", docID)

	err := convertDotLinesToBullets(docID)
	if err != nil {
		log.Fatalf("Error updating document: %v", err)
	}
}

func convertDotLinesToBullets(docID string) error {
	ctx := context.Background()

	// Load credentials
	credentialsFile := "churchoutline.json"
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return fmt.Errorf("churchoutline.json not found. Please follow setup instructions in README.md")
	}

	// Create Docs service with write permissions
	docsService, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(docs.DocumentsScope))
	if err != nil {
		return fmt.Errorf("unable to create Docs service: %v", err)
	}

	// Get document to analyze content
	doc, err := docsService.Documents.Get(docID).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve document: %v", err)
	}

	var requests []*docs.Request
	updatedCount := 0

	// Process in reverse order to maintain correct indices
	for i := len(doc.Body.Content) - 1; i >= 0; i-- {
		element := doc.Body.Content[i]
		if element.Paragraph == nil || len(element.Paragraph.Elements) == 0 {
			continue
		}

		var deleteStart, deleteEnd int64
		foundDot := false

		for _, pe := range element.Paragraph.Elements {
			if pe.TextRun == nil {
				continue
			}

			content := pe.TextRun.Content
			if content == "" {
				continue
			}

			runes := []rune(content)
			for runeIdx, r := range runes {
				if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
					continue
				}
				if r == '·' {
					prefixUnits := len(utf16.Encode(runes[:runeIdx]))
					deleteStart = pe.StartIndex + int64(prefixUnits)
					deleteEnd = deleteStart + int64(len(utf16.Encode([]rune{r})))
					foundDot = true
				}
				// Stop at first non-whitespace char, regardless of whether it was dot
				goto firstCharDone
			}
		}
	firstCharDone:

		if !foundDot {
			continue
		}

		// Apply bullets to the paragraph
		requests = append(requests, &docs.Request{
			CreateParagraphBullets: &docs.CreateParagraphBulletsRequest{
				Range: &docs.Range{
					StartIndex: element.StartIndex,
					EndIndex:   element.EndIndex,
				},
				BulletPreset: "BULLET_CHECKBOX",
			},
		})

		// Delete the leading '·' character
		requests = append(requests, &docs.Request{
			DeleteContentRange: &docs.DeleteContentRangeRequest{
				Range: &docs.Range{
					StartIndex: deleteStart,
					EndIndex:   deleteEnd,
				},
			},
		})

		updatedCount++
	}

	if len(requests) == 0 {
		fmt.Println("No lines starting with '·' found.")
		return nil
	}

	fmt.Printf("Converting %d lines starting with '·' into bullet points...\n", updatedCount)

	batchUpdateRequest := &docs.BatchUpdateDocumentRequest{Requests: requests}
	_, err = docsService.Documents.BatchUpdate(docID, batchUpdateRequest).Do()
	if err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}

	return nil
}

// createNewDocWithPublicEdit creates a new Google Doc and sets permissions for anyone with the link to edit
func createNewDocWithPublicEdit() error {
	ctx := context.Background()

	// Load credentials
	credentialsFile := "churchoutline.json"
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return fmt.Errorf("churchoutline.json not found. Please follow setup instructions in README.md")
	}

	// Create Drive service with full permissions
	driveService, err := drive.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(drive.DriveFileScope))
	if err != nil {
		return fmt.Errorf("unable to create Drive service: %v", err)
	}

	// Create a new Google Doc
	doc := &drive.File{
		Title:    "New Document",
		MimeType: "application/vnd.google-apps.document",
	}

	createdFile, err := driveService.Files.Insert(doc).Fields("id, title, alternateLink").Do()
	if err != nil {
		return fmt.Errorf("unable to create document: %v", err)
	}

	fmt.Printf("Document created successfully!\n")
	fmt.Printf("Document ID: %s\n", createdFile.Id)
	fmt.Printf("Document Name: %s\n", createdFile.Title)
	fmt.Printf("Document URL: %s\n", createdFile.AlternateLink)

	// Set permissions: anyone with the link can edit
	permission := &drive.Permission{
		Type: "anyone",
		Role: "writer",
	}

	_, err = driveService.Permissions.Insert(createdFile.Id, permission).Do()
	if err != nil {
		return fmt.Errorf("unable to set permissions: %v", err)
	}

	fmt.Println("\nPermissions set: Anyone with the link can edit")
	fmt.Println("Test action completed successfully!")

	return nil
}

// insertTabsAtLineStart inserts a tab character at the beginning of every line in the document
func insertTabsAtLineStart(docID string) error {
	ctx := context.Background()

	// Load credentials
	credentialsFile := "churchoutline.json"
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return fmt.Errorf("churchoutline.json not found. Please follow setup instructions in README.md")
	}

	// Create Docs service with write permissions
	docsService, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(docs.DocumentsScope))
	if err != nil {
		return fmt.Errorf("unable to create Docs service: %v", err)
	}

	// Get document to find all paragraph ranges
	doc, err := docsService.Documents.Get(docID).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve document: %v", err)
	}

	// Build batch update requests to insert tabs at the start of each paragraph
	var requests []*docs.Request

	// Iterate through document content to find paragraphs
	// We need to process in reverse order to maintain correct indices after insertions
	var paragraphIndices []int64
	for _, element := range doc.Body.Content {
		if element.Paragraph != nil {
			// Check if paragraph has actual text content
			hasText := false
			for _, paragraphElement := range element.Paragraph.Elements {
				if paragraphElement.TextRun != nil && strings.TrimSpace(paragraphElement.TextRun.Content) != "" {
					hasText = true
					break
				}
			}
			if hasText {
				paragraphIndices = append(paragraphIndices, element.StartIndex)
			}
		}
	}

	// Process in reverse order to maintain correct indices
	for i := len(paragraphIndices) - 1; i >= 0; i-- {
		insertRequest := &docs.Request{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{
					Index: paragraphIndices[i],
				},
				Text: "\t",
			},
		}
		requests = append(requests, insertRequest)
	}

	if len(requests) == 0 {
		return fmt.Errorf("no paragraphs found to update")
	}

	fmt.Printf("Inserting tabs at the beginning of %d lines...\n", len(requests))

	// Execute batch update
	batchUpdateRequest := &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}

	_, err = docsService.Documents.BatchUpdate(docID, batchUpdateRequest).Do()
	if err != nil {
		return fmt.Errorf("failed to insert tabs: %v", err)
	}

	return nil
}

// centerAlignDocument aligns all lines in a document to center
func centerAlignDocument(docURL string) {
	docID := extractDocumentID(docURL)
	if docID == "" {
		log.Fatal("Invalid Google Docs URL. Please provide a valid document URL.")
	}

	fmt.Printf("Document ID: %s\n", docID)

	err := applyCenterAlignment(docID)
	if err != nil {
		log.Fatalf("Error applying center alignment: %v", err)
	}

	fmt.Println("Document center alignment completed successfully!")
}

// applyCenterAlignment applies center alignment to all paragraphs in the document
func applyCenterAlignment(docID string) error {
	ctx := context.Background()

	// Load credentials
	credentialsFile := "churchoutline.json"
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return fmt.Errorf("churchoutline.json not found. Please follow setup instructions in README.md")
	}

	// Create Docs service with write permissions
	docsService, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(docs.DocumentsScope))
	if err != nil {
		return fmt.Errorf("unable to create Docs service: %v", err)
	}

	// Get document to find all paragraph ranges
	doc, err := docsService.Documents.Get(docID).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve document: %v", err)
	}

	// Build batch update requests for center alignment
	var requests []*docs.Request

	// Iterate through document content to find paragraphs
	for _, element := range doc.Body.Content {
		if element.Paragraph != nil {
			// Create update request for center alignment
			updateRequest := &docs.Request{
				UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
					Range: &docs.Range{
						StartIndex: element.StartIndex,
						EndIndex:   element.EndIndex,
					},
					ParagraphStyle: &docs.ParagraphStyle{
						Alignment: "CENTER",
					},
					Fields: "alignment",
				},
			}
			requests = append(requests, updateRequest)
		}
	}

	if len(requests) == 0 {
		return fmt.Errorf("no paragraphs found to update")
	}

	fmt.Printf("Applying center alignment to %d paragraphs...\n", len(requests))

	// Execute batch update
	batchUpdateRequest := &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}

	_, err = docsService.Documents.BatchUpdate(docID, batchUpdateRequest).Do()
	if err != nil {
		return fmt.Errorf("failed to apply center alignment: %v", err)
	}

	return nil
}

// addSpacingAfterChineseLines adds empty lines after lines that start with Chinese characters
func addSpacingAfterChineseLines(docURL string) {
	docID := extractDocumentID(docURL)
	if docID == "" {
		log.Fatal("Invalid Google Docs URL. Please provide a valid document URL.")
	}

	fmt.Printf("Document ID: %s\n", docID)

	err := applyChineseLineSpacing(docID)
	if err != nil {
		log.Fatalf("Error adding spacing after Chinese lines: %v", err)
	}

	fmt.Println("Chinese line spacing completed successfully!")
}

// applyChineseLineSpacing adds empty lines after lines that start with Chinese characters
func applyChineseLineSpacing(docID string) error {
	ctx := context.Background()

	// Load credentials
	credentialsFile := "churchoutline.json"
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return fmt.Errorf("churchoutline.json not found. Please follow setup instructions in README.md")
	}

	// Create Docs service with write permissions
	docsService, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(docs.DocumentsScope))
	if err != nil {
		return fmt.Errorf("unable to create Docs service: %v", err)
	}

	// Get document to analyze content
	doc, err := docsService.Documents.Get(docID).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve document: %v", err)
	}

	// Find lines that start with Chinese characters and collect insertion points
	var insertRequests []*docs.Request

	// Process document content in reverse order to maintain correct indices
	for i := len(doc.Body.Content) - 1; i >= 0; i-- {
		element := doc.Body.Content[i]
		if element.Paragraph != nil && len(element.Paragraph.Elements) > 0 {
			// Get the text content of the paragraph
			var paragraphText strings.Builder
			for _, elem := range element.Paragraph.Elements {
				if elem.TextRun != nil {
					paragraphText.WriteString(elem.TextRun.Content)
				}
			}

			text := strings.TrimSpace(paragraphText.String())
			if text != "" && startsWithChinese(text) {
				// Insert empty paragraph after this line
				insertRequest := &docs.Request{
					InsertText: &docs.InsertTextRequest{
						Location: &docs.Location{
							Index: element.EndIndex,
						},
						Text: "\n",
					},
				}
				insertRequests = append(insertRequests, insertRequest)
			}
		}
	}

	if len(insertRequests) == 0 {
		fmt.Println("No lines starting with Chinese characters found.")
		return nil
	}

	fmt.Printf("Adding empty lines after %d Chinese lines...\n", len(insertRequests))

	// Execute batch update
	batchUpdateRequest := &docs.BatchUpdateDocumentRequest{
		Requests: insertRequests,
	}

	_, err = docsService.Documents.BatchUpdate(docID, batchUpdateRequest).Do()
	if err != nil {
		return fmt.Errorf("failed to add spacing after Chinese lines: %v", err)
	}

	return nil
}

// applyFormattingToRange applies LineFeatures to a specific range in the target document
func applyFormattingToRange(docsService *docs.Service, docID string, startIndex, endIndex int64, features *LineFeatures) error {

	// Add configurable delay after formatting application to avoid API rate limits
	time.Sleep(time.Duration(FormattingDelaySeconds) * time.Second)
	fmt.Printf("Applying features to range [%d,%d)\n", startIndex, endIndex)

	var requests []*docs.Request

	// Print all the features applied to this line
	fmt.Println("Applying the following features to lines:")
	fmt.Printf("Alignment: %s\n", features.Alignment)
	if features.FirstLineIndent != nil {
		fmt.Printf("First Line Indent: %f\n", *features.FirstLineIndent)
	}
	if features.LeftIndent != nil {
		fmt.Printf("Left Indent: %f\n", *features.LeftIndent)
	}
	if features.RightIndent != nil {
		fmt.Printf("Right Indent: %f\n", *features.RightIndent)
	}
	fmt.Printf("Has Bullet: %t\n", features.HasBullet)
	fmt.Printf("Leading Tabs: %d\n", features.LeadingTabs)

	// Apply paragraph style (alignment, indentation, bullets)
	if features.Alignment != "" || features.FirstLineIndent != nil || features.LeftIndent != nil || features.RightIndent != nil || features.HasBullet {
		paragraphStyle := &docs.ParagraphStyle{}
		fields := []string{}

		if features.Alignment != "" {
			paragraphStyle.Alignment = features.Alignment
			fields = append(fields, "alignment")
		}

		if features.FirstLineIndent != nil && *features.FirstLineIndent != 0 {
			paragraphStyle.IndentFirstLine = &docs.Dimension{
				Magnitude: *features.FirstLineIndent,
				Unit:      "PT",
			}
			fields = append(fields, "indentFirstLine")
		}

		if features.LeftIndent != nil && *features.LeftIndent != 0 {
			paragraphStyle.IndentStart = &docs.Dimension{
				Magnitude: *features.LeftIndent,
				Unit:      "PT",
			}
			fields = append(fields, "indentStart")
		}

		if features.RightIndent != nil && *features.RightIndent != 0 {
			paragraphStyle.IndentEnd = &docs.Dimension{
				Magnitude: *features.RightIndent,
				Unit:      "PT",
			}
			fields = append(fields, "indentEnd")
		}

		if len(fields) > 0 {
			request := &docs.Request{
				UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
					Range: &docs.Range{
						StartIndex: startIndex,
						EndIndex:   endIndex,
					},
					ParagraphStyle: paragraphStyle,
					Fields:         strings.Join(fields, ","),
				},
			}
			requests = append(requests, request)
		}
	}

	// Apply text style (font, size, bold, italic, underline, color)
	if features.FontFamily != "" || features.FontSize != nil || features.Bold || features.Italic || features.Underline || features.TextColor != nil {
		textStyle := &docs.TextStyle{}
		fields := []string{}

		if features.FontFamily != "" {
			textStyle.WeightedFontFamily = &docs.WeightedFontFamily{
				FontFamily: features.FontFamily,
			}
			fields = append(fields, "weightedFontFamily")
		}

		if features.FontSize != nil && *features.FontSize != 0 {
			textStyle.FontSize = &docs.Dimension{
				Magnitude: *features.FontSize,
				Unit:      "PT",
			}
			fields = append(fields, "fontSize")
		}

		if features.Bold {
			textStyle.Bold = true
			fields = append(fields, "bold")
		}

		if features.Italic {
			textStyle.Italic = true
			fields = append(fields, "italic")
		}

		if features.Underline {
			textStyle.Underline = true
			fields = append(fields, "underline")
		}

		if features.TextColor != nil {
			// Apply RGB color from RGBColor struct
			textStyle.ForegroundColor = &docs.OptionalColor{
				Color: &docs.Color{
					RgbColor: &docs.RgbColor{
						Red:   features.TextColor.Red,
						Green: features.TextColor.Green,
						Blue:  features.TextColor.Blue,
					},
				},
			}
			fields = append(fields, "foregroundColor")
		}

		if len(fields) > 0 {
			request := &docs.Request{
				UpdateTextStyle: &docs.UpdateTextStyleRequest{
					Range: &docs.Range{
						StartIndex: startIndex,
						EndIndex:   endIndex,
					},
					TextStyle: textStyle,
					Fields:    strings.Join(fields, ","),
				},
			}
			requests = append(requests, request)
		}
	}

	// Execute batch update if we have requests
	if len(requests) > 0 {
		batchUpdateRequest := &docs.BatchUpdateDocumentRequest{
			Requests: requests,
		}

		_, err := docsService.Documents.BatchUpdate(docID, batchUpdateRequest).Do()
		if err != nil {
			return fmt.Errorf("failed to apply formatting: %v", err)
		}
	}

	return nil
}

// syncDocumentFormatting synchronizes formatting from source to target document
func syncDocumentFormatting(sourceURL, targetURL string, startLoop int) {
	sourceDocID := extractDocumentID(sourceURL)
	targetDocID := extractDocumentID(targetURL)

	if sourceDocID == "" {
		log.Fatal("Invalid source Google Docs URL. Please provide a valid document URL.")
	}
	if targetDocID == "" {
		log.Fatal("Invalid target Google Docs URL. Please provide a valid document URL.")
	}

	fmt.Printf("Source Document ID: %s\n", sourceDocID)
	fmt.Printf("Target Document ID: %s\n", targetDocID)

	err := processDualDocuments(sourceDocID, targetDocID, startLoop)
	if err != nil {
		log.Fatalf("Error synchronizing documents: %v", err)
	}

	fmt.Println("Document formatting synchronization completed successfully!")
}

// Utility functions for dual-document synchronization

// extractLineFeatures extracts all formatting features from a document element and text run
func extractLineFeatures(element *docs.StructuralElement, textRun *docs.TextRun, text string) *LineFeatures {
	features := &LineFeatures{
		Text: text,
	}

	// Count leading tabs from textRun.Content
	leadingTabs := 0
	if textRun != nil && textRun.Content != "" {
		for _, char := range textRun.Content {
			if char == '\t' {
				leadingTabs++
			} else {
				break
			}
		}
	}
	features.LeadingTabs = leadingTabs

	// Extract paragraph style information
	if element.Paragraph != nil && element.Paragraph.ParagraphStyle != nil {
		style := element.Paragraph.ParagraphStyle

		// Alignment
		if style.Alignment != "" {
			features.Alignment = style.Alignment
		} else {
			features.Alignment = "START"
		}

		// Indentation
		if style.IndentFirstLine != nil {
			indent := style.IndentFirstLine.Magnitude
			features.FirstLineIndent = &indent
		}
		if style.IndentStart != nil {
			indent := style.IndentStart.Magnitude
			features.LeftIndent = &indent
		}
		if style.IndentEnd != nil {
			indent := style.IndentEnd.Magnitude
			features.RightIndent = &indent
		}

		// Bullet points
		if element.Paragraph.Bullet != nil {
			features.HasBullet = true
			features.ListId = element.Paragraph.Bullet.ListId
			features.NestingLevel = element.Paragraph.Bullet.NestingLevel
		}
	}

	// Extract text formatting from text run
	if textRun != nil && textRun.TextStyle != nil {
		textStyle := textRun.TextStyle

		// Font family
		if textStyle.WeightedFontFamily != nil && textStyle.WeightedFontFamily.FontFamily != "" {
			features.FontFamily = textStyle.WeightedFontFamily.FontFamily
		}

		// Font size
		if textStyle.FontSize != nil {
			size := textStyle.FontSize.Magnitude
			features.FontSize = &size
		}

		// Text formatting
		features.Bold = textStyle.Bold
		features.Italic = textStyle.Italic
		features.Underline = textStyle.Underline

		// Text color
		if textStyle.ForegroundColor != nil && textStyle.ForegroundColor.Color != nil {
			if textStyle.ForegroundColor.Color.RgbColor != nil {
				rgb := textStyle.ForegroundColor.Color.RgbColor
				features.TextColor = &RGBColor{
					Red:   rgb.Red * 255,
					Green: rgb.Green * 255,
					Blue:  rgb.Blue * 255,
				}
			}
		}
	}

	return features
}

// processDualDocuments implements the main dual-document synchronization algorithm
func processDualDocuments(sourceDocID, targetDocID string, startLoop int) error {
	ctx := context.Background()

	// Load credentials
	credentialsFile := "churchoutline.json"
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return fmt.Errorf("churchoutline.json not found. Please follow setup instructions in README.md")
	}

	// Create Docs service with write permissions
	docsService, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(docs.DocumentsScope))
	if err != nil {
		return fmt.Errorf("unable to create Docs service: %v", err)
	}

	// Get source document
	sourceDoc, err := docsService.Documents.Get(sourceDocID).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve source document: %v", err)
	}

	// Get target document
	targetDoc, err := docsService.Documents.Get(targetDocID).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve target document: %v", err)
	}

	// Initialize cursors
	sourceCursor := &DocumentCursor{Document: sourceDoc, ElementIndex: 0, LineIndex: 0}
	targetCursor := &DocumentCursor{Document: targetDoc, ElementIndex: 0, LineIndex: 0}

	// Process documents
	return synchronizeDocuments(sourceCursor, targetCursor, targetDocID, docsService, startLoop)
}

// synchronizeDocuments performs the actual synchronization between two documents
func synchronizeDocuments(sourceCursor, targetCursor *DocumentCursor, targetDocID string, docsService *docs.Service, startLoop int) error {
	sourceLineNum := 0
	targetLineNum := 0
	var previousFeatures *LineFeatures

	// Track what was processed last - start with false so we read the first source line normally
	lastProcessedWasChinese := false

	// Hashmap to track tabs to add for each loop iteration
	tabsToAddMap := make(map[int]int)

	var sourceLineInfo *LineInfo
	var sourceKey string
	var sourceFeatures *LineFeatures

	// Read the first source line
	sourceLineInfo, err := getNextNonEmptyLine(sourceCursor)
	if err != nil {
		return fmt.Errorf("error reading first source line: %v", err)
	}
	sourceLineNum++
	sourceKey = generateLineKey(sourceLineInfo.Text)
	sourceFeatures = extractLineFeatures(sourceLineInfo.Element, sourceLineInfo.TextRun, sourceLineInfo.Text)
	fmt.Printf("Source Line %d: %s (key: %s)\n", sourceLineNum, sourceLineInfo.Text, sourceKey)

	fmt.Println("Starting document synchronization...")

	if startLoop > 1 {
		fmt.Printf("Fast-forwarding to loop %d...\n", startLoop)
		for loopID := 1; loopID < startLoop; loopID++ {
			if shouldAdvanceSourceCursor(lastProcessedWasChinese) {
				var err error
				sourceLineInfo, err = getNextNonEmptyLine(sourceCursor)
				if err != nil {
					if err.Error() == "end of document" {
						return fmt.Errorf("reached end of source document while fast-forwarding at source line %d", sourceLineNum)
					}
					return fmt.Errorf("error reading source document while fast-forwarding: %v", err)
				}
				sourceLineNum++
				sourceKey = generateLineKey(sourceLineInfo.Text)
				sourceFeatures = extractLineFeatures(sourceLineInfo.Element, sourceLineInfo.TextRun, sourceLineInfo.Text)
			}

			targetLineInfo, err := getNextNonEmptyLine(targetCursor)
			if err != nil {
				if err.Error() == "end of document" {
					return fmt.Errorf("reached end of target document while fast-forwarding at target line %d", targetLineNum)
				}
				return fmt.Errorf("error reading target document while fast-forwarding: %v", err)
			}
			targetLineNum++

			decision := AnalyzeLineMatch(sourceLineInfo.Text, targetLineInfo.Text, previousFeatures, lastProcessedWasChinese)
			if decision.LineType == LineTypeChinese || decision.LineType == LineTypeMixed {
				lastProcessedWasChinese = true
			} else {
				targetKey := generateLineKey(targetLineInfo.Text)
				if !decision.LinesMatch {
					return &SyncError{
						SourceLine: sourceLineNum,
						TargetLine: targetLineNum,
						SourceKey:  sourceKey,
						TargetKey:  targetKey,
						Message:    "Line content mismatch",
					}
				}
				previousFeatures = sourceFeatures
				lastProcessedWasChinese = false
			}
		}
	}

	for loopID := startLoop; ; loopID++ {
		fmt.Printf("Loop %d, lastProcessedWasChinese: %v\n", loopID, lastProcessedWasChinese)
		// Only advance source cursor if the last target line processed was Chinese
		if shouldAdvanceSourceCursor(lastProcessedWasChinese) {
			// Advance to next English source line (source only has English lines)
			var err error
			sourceLineInfo, err = getNextNonEmptyLine(sourceCursor)
			if err != nil {
				if err.Error() == "end of document" {
					fmt.Printf("Reached end of source document at line %d\n", sourceLineNum)
					break
				}
				return fmt.Errorf("error reading source document: %v", err)
			}
			sourceLineNum++

			// Generate key and extract features from source line
			sourceKey = generateLineKey(sourceLineInfo.Text)
			sourceFeatures = extractLineFeatures(sourceLineInfo.Element, sourceLineInfo.TextRun, sourceLineInfo.Text)

			fmt.Printf("Source Line %d: %s (key: %s)\n", sourceLineNum, sourceLineInfo.Text, sourceKey)
		}

		// Always move target cursor to next non-empty line
		targetLineInfo, err := getNextNonEmptyLine(targetCursor)
		if err != nil {
			if err.Error() == "end of document" {
				fmt.Printf("Reached end of target document at line %d\n", targetLineNum)
				break
			}
			return fmt.Errorf("error reading target document: %v", err)
		}
		targetLineNum++

		// Use matcher to analyze the line and make decisions
		decision := AnalyzeLineMatch(sourceLineInfo.Text, targetLineInfo.Text, previousFeatures, lastProcessedWasChinese)
		fmt.Printf("Line Decision: %+v\n", decision)

		if decision.LineType == LineTypeChinese || decision.LineType == LineTypeMixed {
			fmt.Printf("Target Line %d (Chinese): %s - applying previous formatting\n", targetLineNum, targetLineInfo.Text)

			// Check if previous features contain tabs
			if previousFeatures != nil && previousFeatures.LeadingTabs > 0 {
				tabsToAddMap[loopID] = previousFeatures.LeadingTabs
				fmt.Printf("  Chinese line: will add %d tabs (from previous features)\n", previousFeatures.LeadingTabs)
			}

			// Apply previous line's formatting if available and decision recommends it
			if decision.ShouldFollowPrevStyle && previousFeatures != nil {
				err := applyFormattingToRange(docsService, targetDocID, targetLineInfo.Element.StartIndex, targetLineInfo.Element.EndIndex, previousFeatures)
				if err != nil {
					fmt.Printf("  Warning: Failed to apply formatting: %v\n", err)
				} else {
					fmt.Printf("  Applied formatting from previous line\n")
				}
			}

			// Mark that we processed a Chinese line - this will trigger source advance next iteration
			lastProcessedWasChinese = true
		} else {
			// Generate key for English target line
			targetKey := generateLineKey(targetLineInfo.Text)
			fmt.Printf("Target Line %d (English): %s (key: %s)\n", targetLineNum, targetLineInfo.Text, targetKey)

			// Check if keys match with current source line
			if !decision.LinesMatch {
				return &SyncError{
					SourceLine: sourceLineNum,
					TargetLine: targetLineNum,
					SourceKey:  sourceKey,
					TargetKey:  targetKey,
					Message:    "Line content mismatch",
				}
			}

			// Check if source features contain tabs
			if sourceFeatures.LeadingTabs > 0 {
				tabsToAddMap[loopID] = sourceFeatures.LeadingTabs
				fmt.Printf("  English line: will add %d tabs (from source features)\n", sourceFeatures.LeadingTabs)
			}

			// Keys match - apply source formatting to target
			err := applyFormattingToRange(docsService, targetDocID, targetLineInfo.Element.StartIndex, targetLineInfo.Element.EndIndex, sourceFeatures)
			if err != nil {
				fmt.Printf("  Warning: Failed to apply formatting: %v\n", err)
			} else {
				fmt.Printf("  Keys match - applied source formatting\n")
			}
			previousFeatures = sourceFeatures

			// Mark that we processed an English line - source cursor stays on same line
			lastProcessedWasChinese = false
		}
	}

	// Print the tabs to add hashmap at the end
	fmt.Println("\n=== Tabs to Add Summary ===")
	if len(tabsToAddMap) == 0 {
		fmt.Println("No tabs to add for any loop iteration")
	} else {
		fmt.Printf("Total loop iterations with tabs: %d\n", len(tabsToAddMap))
		for loopID := 1; loopID <= len(tabsToAddMap)+10; loopID++ {
			if tabs, exists := tabsToAddMap[loopID]; exists {
				fmt.Printf("Loop %d: %d tab(s)\n", loopID, tabs)
			}
		}
	}
	fmt.Println("==========================")

	// Now insert tabs at the beginning of lines in target document
	if len(tabsToAddMap) > 0 {
		fmt.Println("Inserting tabs into target document...")

		// Re-fetch the target document to get fresh indices
		targetDoc, err := docsService.Documents.Get(targetDocID).Do()
		if err != nil {
			return fmt.Errorf("unable to re-fetch target document: %v", err)
		}

		// Reset target cursor to start of document
		targetCursor = &DocumentCursor{Document: targetDoc, ElementIndex: 0, LineIndex: 0}

		// Collect line start indices for each loop iteration
		lineStartIndices := make(map[int]int64)
		currentLoopID := 1

		for {
			lineInfo, err := getNextNonEmptyLine(targetCursor)
			if err != nil {
				if err.Error() == "end of document" {
					break
				}
				return fmt.Errorf("error reading target document for tab insertion: %v", err)
			}

			// Store the start index for this loop iteration
			lineStartIndices[currentLoopID] = lineInfo.Element.StartIndex
			currentLoopID++
		}

		// Build batch insert requests in reverse order to maintain correct indices
		var insertRequests []*docs.Request

		// Process in reverse order of loop IDs
		for loopID := currentLoopID - 1; loopID >= 1; loopID-- {
			if numTabs, exists := tabsToAddMap[loopID]; exists {
				if startIndex, hasIndex := lineStartIndices[loopID]; hasIndex {
					tabs := strings.Repeat("\t", numTabs)
					insertRequest := &docs.Request{
						InsertText: &docs.InsertTextRequest{
							Location: &docs.Location{
								Index: startIndex,
							},
							Text: tabs,
						},
					}
					insertRequests = append(insertRequests, insertRequest)
					fmt.Printf("  Preparing to insert %d tab(s) at loop %d (index %d)\n", numTabs, loopID, startIndex)
				}
			}
		}

		// Execute batch insert
		if len(insertRequests) > 0 {
			batchUpdateRequest := &docs.BatchUpdateDocumentRequest{
				Requests: insertRequests,
			}

			_, err = docsService.Documents.BatchUpdate(targetDocID, batchUpdateRequest).Do()
			if err != nil {
				return fmt.Errorf("failed to insert tabs: %v", err)
			}

			fmt.Printf("Successfully inserted tabs at %d locations\n", len(insertRequests))
		}
	}

	return nil
}

// getNextNonEmptyLine advances cursor to the next non-empty line
func getNextNonEmptyLine(cursor *DocumentCursor) (*LineInfo, error) {
	if cursor.Document.Body == nil || len(cursor.Document.Body.Content) == 0 {
		return nil, fmt.Errorf("empty document")
	}

	// Continue from current position
	for cursor.ElementIndex < len(cursor.Document.Body.Content) {
		element := cursor.Document.Body.Content[cursor.ElementIndex]

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

			paragraphText := strings.TrimSpace(textContent.String())
			if paragraphText != "" {
				// Split paragraph into lines
				lines := strings.Split(paragraphText, "\n")

				// Continue from current line index within this paragraph
				for cursor.LineIndex < len(lines) {
					line := strings.TrimSpace(lines[cursor.LineIndex])
					if line != "" {
						lineInfo := &LineInfo{
							Text:    line,
							Element: element,
							TextRun: firstTextRun,
						}

						// Advance to next line for next call
						cursor.LineIndex++
						if cursor.LineIndex >= len(lines) {
							cursor.ElementIndex++
							cursor.LineIndex = 0
						}

						return lineInfo, nil
					}
					cursor.LineIndex++
				}

				// Reset line index and move to next element
				cursor.LineIndex = 0
			}
		}

		cursor.ElementIndex++
	}

	return nil, fmt.Errorf("end of document")
}

// extractDocumentID extracts the document ID from a Google Docs URL
func extractDocumentID(url string) string {
	// Pattern to match Google Docs URLs and extract document ID
	re := regexp.MustCompile(`/document/d/([a-zA-Z0-9-_]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// getFirstLineFromDoc reads a Google Doc and returns the first line of text
