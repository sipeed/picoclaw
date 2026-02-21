# Creating Skills

This guide explains how to create skills for PicoClaw.

## Overview

Skills are reusable prompt templates that enhance the agent's capabilities. They are defined in Markdown files and can be loaded from multiple locations. Skills are a powerful way to:

- Package domain expertise
- Create reusable workflows
- Share capabilities across agents
- Customize agent behavior for specific tasks

## Skill File Format

Skills are defined in `SKILL.md` files with optional YAML frontmatter:

```markdown
---
name: code-review
description: Review code for quality, security, and best practices
---

# Code Review Skill

You are a code reviewer. When reviewing code, analyze:

## Code Quality
- Readability and maintainability
- Naming conventions
- Code organization

## Security
- Input validation
- Authentication/authorization
- Potential vulnerabilities

## Best Practices
- Design patterns
- Error handling
- Performance considerations

## Output Format

Provide your review in the following format:

### Summary
[Brief summary of the review]

### Issues Found
- [List of issues with severity]

### Recommendations
- [List of recommendations]

### Positive Aspects
- [Things done well]
```

## Skill Locations

Skills are loaded from three locations in priority order:

1. **Workspace Skills** (highest priority)
   - Location: `~/.picoclaw/workspace/skills/<skill-name>/SKILL.md`
   - Use: Project-specific skills

2. **Global Skills**
   - Location: `~/.picoclaw/skills/<skill-name>/SKILL.md`
   - Use: User-wide skills

3. **Built-in Skills** (lowest priority)
   - Location: Embedded in binary or system directory
   - Use: System-provided skills

When skills have the same name, higher-priority locations override lower ones.

## Creating a Skill

### Step 1: Create Skill Directory

```bash
mkdir -p ~/.picoclaw/skills/my-skill
```

### Step 2: Create SKILL.md

Create `~/.picoclaw/skills/my-skill/SKILL.md`:

```markdown
---
name: my-skill
description: A brief description of what this skill does
---

# My Skill

[Detailed skill instructions go here]
```

### Step 3: Use in Agent

Skills are automatically discovered. The agent will see available skills and can invoke them.

## Skill Metadata

### Name

The skill name must:
- Be alphanumeric with hyphens
- Be at most 64 characters
- Match the pattern: `^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`

```yaml
---
name: valid-skill-name
---
```

### Description

The description should:
- Be at most 1024 characters
- Explain what the skill does
- Help the LLM decide when to use it

```yaml
---
description: Analyzes Python code for potential bugs and suggests improvements
---
```

## Skill Content

The skill content (after frontmatter) is injected into the system prompt when the skill is active. Write clear, detailed instructions:

### Good Skill Content

```markdown
# Database Query Skill

You are a database expert. When helping with database queries:

## Supported Databases
- PostgreSQL
- MySQL
- SQLite

## Query Guidelines

1. **Always use parameterized queries** to prevent SQL injection
2. **Include indexes** for frequently queried columns
3. **Use transactions** for multi-step operations

## Example Queries

### Select with Join
```sql
SELECT u.name, o.total
FROM users u
JOIN orders o ON u.id = o.user_id
WHERE o.created_at > ?
```

## Output Format

When generating SQL:
1. Explain what the query does
2. Show the SQL statement
3. List any assumptions made
```

### Tips for Effective Skills

1. **Be Specific**: Provide concrete examples and guidelines
2. **Structure Content**: Use headings, lists, and code blocks
3. **Define Scope**: Clearly state what the skill does and doesn't do
4. **Include Examples**: Show expected inputs and outputs
5. **Set Boundaries**: Define limitations and edge cases

## Example Skills

### Code Generation Skill

```markdown
---
name: code-gen
description: Generate high-quality code in various programming languages
---

# Code Generation Skill

Generate clean, well-documented code following these principles:

## Code Style

### Python
- Follow PEP 8
- Use type hints
- Document with docstrings

### JavaScript/TypeScript
- Use const/let (never var)
- Prefer arrow functions
- Use JSDoc comments

### Go
- Follow gofmt style
- Handle errors explicitly
- Use meaningful variable names

## Documentation

Always include:
- Purpose of the function/module
- Parameter descriptions
- Return value description
- Example usage

## Example

```python
def calculate_total(items: list[dict]) -> float:
    """
    Calculate the total price of items.

    Args:
        items: List of item dictionaries with 'price' and 'quantity' keys.

    Returns:
        Total price as a float.

    Example:
        >>> items = [{'price': 10.0, 'quantity': 2}]
        >>> calculate_total(items)
        20.0
    """
    return sum(item['price'] * item['quantity'] for item in items)
```
```

### Writing Assistant Skill

```markdown
---
name: writing-assistant
description: Help with writing, editing, and improving text
---

# Writing Assistant Skill

You are a writing assistant. Help users improve their writing.

## Services

### Editing
- Fix grammar and spelling
- Improve clarity
- Enhance readability

### Style Suggestions
- Adjust tone (formal/casual)
- Improve flow
- Strengthen word choice

### Structure
- Organize paragraphs
- Create outlines
- Develop arguments

## Guidelines

1. Preserve the author's voice
2. Explain significant changes
3. Offer alternatives for major revisions
4. Consider the target audience

## Output Format

When editing, provide:
1. **Revised text**: The improved version
2. **Changes made**: Brief explanation of edits
3. **Suggestions**: Optional further improvements
```

### Data Analysis Skill

```markdown
---
name: data-analysis
description: Analyze datasets and provide insights
---

# Data Analysis Skill

You are a data analyst. Help users understand their data.

## Capabilities

### Descriptive Statistics
- Mean, median, mode
- Standard deviation
- Distribution analysis

### Pattern Recognition
- Trends over time
- Correlations
- Anomalies

### Visualization Suggestions
- Chart types
- Color schemes
- Labeling best practices

## Analysis Process

1. **Understand the Data**
   - What does each column represent?
   - What is the time range?
   - Are there missing values?

2. **Clean the Data**
   - Handle nulls
   - Remove duplicates
   - Fix inconsistencies

3. **Analyze**
   - Calculate statistics
   - Find patterns
   - Identify insights

4. **Report**
   - Summarize findings
   - Highlight key insights
   - Recommend actions

## Output Format

### Executive Summary
[2-3 sentence overview]

### Key Findings
- [Finding 1]
- [Finding 2]
- [Finding 3]

### Detailed Analysis
[In-depth analysis]

### Recommendations
- [Recommendation 1]
- [Recommendation 2]
```

## Frontmatter Formats

PicoClaw supports both JSON and YAML frontmatter:

### YAML (Recommended)

```markdown
---
name: my-skill
description: Skill description
---

Content here...
```

### JSON

```markdown
---
{
  "name": "my-skill",
  "description": "Skill description"
}
---

Content here...
```

## Loading Skills

Skills are loaded by the `SkillsLoader`:

```go
loader := skills.NewSkillsLoader(
    workspace,           // Workspace directory
    globalSkillsPath,    // ~/.picoclaw/skills
    builtinSkillsPath,   // Built-in skills directory
)

// List all available skills
allSkills := loader.ListSkills()

// Load a specific skill
content, found := loader.LoadSkill("my-skill")

// Build summary for context
summary := loader.BuildSkillsSummary()
```

## Testing Skills

Test your skills by:

1. **Creating the skill file** in the appropriate directory
2. **Starting PicoClaw** with debug mode
3. **Asking the agent** about available skills
4. **Invoking the skill** with a relevant request

```bash
# Start with debug
./build/picoclaw agent --debug

# Ask about skills
> What skills do you have available?

> Use the code-review skill to review this function:
> def add(a, b): return a + b
```

## Best Practices

1. **Single Purpose**: Each skill should do one thing well
2. **Clear Description**: Help the LLM know when to use it
3. **Comprehensive Content**: Include all necessary context
4. **Examples**: Show expected inputs and outputs
5. **Limitations**: State what the skill cannot do
6. **Regular Updates**: Keep skills current with requirements

## Sharing Skills

Skills can be shared by:

1. **Git Repository**: Store skills in a repo
2. **Copy to Workspace**: Copy skill directories
3. **Package Distribution**: Create skill packages

Example structure for a skill repository:

```
my-skills/
├── code-review/
│   └── SKILL.md
├── data-analysis/
│   └── SKILL.md
└── writing-assistant/
    └── SKILL.md
```

Users can clone and link:

```bash
git clone https://github.com/user/my-skills.git
ln -s my-skills/* ~/.picoclaw/skills/
```

## See Also

- [User Guide: Using Skills](../../user-guide/skills/using-skills.md)
- [User Guide: Creating Skills](../../user-guide/skills/creating-skills.md)
- [Skills Loader Source](https://github.com/sipeed/picoclaw/tree/main/pkg/skills)
