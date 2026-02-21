# Web Tools

Search the web and fetch web page content.

## Tools

### web_search

Search the web for information.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Search query |

**Example:**

```json
{
  "query": "latest AI news 2025"
}
```

**Providers (in order of preference):**

1. Perplexity (if configured)
2. Brave Search (if configured)
3. DuckDuckGo (always available)

### web_fetch

Fetch and extract content from a web page.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `url` | string | Yes | URL to fetch |
| `format` | string | No | Output format: `text` or `json` |

**Example:**

```json
{
  "url": "https://example.com/article",
  "format": "text"
}
```

## Configuration

### Brave Search (Recommended)

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      }
    }
  }
}
```

Get a free API key at [brave.com/search/api](https://brave.com/search/api) (2000 queries/month).

### DuckDuckGo (Default)

No API key required, always available:

```json
{
  "tools": {
    "web": {
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

### Perplexity

```json
{
  "tools": {
    "web": {
      "perplexity": {
        "enabled": true,
        "api_key": "pplx-xxx",
        "max_results": 5
      }
    }
  }
}
```

## Usage Examples

### Search

```
User: "Search for the weather in Tokyo"

Agent uses web_search:
{
  "query": "weather Tokyo today"
}

Agent: "According to search results, Tokyo is currently 15°C with partly cloudy skies..."
```

### Fetch Page

```
User: "Summarize this article: https://example.com/article"

Agent uses web_fetch:
{
  "url": "https://example.com/article"
}

Agent: "The article discusses... [summary of content]"
```

### Combined Search and Fetch

```
User: "Find information about the latest iPhone and summarize"

Agent uses web_search, then web_fetch:
{
  "query": "latest iPhone 2025"
}

Then fetches top results for details.

Agent: "The latest iPhone features... [comprehensive summary]"
```

## Troubleshooting

### "API configuration error"

This means no search API key is configured. DuckDuckGo should work automatically.

To enable Brave Search:
1. Get API key from [brave.com/search/api](https://brave.com/search/api)
2. Add to config.json

### Rate Limiting

If you hit rate limits:
1. Reduce `max_results`
2. Use fallback providers
3. Wait and retry

## See Also

- [Tools Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)
