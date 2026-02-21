# Contribution Guidelines

Thank you for your interest in contributing to PicoClaw! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Commit Guidelines](#commit-guidelines)
- [Reporting Issues](#reporting-issues)
- [Feature Requests](#feature-requests)

## Code of Conduct

Be respectful and inclusive. We welcome contributions from everyone. Please be constructive in discussions and reviews.

## Getting Started

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/picoclaw.git
cd picoclaw

# Add upstream remote
git remote add upstream https://github.com/sipeed/picoclaw.git
```

### 2. Set Up Development Environment

```bash
# Install dependencies
make deps

# Build
make build

# Run tests
make test
```

### 3. Create a Branch

```bash
# Update main branch
git checkout main
git pull upstream main

# Create feature branch
git checkout -b feature/my-feature
```

## Development Workflow

### Make Changes

1. Make your changes in your feature branch
2. Add or update tests as needed
3. Update documentation if applicable

### Run Checks

Before submitting, run all checks:

```bash
make check
```

This runs:
- `make deps` - Download dependencies
- `make fmt` - Format code
- `make vet` - Run linter
- `make test` - Run tests

### Run Specific Tests

```bash
# Test a specific package
go test ./pkg/tools/... -v

# Run a specific test
go test ./pkg/tools/... -v -run TestMessageTool
```

## Pull Request Process

### 1. Push Changes

```bash
git push origin feature/my-feature
```

### 2. Create Pull Request

1. Go to your fork on GitHub
2. Click "Pull Request"
3. Select your feature branch
4. Fill in the PR template

### 3. PR Requirements

- [ ] All tests pass
- [ ] Code is formatted (`make fmt`)
- [ ] Linter passes (`make vet`)
- [ ] Documentation updated (if applicable)
- [ ] PR description explains the change
- [ ] Commits follow [commit guidelines](#commit-guidelines)

### 4. Code Review

- Respond to review feedback promptly
- Make requested changes in new commits
- Mark conversations as resolved

### 5. Merge

PRs are merged by maintainers after approval.

## Coding Standards

### Go Style

Follow standard Go conventions:

```bash
# Format code
make fmt

# Or manually
go fmt ./pkg/...
```

### Error Handling

Wrap errors with context:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to process message: %w", err)
}

// Bad
if err != nil {
    return err
}
```

### Logging

Use structured logging:

```go
// Good
logger.InfoCF("agent", "Processing message",
    map[string]interface{}{
        "channel": msg.Channel,
        "chat_id": msg.ChatID,
    })

// Bad
log.Printf("Processing message from %s", msg.Channel)
```

### Comments

- Write self-documenting code
- Add comments for exported functions
- Document complex logic

```go
// Execute processes the tool call with the given arguments.
// It returns a ToolResult containing the output for both the LLM and user.
func (t *MessageTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    // Implementation...
}
```

### Interfaces

Use interfaces for extensibility:

```go
// Good - interface for extension
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) *ToolResult
}

// Bad - concrete type only
type MessageTool struct { ... }
```

### Context

Always accept context for long-running operations:

```go
// Good
func (p *Provider) Chat(ctx context.Context, ...) (*Response, error)

// Bad
func (p *Provider) Chat(...) (*Response, error)
```

## Commit Guidelines

### Commit Message Format

```
<type>: <subject>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code restructuring
- `test`: Adding/updating tests
- `chore`: Maintenance tasks

### Examples

```
feat: add support for Slack channel

Add Slack channel integration using the Slack API.
Supports both public channels and direct messages.

Closes #123
```

```
fix: handle empty message content in agent loop

The agent loop was crashing when receiving empty messages.
Added validation to skip processing empty content.
```

```
docs: update installation instructions for Windows
```

### Keep Commits Atomic

- One logical change per commit
- Each commit should be able to stand alone
- Don't mix unrelated changes

## Reporting Issues

### Bug Reports

Include:

1. **Description**: Clear description of the bug
2. **Steps to Reproduce**: How to trigger the bug
3. **Expected Behavior**: What you expected to happen
4. **Actual Behavior**: What actually happened
5. **Environment**:
   - OS and version
   - Go version
   - PicoClaw version
6. **Logs**: Relevant log output (use `--debug`)

### Issue Template

~~~markdown
## Description
[Describe the bug]

## Steps to Reproduce
1. [First step]
2. [Second step]
3. [Third step]

## Expected Behavior
[What should happen]

## Actual Behavior
[What actually happens]

## Environment
- OS: [e.g., macOS 14.0]
- Go version: [e.g., 1.21]
- PicoClaw version: [e.g., v0.1.0]

## Logs
```
[paste relevant logs]
```
~~~

## Feature Requests

Include:

1. **Problem**: What problem does this solve?
2. **Solution**: Describe your proposed solution
3. **Alternatives**: Other solutions you considered
4. **Use Case**: How would this be used?

## Project Structure

When adding new features, place them in appropriate directories:

```
pkg/
├── agent/         # Core agent logic
├── bus/           # Message bus
├── channels/      # Platform integrations
│   └── slack.go   # Add new channels here
├── providers/     # LLM providers
│   └── myprovider/
│       └── provider.go  # Add new providers here
├── tools/         # Tool implementations
│   └── mytool.go  # Add new tools here
└── skills/        # Skill loading
```

## Documentation

Update documentation for:

- New features
- Changed behavior
- New configuration options
- New CLI commands

Documentation is in `docs/`:

```
docs/
├── getting-started/   # Installation, quick start
├── user-guide/        # Feature documentation
└── developer-guide/   # This documentation
```

## Questions?

- Open an issue for questions
- Check existing issues first
- Be specific and provide context

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to PicoClaw!
