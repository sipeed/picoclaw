---
name: weather
description: Get current weather and forecasts (no API key required).
homepage: https://open-meteo.com/en/docs
metadata: {"nanobot":{"emoji":"🌤️","requires":{"bins":["curl"]}}}
---

# Weather

Use coordinate-based weather lookup by default.

Important:
- Always geocode place names first. Do not query weather providers with a raw city string when the place could be ambiguous.
- This is especially important for short city names and Chinese place names such as `成都`, `上海`, `北京`, etc.
- In the final answer, mention the resolved place name plus province/state/country so the user can verify the location is correct.

## Reliable Flow

1. Geocode the user-provided location with Open-Meteo's geocoding API.
2. Pick the best match using `name`, `admin1`, `country`, and coordinates.
3. If there are multiple plausible matches, ask a clarification question instead of guessing.
4. Query weather by latitude/longitude with `timezone=auto`.
5. Optionally use wttr.in only as a quick plain-text fallback or sanity check after the location is already disambiguated.

## Open-Meteo Geocoding (Primary)

Use `curl -sG` with `--data-urlencode` so non-ASCII place names work correctly.

Example: Chinese city name
```bash
curl -sG "https://geocoding-api.open-meteo.com/v1/search" \
  --data-urlencode "name=成都" \
  --data "count=5" \
  --data "language=zh" \
  --data "format=json"
```

Example: English city name
```bash
curl -sG "https://geocoding-api.open-meteo.com/v1/search" \
  --data-urlencode "name=Chengdu" \
  --data "count=5" \
  --data "language=en" \
  --data "format=json"
```

Tips:
- Prefer exact-name matches first.
- Use `admin1` and `country` to avoid same-name city mistakes.
- If the user already gave a province/state/country, include it in your reasoning when choosing the result.

## Open-Meteo Forecast (Primary)

After choosing coordinates, query weather by latitude/longitude:

```bash
curl -sG "https://api.open-meteo.com/v1/forecast" \
  --data "latitude=30.67" \
  --data "longitude=104.07" \
  --data "current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m" \
  --data "daily=weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max" \
  --data "forecast_days=3" \
  --data "timezone=auto"
```

Response guidance:
- State the resolved location clearly, for example `Chengdu, Sichuan, China`.
- Include current temperature, wind, humidity, and a short forecast.
- If weather code interpretation is needed, translate it into plain language instead of dumping raw codes.

## wttr.in (Fallback / Quick Text Output)

wttr.in is useful for quick human-readable text, but it is less reliable for ambiguous city names.

Use it only when:
- the location is already disambiguated, or
- you need a compact plain-text summary quickly.

Preferred pattern:
```bash
curl -s "wttr.in/Chengdu,Sichuan?format=%l:+%c+%t+%h+%w&m"
```

Avoid:
```bash
curl -s "wttr.in/成都?format=3"
curl -s "wttr.in/Shanghai?format=3"
```

because short raw names can resolve to the wrong place.

## Summary

Preferred order:
1. Open-Meteo geocoding
2. Open-Meteo forecast by coordinates
3. wttr.in only after the location is already verified
