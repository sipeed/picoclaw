---
name: weather
description: Get current weather and forecasts with verified location matching (no API key required).
homepage: https://wttr.in/:help
metadata: {"nanobot":{"emoji":"🌤️","requires":{"bins":["curl"]}}}
---

# Weather

Use the most reliable location match first. For Chinese city names or other non-Latin input, prefer `wttr.in` with the original query because it resolves native names directly. Use Open-Meteo for structured current conditions and forecasts only after you have confirmed the exact city.

## Accuracy Rules

- Always restate the matched location, region/country, and observation time in the final answer.
- Do not trust the first geocoding hit blindly. Check `country`, `admin1`, `admin2`, and `population`.
- For Chinese city queries, do not send Hanzi directly to Open-Meteo geocoding unless the top result is obviously correct. Prefer `wttr.in` with the original Chinese name, or geocode the English/pinyin city name instead.
- If multiple plausible matches remain, ask a follow-up question or state the assumption clearly.
- Use `timezone=auto` when calling Open-Meteo so the reported time matches the location.

## wttr.in (best for direct city-name queries)

Quick current conditions:
```bash
curl -s "https://wttr.in/London?format=%l:+%c+%t+%h+%w"
```

Chinese city example:
```bash
curl -s "https://wttr.in/%E6%88%90%E9%83%BD?format=%l:+%c+%t+%h+%w"
curl -s "https://wttr.in/%E4%B8%8A%E6%B5%B7?format=%l:+%c+%t+%h+%w"
```

JSON output if you need more detail:
```bash
curl -s "https://wttr.in/Chengdu?format=j1"
```

Tips:
- URL-encode spaces: `New York` -> `New+York`
- URL-encode non-ASCII text before sending the request
- Use `?m` for metric units and `?u` for US units

## Open-Meteo (best for structured forecasts)

1. Geocode the city and verify the returned location metadata:
```bash
curl -s "https://geocoding-api.open-meteo.com/v1/search?name=Chengdu&count=3&language=en&format=json"
```

2. Query current weather and today's forecast with the verified coordinates:
```bash
curl -s "https://api.open-meteo.com/v1/forecast?latitude=30.66667&longitude=104.06667&current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m&daily=weather_code,temperature_2m_max,temperature_2m_min&forecast_days=1&timezone=auto"
```

Important:
- For Chinese inputs like `成都`, geocoding `name=%E6%88%90%E9%83%BD` may return smaller homonym locations first. Prefer `Chengdu` after verifying it matches Sichuan, China.
- If geocoding looks suspicious, fall back to `wttr.in` for the original city name instead of presenting a likely wrong result.

Docs: https://open-meteo.com/en/docs
