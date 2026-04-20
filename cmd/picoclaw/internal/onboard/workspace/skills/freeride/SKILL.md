# FreeRide Skill

FreeRide gives you unlimited free AI in PicoClaw by automatically managing OpenRouter's free models.

## Usage

- `/freeride auto`: Auto-configure best model + fallbacks.
- `/freeride list`: See all 30+ free models ranked.
- `/freeride status`: Check your current setup.
- `/freeride timeout 120`: Set request timeout for free models (seconds).

## How it works

The skill uses the `freeride` tool to fetch free models from OpenRouter, ranks them by context length, capabilities, recency, and provider trust, and then updates your PicoClaw configuration with the best models as fallbacks.

## Setup

Ensure you have your OpenRouter API key set in your K3s secrets or environment variables as `OPENROUTER_API_KEY`.
