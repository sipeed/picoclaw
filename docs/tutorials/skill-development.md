# Skill Development Tutorial

This tutorial guides you through creating custom skills for PicoClaw.

## Prerequisites

- 30 minutes
- PicoClaw installed and configured
- Basic understanding of PicoClaw tools
- Text editor for creating files

## Overview

Skills are reusable packages that extend PicoClaw capabilities:

```
┌─────────────────────────────────────────┐
│                Skill                     │
├─────────────────────────────────────────┤
│  skill.md     - Behavior prompts         │
│  tools.json   - Custom tool definitions  │
│  templates/   - File templates           │
│  examples/    - Usage examples           │
└─────────────────────────────────────────┘
```

## Part 1: Understanding Skills

### What Skills Can Do

- Define specialized behavior prompts
- Provide domain-specific knowledge
- Include templates and examples
- Configure tool usage

### Skill Location

Skills are stored in the workspace:

```
~/.picoclaw/workspace/skills/
├── built-in/           # System skills
├── custom/             # Your custom skills
└── installed/          # Installed from external sources
```

## Part 2: Create Your First Skill

### Example: Meeting Notes Skill

Create a skill for managing meeting notes.

#### Create Skill Directory

```bash
mkdir -p ~/.picoclaw/workspace/skills/meeting-notes
cd ~/.picoclaw/workspace/skills/meeting-notes
```

#### Create skill.md

```bash
nano skill.md
```

```markdown
# Meeting Notes Skill

## Purpose
Help users capture, organize, and search meeting notes.

## Behavior
When helping with meeting notes:
1. Use consistent formatting
2. Extract action items automatically
3. Identify attendees and decisions
4. Store notes with timestamps

## Note Format
```markdown
# Meeting: [Title]
Date: [Date]
Attendees: [List]

## Agenda
- [Item 1]
- [Item 2]

## Discussion
[Key points]

## Decisions
- [Decision 1]
- [Decision 2]

## Action Items
- [ ] [Task] - [Owner] - [Due Date]
```

## Commands
- "New meeting note" - Create a new note
- "Add action item" - Add to current meeting
- "List meetings" - Show recent meetings
- "Search notes" - Search through all notes
```

#### Test the Skill

```bash
picoclaw agent -m "Use the meeting-notes skill to create a new meeting note about project planning"
```

## Part 3: Add Custom Tools

### Create tools.json

Some skills benefit from custom tool definitions:

```bash
nano tools.json
```

```json
{
  "tools": [
    {
      "name": "create_meeting",
      "description": "Create a new meeting note file",
      "parameters": {
        "type": "object",
        "properties": {
          "title": {
            "type": "string",
            "description": "Meeting title"
          },
          "attendees": {
            "type": "array",
            "items": {"type": "string"},
            "description": "List of attendee names"
          },
          "date": {
            "type": "string",
            "description": "Meeting date (YYYY-MM-DD)"
          }
        },
        "required": ["title"]
      },
      "implementation": {
        "type": "write_file",
        "path_template": "meetings/{{date}}-{{slugify title}}.md",
        "content_template": "# Meeting: {{title}}\nDate: {{date}}\nAttendees: {{join attendees ', '}}\n\n## Agenda\n\n## Discussion\n\n## Decisions\n\n## Action Items\n"
      }
    }
  ]
}
```

## Part 4: Add Templates

### Create Templates Directory

```bash
mkdir -p templates
nano templates/meeting.md
```

```markdown
# Meeting: {{title}}
Date: {{date}}
Time: {{time}}
Attendees: {{attendees}}

## Agenda
{{#each agenda_items}}
- {{this}}
{{/each}}

## Discussion
{{discussion}}

## Decisions
{{#each decisions}}
- {{this}}
{{/each}}

## Action Items
{{#each action_items}}
- [ ] {{this.task}} - {{this.owner}} - {{this.due}}
{{/each}}

---
Created with PicoClaw Meeting Notes Skill
```

## Part 5: Add Examples

### Create Examples

```bash
mkdir -p examples
nano examples/basic-usage.md
```

```markdown
# Meeting Notes Examples

## Example 1: Create a Meeting Note

User: "Create a meeting note for today's standup with Alice, Bob, and Carol"

Agent creates: meetings/2024-01-15-standup.md

## Example 2: Add Action Items

User: "Add action item: Bob will finish the API by Friday"

Agent appends to current meeting:
- [ ] Finish the API - Bob - Friday

## Example 3: Search Notes

User: "Search meeting notes for 'budget'"

Agent searches all meeting files and returns matches.
```

## Part 6: Skill Manifest

Create a manifest for sharing:

```bash
nano skill.json
```

```json
{
  "name": "meeting-notes",
  "version": "1.0.0",
  "description": "Capture and organize meeting notes",
  "author": "Your Name",
  "tags": ["productivity", "notes", "meetings"],
  "min_picoclaw_version": "1.0.0",
  "files": [
    "skill.md",
    "tools.json",
    "templates/meeting.md",
    "examples/basic-usage.md"
  ]
}
```

## Part 7: More Skill Examples

### Example: Code Review Skill

```markdown
# Code Review Skill

## Purpose
Provide thorough code reviews with actionable feedback.

## Review Checklist
1. **Functionality**: Does the code do what it's supposed to?
2. **Readability**: Is the code easy to understand?
3. **Performance**: Are there obvious bottlenecks?
4. **Security**: Are there potential vulnerabilities?
5. **Testing**: Is there adequate test coverage?

## Output Format
```markdown
## Code Review

### Summary
[Brief overall impression]

### Strengths
- [What's done well]

### Issues
#### Critical
- [Must fix]

#### Suggestions
- [Nice to have]

### Questions
- [Clarification needed]
```

## Commands
- "Review this code: [paste code]"
- "Review file [path]"
- "Check for security issues in [path]"
```

### Example: Documentation Skill

```markdown
# Documentation Skill

## Purpose
Generate and maintain technical documentation.

## Capabilities
- Generate API documentation
- Create README files
- Write user guides
- Document code comments

## Documentation Style
- Clear and concise
- Include code examples
- Use proper markdown formatting
- Add diagrams where helpful

## Templates Available
- README.md
- API_REFERENCE.md
- CONTRIBUTING.md
- CHANGELOG.md
```

### Example: DevOps Skill

```markdown
# DevOps Skill

## Purpose
Help with deployment, CI/CD, and infrastructure.

## Capabilities
- Docker configuration
- CI/CD pipeline setup
- Kubernetes manifests
- Infrastructure as code

## Best Practices
- Use version control
- Implement proper logging
- Set up monitoring
- Configure alerts
- Use secrets management

## Common Tasks
- "Create a Dockerfile for [app]"
- "Set up GitHub Actions for [project]"
- "Create Kubernetes deployment for [service]"
```

## Part 8: Using Skills

### List Available Skills

```bash
picoclaw skills list
```

### Use a Skill

```bash
# In conversation
picoclaw agent -m "Use the meeting-notes skill to create a note"

# Or load skill explicitly
picoclaw agent --skill meeting-notes -m "Create a new meeting note"
```

### Skill Auto-Loading

Skills in the workspace are automatically available:

```
User: "Create a meeting note for my standup"

Agent recognizes the meeting-notes skill context and uses it.
```

## Part 9: Sharing Skills

### Package for Distribution

```bash
# Create archive
cd ~/.picoclaw/workspace/skills
tar -czvf meeting-notes-skill.tar.gz meeting-notes/
```

### Install from Archive

```bash
picoclaw skills install meeting-notes-skill.tar.gz
```

### Install from URL

```bash
picoclaw skills install https://example.com/skills/meeting-notes.tar.gz
```

### Install from GitHub

```bash
picoclaw skills install github:user/picoclaw-skills/meeting-notes
```

## Part 10: Testing Skills

### Manual Testing

```bash
# Test basic functionality
picoclaw agent -m "Use meeting-notes to create a test meeting"

# Verify file creation
ls ~/.picoclaw/workspace/meetings/

# Test edge cases
picoclaw agent -m "Create a meeting note with no attendees"
```

### Debug Mode

```bash
picoclaw agent --debug -m "Use meeting-notes skill"
```

## Best Practices

### 1. Clear Purpose

Each skill should have one clear purpose:

```
Good: "Create and manage meeting notes"
Bad: "Handle meetings, emails, and documents"
```

### 2. Specific Instructions

Give explicit instructions in skill.md:

```markdown
## Behavior
1. Always ask for missing required fields
2. Use ISO date format (YYYY-MM-DD)
3. Create backups before modifying
```

### 3. Error Handling

Include error handling guidance:

```markdown
## Error Handling
- If file exists, prompt to overwrite or append
- If template is missing, use default format
- If date is invalid, use current date
```

### 4. Examples

Provide usage examples:

```markdown
## Examples

### Creating a Meeting
User: "New meeting: Project kickoff with John and Jane"
Creates: meetings/2024-01-15-project-kickoff.md

### Adding Actions
User: "Add action: John to prepare slides by Wednesday"
Appends action item to current meeting
```

## Troubleshooting

### Skill Not Loading

1. Check file permissions
2. Verify skill.md syntax
3. Check workspace path

### Unexpected Behavior

1. Review skill.md instructions
2. Check for conflicts with other skills
3. Use debug mode

### Tools Not Working

1. Verify tools.json syntax
2. Check parameter definitions
3. Test tool calls in debug mode

## Next Steps

- [Skills Documentation](../user-guide/skills/README.md)
- [Built-in Skills](../user-guide/skills/builtin-skills.md)
- [Creating Skills Guide](../user-guide/skills/creating-skills.md)

## Summary

You learned:
- What skills are and how they work
- How to create skill.md behavior files
- How to add custom tools
- How to use templates
- How to package and share skills
- Best practices for skill development

You can now extend PicoClaw with custom skills!
