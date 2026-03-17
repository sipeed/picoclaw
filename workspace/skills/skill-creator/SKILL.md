---
name: skill-creator
description: Create or update PicoClaw skills. Use when designing, structuring, or documenting skills with SKILL.md plus optional scripts, references, and assets.
---

# Skill Creator

This skill provides guidance for creating effective skills.

## About Skills

Skills are modular, self-contained packages that extend the agent's capabilities by providing
specialized knowledge, workflows, and tools. Think of them as "onboarding guides" for specific
domains or tasks—they transform the agent from a general-purpose agent into a specialized agent
equipped with procedural knowledge that no model can fully possess.

### What Skills Provide

1. Specialized workflows - Multi-step procedures for specific domains
2. Tool integrations - Instructions for working with specific file formats or APIs
3. Domain expertise - Company-specific knowledge, schemas, business logic
4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks

## Core Principles

### Concise is Key

The context window is a public good. Skills share the context window with everything else the agent needs: system prompt, conversation history, other Skills' metadata, and the actual user request.

**Default assumption: the agent is already very smart.** Only add context the agent doesn't already have. Challenge each piece of information: "Does the agent really need this explanation?" and "Does this paragraph justify its token cost?"

Prefer concise examples over verbose explanations.

### Set Appropriate Degrees of Freedom

Match the level of specificity to the task's fragility and variability:

**High freedom (text-based instructions)**: Use when multiple approaches are valid, decisions depend on context, or heuristics guide the approach.

**Medium freedom (pseudocode or scripts with parameters)**: Use when a preferred pattern exists, some variation is acceptable, or configuration affects behavior.

**Low freedom (specific scripts, few parameters)**: Use when operations are fragile and error-prone, consistency is critical, or a specific sequence must be followed.

Think of the agent as exploring a path: a narrow bridge with cliffs needs specific guardrails (low freedom), while an open field allows many routes (high freedom).

### Anatomy of a Skill

Every skill consists of a required SKILL.md file and optional bundled resources:

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter metadata (required)
│   │   ├── name: (required)
│   │   └── description: (required)
│   └── Markdown instructions (required)
└── Bundled Resources (optional)
    ├── scripts/          - Executable code (Python/Bash/etc.)
    ├── references/       - Documentation intended to be loaded into context as needed
    └── assets/           - Files used in output (templates, icons, fonts, etc.)
```

#### SKILL.md (required)

Every SKILL.md consists of:

- **Frontmatter** (YAML): Contains `name` and `description` fields. These are the only fields that the agent reads to determine when the skill gets used, thus it is very important to be clear and comprehensive in describing what the skill is, and when it should be used.
- **Body** (Markdown): Instructions and guidance for using the skill. Only loaded AFTER the skill triggers (if at all).

#### Bundled Resources (optional)

##### Scripts (`scripts/`)

Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.

- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed
- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks
- **Benefits**: Token efficient, deterministic, may be executed without loading into context
- **Note**: Scripts may still need to be read by the agent for patching or environment-specific adjustments

##### References (`references/`)

Documentation and reference material intended to be loaded as needed into context to inform the agent's process and thinking.

- **When to include**: For documentation that the agent should reference while working
- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications
- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides
- **Benefits**: Keeps SKILL.md lean, loaded only when the agent determines it's needed
- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md
- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.

##### Assets (`assets/`)

Files not intended to be loaded into context, but rather used within the output the agent produces.

- **When to include**: When the skill needs files that will be used in the final output
- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography
- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified
- **Benefits**: Separates output resources from documentation, enables the agent to use files without loading them into context

#### What to Not Include in a Skill

A skill should only contain essential files that directly support its functionality. Do NOT create extraneous documentation or auxiliary files, including:

- README.md
- INSTALLATION_GUIDE.md
- QUICK_REFERENCE.md
- CHANGELOG.md
- etc.

The skill should only contain the information needed for an AI agent to do the job at hand. It should not contain auxiliary context about the process that went into creating it, setup and testing procedures, user-facing documentation, etc. Creating additional documentation files just adds clutter and confusion.

### Progressive Disclosure Design Principle

Skills use a three-level loading system to manage context efficiently:

1. **Metadata (name + description)** - Always in context (~100 words)
2. **SKILL.md body** - When skill triggers (<5k words)
3. **Bundled resources** - As needed by the agent (Unlimited because scripts can be executed without reading into context window)

#### Progressive Disclosure Patterns

Keep SKILL.md body to the essentials and under 500 lines to minimize context bloat. Split content into separate files when approaching this limit. When splitting out content into other files, it is very important to reference them from SKILL.md and describe clearly when to read them, to ensure the reader of the skill knows they exist and when to use them.

**Key principle:** When a skill supports multiple variations, frameworks, or options, keep only the core workflow and selection guidance in SKILL.md. Move variant-specific details (patterns, examples, configuration) into separate reference files.

**Pattern 1: High-level guide with references**

```markdown
# PDF Processing

## Quick start

Extract text with pdfplumber:
[code example]

## Advanced features

- **Form filling**: See [FORMS.md](FORMS.md) for complete guide
- **API reference**: See [REFERENCE.md](REFERENCE.md) for all methods
- **Examples**: See [EXAMPLES.md](EXAMPLES.md) for common patterns
```

the agent loads FORMS.md, REFERENCE.md, or EXAMPLES.md only when needed.

**Pattern 2: Domain-specific organization**

For Skills with multiple domains, organize content by domain to avoid loading irrelevant context:

```
bigquery-skill/
├── SKILL.md (overview and navigation)
└── reference/
    ├── finance.md (revenue, billing metrics)
    ├── sales.md (opportunities, pipeline)
    ├── product.md (API usage, features)
    └── marketing.md (campaigns, attribution)
```

When a user asks about sales metrics, the agent only reads sales.md.

Similarly, for skills supporting multiple frameworks or variants, organize by variant:

```
cloud-deploy/
├── SKILL.md (workflow + provider selection)
└── references/
    ├── aws.md (AWS deployment patterns)
    ├── gcp.md (GCP deployment patterns)
    └── azure.md (Azure deployment patterns)
```

When the user chooses AWS, the agent only reads aws.md.

**Pattern 3: Conditional details**

Show basic content, link to advanced content:

```markdown
# DOCX Processing

## Creating documents

Use docx-js for new documents. See [DOCX-JS.md](DOCX-JS.md).

## Editing documents

For simple edits, modify the XML directly.

**For tracked changes**: See [REDLINING.md](REDLINING.md)
**For OOXML details**: See [OOXML.md](OOXML.md)
```

the agent reads REDLINING.md or OOXML.md only when the user needs those features.

**Important guidelines:**

- **Avoid deeply nested references** - Keep references one level deep from SKILL.md. All reference files should link directly from SKILL.md.
- **Structure longer reference files** - For files longer than 100 lines, include a table of contents at the top so the agent can see the full scope when previewing.

## Skill Creation Process

Create or update a skill with this workflow:

1. Understand the intended examples and trigger phrases
2. Plan which reusable resources belong in `scripts/`, `references/`, or `assets/`
3. Initialize the folder structure manually
4. Write `SKILL.md` and any bundled resources
5. Validate the skill inside PicoClaw
6. Iterate based on real usage

Follow the steps in order unless you are only making a narrow edit to an existing skill.

### Skill Naming

- Use lowercase letters, digits, and hyphens only; normalize user-provided titles to hyphen-case (for example, "Plan Mode" -> `plan-mode`).
- Keep the name under 64 characters.
- Prefer short, action-oriented names.
- Namespace by tool when that improves triggering clarity (for example, `gh-address-comments`).
- Make the folder name exactly match the skill name.

### Step 1: Understand the Skill with Concrete Examples

Gather a few concrete prompts the skill should handle before writing anything. Use those examples to decide:

- What users will ask that should trigger the skill
- Which parts need deterministic scripts versus simple instructions
- Which details are stable enough to store as reusable references

Avoid asking every possible question up front. Start with the most important examples and refine only if the trigger boundary is still unclear.

### Step 2: Plan the Reusable Contents

For each example, decide what should be reusable:

- `scripts/` when the same code would otherwise be rewritten or must be deterministic
- `references/` when the agent needs documentation, schemas, policies, or long examples
- `assets/` when the skill needs templates or files that become part of the output

Only create directories that the skill will actually use.

### Step 3: Initialize the Skill Manually

PicoClaw does not ship `scripts/init_skill.py` or `scripts/package_skill.py`. Create the scaffold directly.

Use one of these roots:

- Local user skill: `~/.picoclaw/workspace/skills/<skill-name>/`
- In-repo builtin skill contribution: `<repo>/workspace/skills/<skill-name>/`

Minimal scaffold:

```bash
SKILL_DIR="$HOME/.picoclaw/workspace/skills/my-skill"
mkdir -p "$SKILL_DIR"
mkdir -p "$SKILL_DIR/scripts" "$SKILL_DIR/references" "$SKILL_DIR/assets"
```

Minimal `SKILL.md` template:

```markdown
---
name: my-skill
description: Describe what the skill does and when it should trigger.
---

# My Skill

1. Put the core workflow here.
2. Keep the body concise.
3. Move long details into references/.
```

If the skill does not need `scripts/`, `references/`, or `assets/`, remove those empty directories.

### Step 4: Write the Skill

When writing the skill, optimize for another agent instance that has no project-specific context besides what you provide.

#### Frontmatter

Write YAML frontmatter with only:

- `name`
- `description`

The `description` is the main trigger surface. Include both what the skill does and when it should be used. Put trigger information in the description, not in a "when to use" section in the body.

#### Body

Write the body in imperative form. Keep it focused on:

- The workflow to follow
- Which bundled files exist and when to open them
- Constraints, quality bars, and decision rules that are not obvious from general model knowledge

If you need longer examples or detailed domain docs, store them in `references/` and link to them directly from `SKILL.md`.

Any script you add must be run and sanity-checked. If you create placeholder files while drafting, delete them before finishing unless they still add value.

### Step 5: Validate the Skill in PicoClaw

Before considering the skill done, verify:

- The folder contains `SKILL.md`
- `SKILL.md` frontmatter has a valid `name` and `description`
- The folder name matches the skill name
- The skill appears in `picoclaw skills list`

When testing from a source checkout, point PicoClaw at the repo's builtin skills explicitly:

```bash
PICOCLAW_BUILTIN_SKILLS="$PWD/workspace/skills" picoclaw skills list
```

Then trigger the skill with a realistic prompt and confirm the agent loads the intended instructions. There is no repo-local packaging helper; for in-repo contributions, commit the skill folder directly.

### Step 6: Iterate

After using the skill on real tasks:

1. Note where the agent hesitated, overused tools, or missed constraints
2. Tighten the trigger description or workflow instructions
3. Move repeated details into bundled resources when that reduces prompt bloat
4. Re-test with the original example prompts
