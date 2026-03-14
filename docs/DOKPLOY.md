# Dokploy Deployment

This repository includes a Dokploy-ready template under [docker/dokploy](../docker/dokploy).

## Included Files

- `docker/dokploy/docker-compose.yml`: runtime service definition
- `docker/dokploy/template.toml`: Dokploy template variables, domain mapping, and config mount

## What This Template Provides

- A persistent workspace volume at `/root/.picoclaw/workspace`
- Automatic gateway host binding to `0.0.0.0`
- A generated `config.json` mount with:
  - model alias: `openrouter-default`
  - model: `openrouter/openai/gpt-4o-mini`
  - OpenRouter API key from template variable

## Usage Notes

- Set your domain and `openrouter_api_key` in Dokploy when deploying the template.
- If you prefer another provider/model, edit the mounted `config.json` content in `template.toml`.
