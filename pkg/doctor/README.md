# PicoClaw Doctor

A diagnostic tool for checking PicoClaw configuration and connectivity.

## Usage

```bash
picoclaw doctor
```

## What it checks

1. **Configuration File** - Verifies config exists and is valid JSON
2. **Workspace** - Checks workspace directory exists and is writable
3. **LLM Providers** - Lists configured providers and their API keys
4. **Provider Connectivity** - Tests network connectivity to configured providers
5. **Tools** - Lists enabled tools (Brave, DuckDuckGo, Firecrawl, SerpAPI)
6. **Channels** - Lists enabled chat channels

## Docker Usage

```bash
# Run doctor with Gemini API key
docker compose run --rm picoclaw-doctor

# Or set your API key in environment
export GEMINI_API_KEY=your-key-here
docker compose run --rm picoclaw-doctor
```

## Exit Codes

- `0` - All checks passed
- `1` - One or more errors found

## Example Output

```
🏥 PicoClaw Doctor
==================

✅ Configuration File
   Configuration loaded successfully
   Path: /root/.picoclaw/config.json

✅ Workspace
   Workspace is accessible and writable
   Path: /root/.picoclaw/workspace

✅ LLM Providers
   1 provider(s) configured
   ✓ Gemini

✅ Provider Connectivity
   All 1 tested providers reachable
   ✓ Gemini: Reachable

⚠️ Tools
   No tools enabled
   At least DuckDuckGo search is recommended

✅ Channels
   No channels enabled (CLI mode only)

----------
✅ 5 passed  ⚠️ 1 warnings  ❌ 0 errors

⚠️  PicoClaw should work, but consider addressing the warnings
```