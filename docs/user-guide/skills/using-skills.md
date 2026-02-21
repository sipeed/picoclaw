# Using Skills

Learn how to effectively use installed skills in PicoClaw.

## How Skills Work

When you interact with PicoClaw, it automatically loads relevant skills based on your request. The skill content is added to the agent's context, providing it with specialized knowledge and command patterns.

## Automatic Skill Activation

Skills are activated automatically based on context. PicoClaw analyzes your request and loads skills that might be relevant:

```
User: "What's the weather like in San Francisco?"

Agent: [Loads weather skill, executes curl command]
       The current weather in San Francisco is partly cloudy
       with a temperature of 18C (64F). Humidity is at 72%.
```

## Viewing Available Skills

List all installed skills to see what capabilities are available:

```bash
picoclaw skills list
```

Output:
```
Installed Skills:
------------------
  weather (workspace)
    Get current weather and forecasts (no API key required)
  github (global)
    Interact with GitHub using the gh CLI
  calculator (builtin)
    Perform mathematical calculations
```

## Viewing Skill Details

To see the full content of a skill:

```bash
picoclaw skills show weather
```

This displays the skill's instructions, commands, and examples that the agent uses.

## Skill Examples

### Weather Skill

The weather skill uses wttr.in for weather data without requiring an API key.

```
User: "Check the weather in London"

Agent: [Uses weather skill]
       London: Partly cloudy, +15C
       Humidity: 68%, Wind: 12 km/h NW
```

```
User: "Get a 3-day forecast for Tokyo"

Agent: [Uses weather skill]
       Here's the forecast for Tokyo...

       Morning    Noon    Evening
       +12C      +18C    +14C
       Rain      Cloudy  Clear
       ...
```

### GitHub Skill

The GitHub skill provides GitHub CLI integration.

```
User: "List open issues in my repo"

Agent: [Uses github skill]
       Here are the open issues:

       #142 - Fix authentication bug
       #141 - Add dark mode support
       #140 - Update documentation
```

```
User: "Check the CI status of PR 55"

Agent: [Uses github skill]
       PR #55 CI Status: All checks passed
       - build: passed
       - test: passed
       - lint: passed
```

### Calculator Skill

The calculator skill performs mathematical calculations.

```
User: "What's 15% of 847?"

Agent: [Uses calculator skill]
       15% of 847 is 127.05
```

```
User: "Calculate the compound interest on $10,000 at 5% for 3 years"

Agent: [Uses calculator skill]
       After 3 years at 5% annual interest:
       Final amount: $11,576.25
       Total interest earned: $1,576.25
```

### News Skill

The news skill fetches current headlines.

```
User: "Show me today's top headlines"

Agent: [Uses news skill]
       Top Headlines:

       1. Tech Giants Report Q3 Earnings
       2. Climate Summit Reaches Agreement
       3. New AI Breakthrough Announced
       ...
```

## Explicit Skill Requests

You can explicitly mention a skill to ensure it's used:

```
User: "Use the github skill to create a new issue about the login bug"

Agent: [Uses github skill, creates issue]
       Created issue #143: "Login bug on mobile devices"
       https://github.com/owner/repo/issues/143
```

## Skill Context in Conversations

Skills remain active throughout a conversation session. If you start talking about weather, the weather skill stays in context for follow-up questions:

```
User: "What's the weather in Paris?"
Agent: Paris: Sunny, +22C

User: "And in Berlin?"
Agent: [Weather skill still active]
       Berlin: Cloudy, +18C

User: "Which is warmer?"
Agent: Paris is warmer at +22C compared to Berlin's +18C.
```

## Tips for Using Skills

1. **Be specific** - Clear requests help the agent select the right skill
2. **Check skill content** - Use `picoclaw skills show <name>` to see what a skill can do
3. **Combine skills** - Complex tasks may use multiple skills together
4. **Customize** - Copy builtin skills to your workspace to modify them

## Troubleshooting

### Skill Not Activating

If a skill isn't being used:

1. Verify it's installed: `picoclaw skills list`
2. Check the skill content: `picoclaw skills show <name>`
3. Try being more explicit: "Use the weather skill to..."

### Skill Commands Failing

Some skills require external tools:

1. Check dependencies in the skill description
2. Install required tools (e.g., `gh` CLI for GitHub skill)
3. Ensure tools are in your PATH

### Outdated Skill

To update a skill:

```bash
picoclaw skills remove <name>
picoclaw skills install <repo>
```

## See Also

- [Skills Overview](README.md)
- [Installing Skills](installing-skills.md)
- [Builtin Skills](builtin-skills.md)
- [Creating Skills](creating-skills.md)
