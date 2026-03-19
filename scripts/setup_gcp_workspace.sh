#!/bin/bash

# setup_gcp_workspace.sh
# Automates the creation of a Google Cloud Project, enabling required APIs for Google Workspace,
# and creating a Service Account for use with an MCP Server.

set -e

echo "🚀 Starting Google Cloud Project Setup for Google Workspace MCP..."

# Check for gcloud CLI
if ! command -v gcloud &> /dev/null; then
    echo "❌ Error: Google Cloud SDK (gcloud) is not installed."
    echo "Please install it from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

echo "🔹 Checking authentication..."
# Check if user is authenticated
if ! gcloud auth print-access-token &> /dev/null; then
    echo "You need to log in to Google Cloud. Opening browser..."
    gcloud auth login
fi

# Ask for a project ID
read -p "Enter a new Project ID (e.g., picoclaw-workspace-mcp): " PROJECT_ID

if [ -z "$PROJECT_ID" ]; then
    echo "❌ Error: Project ID cannot be empty."
    exit 1
fi

echo "🔹 Creating Google Cloud Project '$PROJECT_ID'..."
if gcloud projects create "$PROJECT_ID" --name="PicoClaw Workspace MCP"; then
    echo "✅ Project created successfully."
else
    echo "⚠️ Project may already exist or there was an error. Proceeding to set it as default..."
fi

echo "🔹 Setting '$PROJECT_ID' as the default project..."
gcloud config set project "$PROJECT_ID"

# Enable billing prompt (required for some APIs, though these are mostly free tier)
echo "⚠️  Note: Some APIs may require a billing account to be linked to the project."
echo "If the next step fails, you may need to link a billing account in the Google Cloud Console:"
echo "👉 https://console.cloud.google.com/billing/linkedaccount?project=$PROJECT_ID"

echo "🔹 Enabling required Google Workspace APIs..."
# Enable Gmail, Calendar, and Drive APIs
gcloud services enable gmail.googleapis.com
gcloud services enable calendar-json.googleapis.com
gcloud services enable drive.googleapis.com
echo "✅ APIs enabled (Gmail, Calendar, Drive)."

# Create Service Account
SA_NAME="picoclaw-agent"
SA_EMAIL="$SA_NAME@$PROJECT_ID.iam.gserviceaccount.com"
KEY_FILE="credentials.json"

echo "🔹 Creating Service Account '$SA_NAME'..."
if ! gcloud iam service-accounts describe "$SA_EMAIL" &> /dev/null; then
    gcloud iam service-accounts create "$SA_NAME" \
        --description="Service account for PicoClaw MCP Google Workspace integration" \
        --display-name="PicoClaw Agent Service Account"
    echo "✅ Service account created: $SA_EMAIL"
else
    echo "✅ Service account already exists: $SA_EMAIL"
fi

echo "🔹 Generating JSON key file for the Service Account..."
if [ -f "$KEY_FILE" ]; then
    echo "⚠️ Key file '$KEY_FILE' already exists. Saving as new-credentials.json instead."
    KEY_FILE="new-credentials.json"
fi

gcloud iam service-accounts keys create "$KEY_FILE" \
    --iam-account="$SA_EMAIL"

echo "🎉 Setup Complete!"
echo "Your Service Account key has been saved to: $KEY_FILE"
echo ""
echo "================================================================"
echo "⚠️  IMPORTANT: Domain-Wide Delegation Required"
echo "================================================================"
echo "To use this Service Account to access user emails and calendars,"
echo "you must enable Domain-Wide Delegation in your Google Workspace Admin Console."
echo ""
echo "1. Go to: https://admin.google.com/ac/owl/domainwidedelegation"
echo "2. Click 'Add new'"
echo "3. Enter the Client ID for the Service Account (found in $KEY_FILE)"
echo "4. Add the following OAuth scopes (comma separated):"
echo "   https://www.googleapis.com/auth/gmail.modify, \\"
echo "   https://www.googleapis.com/auth/calendar, \\"
echo "   https://www.googleapis.com/auth/drive"
echo "================================================================"
echo "Once configured, you can pass this $KEY_FILE to your chosen MCP server."
