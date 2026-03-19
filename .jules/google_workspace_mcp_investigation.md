# Integrating Google Workspace into PicoClaw using MCP

## Overview

The goal is to provide PicoClaw agents with the ability to read and mutate data in a user's Google Workspace, specifically Gmail, Google Calendar, and Google Drive. Given the requirement to keep this integration as an "off-context separate service" and the existing support for Model Context Protocol (MCP) in PicoClaw (both natively via `pkg/agent/loop_mcp.go` and dynamically via `mcp2cli`), using an external MCP Server is the recommended architectural approach.

## Recommended External Service Architecture

We recommend deploying a specialized Google Workspace MCP server that handles authentication and exposes tools for Gmail, Calendar, and Drive.

There are several open-source MCP servers built for this exact purpose:
1. **[epaproditus/google-workspace-mcp-server](https://github.com/epaproditus/google-workspace-mcp-server)**: Provides robust Gmail (`list_emails`, `search_emails`, `send_email`) and Calendar APIs.
2. **[aaronsb/google-workspace-mcp](https://github.com/aaronsb/google-workspace-mcp)**: Comprehensive coverage for Gmail, Calendar, and Drive.
3. **[ghaziahamat/google-workspace-mcp](https://github.com/ghaziahamat/google-workspace-mcp)**: A Python-based collection of independent servers for Gmail, Calendar, and Drive.

These servers can run locally via Docker, `npx`, or Python `uvx`, and PicoClaw can connect to them seamlessly using its existing `mcp2cli` tool.

### Example PicoClaw Tool Execution via `mcp2cli`

PicoClaw's `mcp2cli` tool allows dynamic execution without loading full JSON schemas upfront:

```bash
# Example agent call internally
mcp2cli --mcp-stdio "npx @epaproditus/google-workspace-mcp-server" send_email --to "user@example.com" --subject "Hello" --body "Test message"
```

## Authentication & Onboarding Strategy

### 1. Service Accounts (Single-Tenant / Headless Automation)

Since you are currently testing and prefer a low-friction setup for yourself, **Service Accounts** are the easiest starting point for backend, headless automation.

**Pros:**
- No browser popup or user interaction required at runtime.
- Excellent for cron jobs, background processing, and server-side automation.

**Cons:**
- Requires Google Workspace **Domain-Wide Delegation**, which means you must be a Google Workspace Super Admin.
- If you are building a multi-tenant SaaS application, you cannot easily use Service Accounts to access emails of users outside your organization (e.g., public `@gmail.com` accounts).

### 2. OAuth 2.0 User Consent (Multi-Tenant / Production)

For a multi-tenant application where users "easily onboard their gsuite", **OAuth 2.0 Authorization Code Flow** is the industry standard.

**Pros:**
- Users simply click "Sign in with Google", grant permissions, and return to your app.
- Works for both personal `@gmail.com` accounts and enterprise Google Workspace accounts.
- The external MCP server (or your backend) stores a `refresh_token` to make API calls on the user's behalf indefinitely.

**Cons:**
- Requires setting up an OAuth Consent Screen in Google Cloud Console.
- Your app needs to be verified by Google if you request sensitive scopes (like Gmail read/write or Drive).

## Automating the GCP Project Setup

To help you get started quickly with the Service Account approach, we will provide a helper script (`scripts/setup_gcp_workspace.sh`) that automates the creation of a Google Cloud Project, enables the necessary APIs, and creates a Service Account with a downloadable JSON key.

### The Automated Flow:
1. Authenticate with `gcloud auth login`.
2. Create a new Google Cloud Project.
3. Enable `gmail.googleapis.com`, `calendar-json.googleapis.com`, and `drive.googleapis.com`.
4. Create a Service Account (`picoclaw-workspace-agent@<project>.iam.gserviceaccount.com`).
5. Download the `credentials.json` key file.

*Note: After running this script, you will still need to manually configure Domain-Wide Delegation in the Google Workspace Admin Console (`admin.google.com`) to allow this Service Account to act on behalf of your email address.*