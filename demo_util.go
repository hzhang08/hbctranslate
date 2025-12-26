package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

func getFirstLineFromDoc(docID string) (string, error) {
	ctx := context.Background()

	// Load credentials from file
	credentialsFile := "churchoutline.json"
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return "", fmt.Errorf("churchoutline.json not found. Please follow setup instructions in README.md")
	}

	// Create Docs service using credentials file directly
	docsService, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(docs.DocumentsReadonlyScope))
	if err != nil {
		return "", fmt.Errorf("unable to create Docs service: %v", err)
	}

	// Get the document
	doc, err := docsService.Documents.Get(docID).Do()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve document: %v", err)
	}

	// Extract first 100 lines of text with formatting
	first100LinesInfo := extractFirst100LinesWithFormatting(doc)
	if first100LinesInfo == "" {
		return "", fmt.Errorf("no text content found in document")
	}

	return first100LinesInfo, nil
}
