# Google Docs Reader Tool

A Golang CLI tool that reads a Google Document and extracts the first line of text content using the Google Docs API.

## Prerequisites

1. **Google Cloud Console Setup**
   - Create a new project in [Google Cloud Console](https://console.cloud.google.com/)
   - Enable the Google Docs API
   - Create credentials (Service Account recommended)

2. **Go Installation**
   - Go 1.21 or later

## Setup Instructions

### Step 1: Enable Google Docs API

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Navigate to "APIs & Services" > "Library"
4. Search for "Google Docs API" and enable it

### Step 2: Create Credentials

#### Option A: Service Account (Recommended)
1. Go to "APIs & Services" > "Credentials"
2. Click "Create Credentials" > "Service Account"
3. Fill in the service account details
4. Click "Create and Continue"
5. Skip role assignment (click "Continue")
6. Click "Done"
7. Click on the created service account
8. Go to "Keys" tab
9. Click "Add Key" > "Create New Key"
10. Select "JSON" and click "Create"
11. Save the downloaded file as `credentials.json` in the project root

#### Option B: OAuth 2.0 (Alternative)
1. Go to "APIs & Services" > "Credentials"
2. Click "Create Credentials" > "OAuth client ID"
3. Select "Desktop application"
4. Download the credentials file and save as `credentials.json`

### Step 3: Share Document (Service Account Only)
If using a service account, share the Google Doc with the service account email:
1. Open your Google Doc
2. Click "Share"
3. Add the service account email (found in credentials.json)
4. Give "Viewer" permissions

## Installation

1. Clone or download this project
2. Place your `credentials.json` file in the project root
3. Install dependencies:
   ```bash
   go mod tidy
   ```

## Usage

### Basic Document Reading

Run the tool with a Google Docs URL:

```bash
go run . "https://docs.google.com/document/d/12sJRJ57pNy9zJ6YMD9_HRNuD_UPuWEWnjIBxvul8sRQ/edit?tab=t.0"
```

#### Example Output
```
Document ID: 12sJRJ57pNy9zJ6YMD9_HRNuD_UPuWEWnjIBxvul8sRQ
First line: This is the first line of the document
```

### Analyze Document Formatting

Analyze and display formatting details for the first 100 lines of a document:

```bash
go run . analyze "https://docs.google.com/document/d/1CitsXplsjF0q6SgdjlwyFd_4el5TPQa63YzflnTD6Z8"
```

This command will:
- Extract and display formatting features for each line
- Show alignment (START, CENTER, END, JUSTIFIED)
- Display indentation values (first line, left, right)
- Show font details (family, size)
- Display text styling (bold, italic, underline)
- Indicate bullet point presence

### Document Synchronization and Formatting

Synchronize formatting between two documents (source and target):

```bash
go run . sync-format "https://docs.google.com/document/d/1FZcELu78lNNvjSEitEpUn_n0B0xCfgrq3uI_1d_Hm7g/edit?tab=t.0" "https://docs.google.com/document/d/1KLQOlC6lY3JujsTtoRxDF79SsVE5DWouNV0JBBAB8kQ/edit?tab=t.0"
```

This command will:
- Read formatting from the source document (first URL)
- Apply matching formatting to corresponding lines in the target document (second URL)
- Handle bilingual documents with English and Chinese content
- Automatically detect line types and apply appropriate formatting rules

### Test Action (Development)

Perform test actions on a document for development and debugging purposes:

```bash
go run . test-action "https://docs.google.com/document/d/1KLQOlC6lY3JujsTtoRxDF79SsVE5DWouNV0JBBAB8kQ/edit?tab=t.0"
```

This command currently:
- Inserts a tab character at the beginning of every line in the document
- Processes paragraphs in reverse order to maintain correct character indices
- Useful for testing document manipulation and API interactions

### Adding Spacing After Chinese Lines

Add empty lines after Chinese text for better readability:

```bash
go run . add-spacing "https://docs.google.com/document/d/1KLQOlC6lY3JujsTtoRxDF79SsVE5DWouNV0JBBAB8kQ/edit?tab=t.0"
```

This command will:
- Scan the document for lines that start with Chinese characters
- Automatically insert empty lines after Chinese text
- Improve document readability and formatting

## Building

To build a standalone executable:

```bash
go build -o googledoc-reader main.go
./googledoc-reader "https://docs.google.com/document/d/YOUR_DOCUMENT_ID/edit"
```

## Troubleshooting

### Common Issues

1. **"credentials.json not found"**
   - Ensure the credentials file is in the project root directory
   - Check the filename is exactly `credentials.json`

2. **"Permission denied" or "Document not found"**
   - For service accounts: Share the document with the service account email
   - For OAuth: Ensure you have access to the document
   - Check that the document ID is correct

3. **"API not enabled"**
   - Enable Google Docs API in Google Cloud Console
   - Wait a few minutes for the API to be fully enabled

4. **"Invalid credentials"**
   - Re-download credentials from Google Cloud Console
   - Ensure the JSON file is valid and not corrupted

### Getting Help

- Check the [Google Docs API documentation](https://developers.google.com/docs/api)
- Verify your Google Cloud Console setup
- Ensure the document is accessible and not private

## Security Notes

- Keep `credentials.json` secure and never commit it to version control
- Add `credentials.json` to your `.gitignore` file
- Use environment variables for production deployments
- Follow the principle of least privilege when setting up API access
