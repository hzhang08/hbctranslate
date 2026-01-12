package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

type e2eConfig struct {
	Input      string `yaml:"input"`
	OutputName string `yaml:"output_name"`
}

type grokMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type grokRequest struct {
	Input []grokMessage `json:"input"`
	Model string        `json:"model"`
}

func runE2E() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg, err := loadE2EConfig("application.yaml")
	if err != nil {
		log.Fatalf("failed to load application.yaml: %v", err)
	}
	cfg.Input = strings.TrimSpace(cfg.Input)
	cfg.OutputName = strings.TrimSpace(cfg.OutputName)
	if cfg.Input == "" || cfg.OutputName == "" {
		log.Fatalf("application.yaml must include non-empty input and output_name")
	}

	systemPrompt, err := readTextFile("system_prompt")
	if err != nil {
		log.Fatalf("failed to read system_prompt: %v", err)
	}
	prefixPrompt, err := readTextFile("prefix_prompt")
	if err != nil {
		log.Fatalf("failed to read prefix_prompt: %v", err)
	}
	grokKey, err := readTextFile("grokkey")
	if err != nil {
		log.Fatalf("failed to read grokkey: %v", err)
	}
	grokKey = strings.TrimSpace(grokKey)
	if grokKey == "" {
		log.Fatalf("grokkey is empty")
	}

	ctx := context.Background()

	created, driveSrv, err := createGoogleDocWithPublicEdit(ctx, "client_json", "token.json", cfg.OutputName)
	if err != nil {
		log.Fatalf("failed to create output doc: %v", err)
	}
	outputDocID := created.Id
	outputURL := fmt.Sprintf("https://docs.google.com/document/d/%s/edit", outputDocID)
	log.Printf("STEP 1 OK: created output doc: %s", outputURL)

	inputDocID := extractDocumentID(cfg.Input)
	if inputDocID == "" {
		log.Fatalf("failed to parse input doc id from url: %s", cfg.Input)
	}

	serviceAccountEmail, err := extractServiceAccountEmail("churchoutline.json")
	if err != nil {
		log.Fatalf("failed to read service account email from churchoutline.json: %v", err)
	}
	if err := grantWriterToServiceAccount(driveSrv, outputDocID, serviceAccountEmail); err != nil {
		log.Fatalf("failed to grant service account writer permission on output doc: %v", err)
	}
	log.Printf("STEP 1.1 OK: granted service account writer access")

	inputText, err := readGoogleDocPlainText(ctx, "churchoutline.json", inputDocID)
	if err != nil {
		log.Fatalf("failed to read input google doc: %v", err)
	}
	log.Printf("STEP 2 OK: read input Google Doc content")

	userContent := prefixPrompt + "\n" + inputText
	req := grokRequest{
		Input: []grokMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Model: "grok-4",
	}
	log.Printf("STEP 3 OK: built Grok request payload")

	translation, err := callGrokResponses(ctx, grokKey, req)
	if err != nil {
		log.Fatalf("failed to call grok: %v", err)
	}
	if translation == "" {
		log.Fatalf("grok returned empty translation")
	}
	log.Printf("STEP 4 OK: received Grok response")

	if err := writeGoogleDocReplaceAll(ctx, "churchoutline.json", outputDocID, translation); err != nil {
		log.Fatalf("failed to write output google doc: %v", err)
	}
	log.Printf("STEP 5 OK: wrote translation to output Google Doc")

	if !waitForUserReview(outputURL) {
		log.Fatalf("aborted")
	}
	log.Printf("STEP 6 OK: user confirmed review")

	syncDocumentFormatting(cfg.Input, outputURL, 1)
}

func loadE2EConfig(path string) (*e2eConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg e2eConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func readTextFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func createGoogleDocWithPublicEdit(ctx context.Context, oauthClientPath, tokenPath, name string) (*drive.File, *drive.Service, error) {
	b, err := os.ReadFile(oauthClientPath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read oauth client file: %w", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse oauth client file to config: %w", err)
	}

	client, err := getOAuthClient(config, tokenPath)
	if err != nil {
		return nil, nil, err
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create Drive service: %w", err)
	}

	f := &drive.File{Name: name, MimeType: "application/vnd.google-apps.document"}
	created, err := srv.Files.Create(f).Fields("id", "name", "mimeType", "webViewLink").Do()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create Google Doc: %w", err)
	}

	perm := &drive.Permission{Type: "anyone", Role: "writer", AllowFileDiscovery: false}
	_, err = srv.Permissions.Create(created.Id, perm).Fields("id").Do()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to set sharing permission: %w", err)
	}

	return created, srv, nil
}

func getOAuthClient(config *oauth2.Config, tokenPath string) (*http.Client, error) {
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenPath, tok)
	}
	return config.Client(context.Background(), tok), nil
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code:\n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func extractServiceAccountEmail(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return "", err
	}
	v, _ := m["client_email"].(string)
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New("client_email not found")
	}
	return v, nil
}

func grantWriterToServiceAccount(srv *drive.Service, fileID, email string) error {
	perm := &drive.Permission{Type: "user", Role: "writer", EmailAddress: email}
	_, err := srv.Permissions.Create(fileID, perm).Fields("id").Do()
	return err
}

func readGoogleDocPlainText(ctx context.Context, credentialsPath, docID string) (string, error) {
	if _, err := os.Stat(credentialsPath); err != nil {
		return "", fmt.Errorf("credentials file not found: %s: %w", credentialsPath, err)
	}

	svc, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsPath), option.WithScopes(docs.DocumentsReadonlyScope))
	if err != nil {
		return "", fmt.Errorf("unable to create docs service: %w", err)
	}

	doc, err := svc.Documents.Get(docID).Do()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve document: %w", err)
	}
	if doc.Body == nil {
		return "", errors.New("document body is nil")
	}

	var sb strings.Builder
	for _, se := range doc.Body.Content {
		if se == nil || se.Paragraph == nil {
			continue
		}
		for _, pe := range se.Paragraph.Elements {
			if pe == nil || pe.TextRun == nil {
				continue
			}
			sb.WriteString(pe.TextRun.Content)
		}
	}

	text := sb.String()
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return text, nil
}

func writeGoogleDocReplaceAll(ctx context.Context, credentialsPath, docID, newText string) error {
	if _, err := os.Stat(credentialsPath); err != nil {
		return fmt.Errorf("credentials file not found: %s: %w", credentialsPath, err)
	}

	svc, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsPath), option.WithScopes(docs.DocumentsScope))
	if err != nil {
		return fmt.Errorf("unable to create docs service: %w", err)
	}

	doc, err := svc.Documents.Get(docID).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve document: %w", err)
	}
	if doc.Body == nil {
		return errors.New("document body is nil")
	}

	endIndex := int64(1)
	if len(doc.Body.Content) > 0 {
		for _, se := range doc.Body.Content {
			if se == nil {
				continue
			}
			if se.EndIndex != 0 && se.EndIndex > endIndex {
				endIndex = se.EndIndex
			}
		}
	}

	deleteEnd := endIndex - 1
	if deleteEnd < 1 {
		deleteEnd = 1
	}

	reqs := []*docs.Request{}
	if deleteEnd > 1 {
		reqs = append(reqs, &docs.Request{DeleteContentRange: &docs.DeleteContentRangeRequest{Range: &docs.Range{StartIndex: 1, EndIndex: deleteEnd}}})
	}
	reqs = append(reqs, &docs.Request{InsertText: &docs.InsertTextRequest{Location: &docs.Location{Index: 1}, Text: newText}})

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{Requests: reqs}).Do()
	if err != nil {
		return fmt.Errorf("batch update failed: %w", err)
	}
	return nil
}

func callGrokResponses(ctx context.Context, apiKey string, req grokRequest) (string, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.x.ai/v1/responses", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))

	client := &http.Client{Timeout: 60 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("grok api error: status=%s body=%s", resp.Status, string(respBody))
	}

	text, err := extractTextFromXAIResponse(respBody)
	if err != nil {
		return "", fmt.Errorf("unable to parse grok response: %w; raw=%s", err, string(respBody))
	}
	return text, nil
}

func extractTextFromXAIResponse(body []byte) (string, error) {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return "", err
	}

	if v, ok := root["output_text"]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s, nil
		}
	}

	if out, ok := root["output"].([]any); ok {
		var sb strings.Builder
		for _, item := range out {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			content, _ := m["content"].([]any)
			for _, c := range content {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				if t, _ := cm["type"].(string); t == "output_text" || t == "text" {
					if txt, _ := cm["text"].(string); txt != "" {
						sb.WriteString(txt)
					}
				}
			}
		}
		s := sb.String()
		if s != "" {
			return s, nil
		}
	}

	if choices, ok := root["choices"].([]any); ok {
		for _, ch := range choices {
			cm, ok := ch.(map[string]any)
			if !ok {
				continue
			}
			msg, _ := cm["message"].(map[string]any)
			if msg == nil {
				continue
			}
			if content, _ := msg["content"].(string); content != "" {
				return content, nil
			}
		}
	}

	return "", errors.New("no recognizable text field found")
}

func waitForUserReview(outputURL string) bool {
	fmt.Printf("Review output doc: %s\n", outputURL)
	fmt.Print("Press Y then Enter to continue, anything else to abort: ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	return strings.EqualFold(line, "y")
}
