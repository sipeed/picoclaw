# Financial Persona with Alpaca MCP & mcp2cli

This document outlines how PicoClaw establishes the **Alpaca MCP Foundation** and routes to a "Financial Persona".

By leveraging the `mcp2cli` tool implemented natively in Go, PicoClaw avoids large JSON schema injection, reducing token overhead by up to 99%.

## 1. Setup the Alpaca MCP Environment
Create a `.env.alpaca` file with your keys in the working directory (an example `.env.alpaca.template` is provided). Ensure you use your Paper Trading keys first to avoid live executions.

## 2. Using the mcp2cli Tool
When a user intent is detected as "Financial", the LLM will be instructed (or naturally figure out) to use the `mcp2cli` tool.
The LLM can list tools:
```bash
mcp2cli --mcp-stdio "uvx alpaca-mcp-server" --env-file .env.alpaca --list
```
It will discover tools like `get_portfolio_history`, `get_market_data`, etc. It can then call them dynamically:
```bash
mcp2cli --mcp-stdio "uvx alpaca-mcp-server" --env-file .env.alpaca get_account
```

This acts as the Semantic Gateway for Financial Personas, delegating API interactions to an authenticated Alpaca sub-shell without storing hardcoded tools.
