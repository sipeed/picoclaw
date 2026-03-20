# Agent Guide - Detailed Task Patterns and Best Practices

This document provides comprehensive guidance for AI agents working with PicoClaw.

## Table of Contents

1. [Solving Tasks](#solving-tasks)
2. [Scheduled Tasks Pattern](#scheduled-tasks-pattern)
3. [Missing Functionality](#missing-functionality)
4. [Best Practices](#best-practices)

## Solving Tasks

When given a task, follow this structured approach:

### 1. Understand the Task
- Read and understand what the user is asking for
- Identify the core requirements and constraints
- Ask clarifying questions if anything is unclear

### 2. Plan the Solution
- Break down complex tasks into smaller steps
- Identify what tools and resources you'll need
- Consider potential challenges and edge cases

### 3. Execute Step by Step
- Work through each step methodically
- Explain your actions as you take them
- Validate results at each stage

### 4. Review and Refine
- Check that the solution meets all requirements
- Look for opportunities to improve
- Document any important findings

## Scheduled Tasks Pattern

For tasks that need to run on a schedule:

### 1. Identify the Schedule
- Determine the frequency (hourly, daily, weekly)
- Note any specific timing requirements
- Consider timezone implications

### 2. Set Up the Schedule
- Use cron expressions for flexible scheduling
- Example: `0 9 * * *` for daily at 9 AM
- Test the schedule to ensure it works as expected

### 3. Handle Failures Gracefully
- Implement retry logic for transient failures
- Log errors for debugging
- Alert on persistent failures

## Missing Functionality

If you encounter a task that requires functionality not currently available:

### 1. Check Available Skills
- Review the skills in the `workspace/skills/` directory
- Each skill has a SKILL.md with usage instructions
- Skills can be combined to accomplish complex tasks

### 2. Skill Discovery
- Browse available skills: `ls workspace/skills/`
- Read skill documentation: `cat workspace/skills/<skill-name>/SKILL.md`
- Look for examples in skill directories

### 3. Request New Features
- If no existing skill meets your needs
- Document the required functionality clearly
- Consider creating a custom skill for reusable logic

## Best Practices

- **Be Explicit**: Clearly state what you're doing and why
- **Stay Focused**: Keep responses relevant to the task at hand
- **Use Memory**: Store important information for future reference
- **Validate Assumptions**: Don't assume - verify when uncertain
- **Respect Constraints**: Work within the system's limitations
- **Start Simple**: Begin with minimal viable solutions, then iterate
- **Document Decisions**: Explain why certain approaches were chosen
