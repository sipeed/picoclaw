# Builtin Skills

PicoClaw includes builtin skills that are always available. These cover common tasks and don't require installation.

## Available Builtin Skills

| Skill | Description | Dependencies |
|-------|-------------|--------------|
| `weather` | Get weather information | `curl` |
| `news` | Fetch news headlines | `curl` |
| `stock` | Check stock prices | `curl` |
| `calculator` | Perform calculations | None |

## Weather Skill

Get current weather and forecasts without any API keys.

### Features

- Current conditions for any location
- Multi-day forecasts
- No API key required (uses wttr.in)
- Supports city names, airport codes, and coordinates

### Examples

**Current weather:**
```
User: "What's the weather in London?"

Agent: London: Partly cloudy, +15C
       Humidity: 68%, Wind: 12 km/h
```

**Compact format:**
```
User: "Quick weather check for Tokyo"

Agent: Tokyo: Sunny +22C
```

**Full forecast:**
```
User: "Show me the 3-day forecast for New York"

Agent: Weather forecast for New York:

       Today          Tomorrow        Day After
       +18C/+12C     +20C/+14C       +17C/+11C
       Partly cloudy  Sunny           Rain
       ...
```

**Airport code:**
```
User: "Weather at JFK airport"

Agent: JFK (New York): Clear, +20C
```

### Technical Details

The weather skill uses:
- **wttr.in** - Primary service, text-based weather
- **Open-Meteo** - Fallback for JSON responses

Format options:
- `?format=3` - One-line output
- `?T` - Full terminal forecast
- `?1` - Today only
- `?0` - Current conditions only

## News Skill

Fetch the latest news headlines from various sources.

### Features

- Top headlines
- Category filtering (tech, business, etc.)
- No API key required
- Headline summaries

### Examples

**Top headlines:**
```
User: "Show me today's news"

Agent: Top Headlines:

       1. Tech Giants Report Record Q3 Earnings
          Major technology companies exceeded analyst expectations...

       2. Climate Summit Reaches Historic Agreement
          World leaders commit to 50% emissions reduction...

       3. New AI Breakthrough in Medical Diagnosis
          Researchers announce 99% accuracy in early detection...
```

**Tech news:**
```
User: "What's new in tech?"

Agent: Technology Headlines:

       1. Apple Announces New Product Line
       2. OpenAI Releases GPT-5 Preview
       3. Quantum Computing Milestone Achieved
       ...
```

## Stock Skill

Check stock prices and market data.

### Features

- Real-time stock prices
- Market indices
- No API key required
- Simple price queries

### Examples

**Stock price:**
```
User: "What's Apple's stock price?"

Agent: AAPL (Apple Inc.): $178.50
       Change: +2.30 (+1.31%)
       Volume: 52.3M
```

**Market indices:**
```
User: "How's the market doing today?"

Agent: Market Summary:

       Dow Jones:    38,500 (+0.5%)
       S&P 500:      5,200 (+0.8%)
       NASDAQ:       16,400 (+1.2%)
```

**Multiple stocks:**
```
User: "Check prices for AAPL, GOOGL, and MSFT"

Agent: Stock Prices:

       AAPL:  $178.50 (+1.31%)
       GOOGL: $141.25 (+0.85%)
       MSFT:  $378.90 (+0.92%)
```

## Calculator Skill

Perform mathematical calculations.

### Features

- Basic arithmetic
- Percentages
- Unit conversions
- Compound interest
- No external dependencies

### Examples

**Basic math:**
```
User: "What's 15% of 847?"

Agent: 15% of 847 is 127.05
```

**Compound interest:**
```
User: "Calculate compound interest on $10,000 at 5% for 3 years, compounded monthly"

Agent: Compound Interest Calculation:

       Principal:     $10,000.00
       Rate:          5% annual
       Period:        3 years
       Compounding:   Monthly

       Final Amount:  $11,614.72
       Interest Earned: $1,614.72
```

**Unit conversion:**
```
User: "Convert 100 miles to kilometers"

Agent: 100 miles = 160.934 kilometers
```

**Expression evaluation:**
```
User: "Calculate (25 * 4) + (100 / 5) - 15"

Agent: (25 * 4) + (100 / 5) - 15 = 105
```

## Installing Builtin Skills

To customize builtin skills, copy them to your workspace:

```bash
picoclaw skills install-builtin
```

This copies all builtin skills to `~/.picoclaw/workspace/skills/`, where you can edit them.

### Listing Builtin Skills

View available builtin skills:

```bash
picoclaw skills list-builtin
```

Output:
```
Available Builtin Skills:
-----------------------
  weather
     Get current weather and forecasts
  news
     Get latest news headlines
  stock
     Check stock prices and market data
  calculator
     Perform mathematical calculations
```

## Customizing Builtin Skills

After installing builtin skills, you can modify them:

```bash
# Install to workspace
picoclaw skills install-builtin

# Edit the weather skill
vim ~/.picoclaw/workspace/skills/weather/SKILL.md

# Your modified version takes precedence
picoclaw skills show weather
```

Common customizations:
- Add local city shortcuts
- Change default format options
- Add new API endpoints
- Include location-specific settings

## Skill Dependencies

Some builtin skills require external tools:

| Skill | Required Tools |
|-------|---------------|
| weather | `curl` |
| news | `curl` |
| stock | `curl` |
| calculator | None (built-in) |

Install dependencies:
```bash
# macOS
brew install curl

# Ubuntu/Debian
sudo apt install curl
```

## See Also

- [Skills Overview](README.md)
- [Using Skills](using-skills.md)
- [Installing Skills](installing-skills.md)
- [Creating Skills](creating-skills.md)
