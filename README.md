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

Run the tool with a Google Docs URL:

```bash
go run main.go "https://docs.google.com/document/d/12sJRJ57pNy9zJ6YMD9_HRNuD_UPuWEWnjIBxvul8sRQ/edit?tab=t.0"
```

### Example Output
```
Document ID: 12sJRJ57pNy9zJ6YMD9_HRNuD_UPuWEWnjIBxvul8sRQ
First line: This is the first line of the document
```

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
