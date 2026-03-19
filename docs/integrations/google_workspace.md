# Integrating Google Workspace (Gmail, Calendar, Drive) with PicoClaw

PicoClaw supports seamless integration with Google Workspace (Gmail, Calendar, Drive) by utilizing an external Model Context Protocol (MCP) server. By bridging an external Google Workspace MCP Server to PicoClaw, your AI assistant gains the ability to:

*   **Read, Search, and Send Emails** (Gmail)
*   **Create, View, and Modify Events** (Google Calendar)
*   **Search, Read, and Write Files** (Google Drive)

This approach ensures PicoClaw remains lightweight while giving users full control over their Google Cloud API keys and authentication configuration.

## 1. Choosing an MCP Server

There are several open-source MCP servers built specifically for Google Workspace integration. We recommend:

*   **[aaronsb/google-workspace-mcp](https://github.com/aaronsb/google-workspace-mcp)**: Comprehensive coverage for Gmail, Calendar, and Drive. Supports both OAuth 2.0 (for Multi-Tenant/Personal) and Service Accounts (for Headless/Enterprise).
*   **[epaproditus/google-workspace-mcp-server](https://github.com/epaproditus/google-workspace-mcp-server)**: Excellent for Gmail and Calendar specifically.

## 2. Authentication Setup

Before connecting the server, you must configure a Google Cloud Project with the necessary APIs enabled.

### Option A: Service Accounts (Recommended for Headless / Single-Tenant)
Service Accounts are best for cron jobs or autonomous background agents since they don't require user interaction (browser logins) at runtime. However, they require you to be a Google Workspace Super Admin to grant Domain-Wide Delegation.

To automate the creation of a Google Cloud Project, enabling the APIs, and downloading a Service Account key, run the provided helper script:

```bash
./scripts/setup_gcp_workspace.sh
```

> **Important**: After running the script, you *must* follow the printed instructions to enable Domain-Wide Delegation in your Google Workspace Admin Console (`admin.google.com`), granting the Service Account access to the required scopes (e.g., `https://www.googleapis.com/auth/gmail.modify`, `https://www.googleapis.com/auth/calendar`, `https://www.googleapis.com/auth/drive`).

### Option B: OAuth 2.0 (Recommended for Personal / Multi-Tenant)
If you want to use a standard `@gmail.com` account or build a multi-tenant app where users easily onboard, use OAuth 2.0 User Consent.

1. Go to Google Cloud Console > APIs & Services > Credentials.
2. Create an "OAuth 2.0 Client ID" (Desktop App type).
3. Download the `client_secrets.json` file.
4. When the MCP server starts, it will provide a link to authenticate in your browser.

## 3. Configuring PicoClaw to Use the Server

You can connect the chosen Google Workspace MCP Server to PicoClaw using the native `mcp2cli` tool.

### Using `mcp2cli` dynamically

If you want the agent to call the tools on the fly without heavy upfront JSON configuration, enable the `mcp2cli` tool in your `~/.picoclaw/config.json`:

```json
{
  "tools": {
    "mcp2cli": {
      "enabled": true
    }
  }
}
```

Now, the agent can dynamically connect and execute tools. For example, the agent might automatically execute:

```bash
mcp2cli --mcp-stdio "npx @epaproditus/google-workspace-mcp-server" --env GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json list_emails
```

### (Alternative) Using the Native MCP Manager (`pkg/agent/loop_mcp.go`)

If you want the Gmail/Calendar/Drive tools to be natively registered inside PicoClaw as top-level tools instead of running them via the CLI tool, configure the native MCP block in your `~/.picoclaw/config.json`:

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "google_workspace": {
          "command": "npx",
          "args": ["-y", "@epaproditus/google-workspace-mcp-server"],
          "env": {
            "GOOGLE_APPLICATION_CREDENTIALS": "/path/to/credentials.json"
          }
        }
      }
    }
  }
}
```

When PicoClaw boots, it will connect to the MCP server, retrieve the `send_email`, `list_emails`, and `create_event` tools, and register them directly with the LLM as `mcp_google_workspace_send_email`, etc.