# ForgeClaw Documentation

ForgeClaw documentation is organized by document type.

This file describes the current documentation layout and what `make lint-docs` checks locally.

ForgeClaw is a fork of PicoClaw. Many command names, paths, environment
variables, and package references still intentionally use `picoclaw`, for
example `picoclaw gateway` and `~/.picoclaw/config.json`. Treat those as
runtime names, not stale branding. Project-level marketing, release news,
hardware sales copy, and download links should be ForgeClaw-specific or omitted.

## Reader Navigation

If you are browsing docs rather than reorganizing them, start with these directory indexes:

- [Guides](guides/README.md): setup, configuration, provider, and workflow guides.
- [Reference](reference/README.md): precise configuration and behavior reference.
- [Operations](operations/README.md): debugging and troubleshooting material.
- [Security](security/README.md): security-focused guides and controls.
- [Architecture](architecture/README.md): implementation notes for current runtime mechanisms.
- [Migration](migration/README.md): upgrade and migration notes.

For channel-specific setup, start with [Chat Apps Configuration](guides/chat-apps.md) and then drill into `docs/channels/<name>/README.md` as needed.

## Principles

- Choose the document type directory first. Do not create language buckets such as `docs/zh/` or `docs/fr/`.
- Keep the `docs/` tree in English unless there is a concrete maintenance plan for translations.
- Keep module-specific docs next to the code they describe instead of moving them into `docs/`.

## Recommended Directories

- `README.md`: English project entry document at the repository root.
- `docs/guides/`: setup and usage guides.
- `docs/reference/`: reference material and detailed configuration docs.
- `docs/operations/`: debugging and troubleshooting docs.
- `docs/security/`: security-related documentation.
- `docs/architecture/`: architecture and internal implementation notes.
- `docs/channels/`: channel-specific integration guides.
- `docs/migration/`: migration notes.

## Recommended Naming

- English documents use the base filename:
  - `README.md`
  - `configuration.md`

## Common Patterns To Avoid

- Language directories under `docs/` such as `docs/zh/`, `docs/ZH/`, `docs/ja/`, or `docs/fr/`
- Nested locale buckets under `docs/guides/` or `docs/channels/telegram/`
- Legacy translation filenames such as `README_zh.md` or `README_CN.md`
- Non-canonical translation-like filenames such as `configuration_zh.md` or `configuration.ZH.md`

## Code-Adjacent Docs

Keep documentation next to the implementation when it primarily describes a package, command, example, or subproject.

Examples:

- `pkg/**/README.md`
- `cmd/**/README.md`
- `web/README.md`
- `examples/**/README.md`

## Adding a New Document

1. Pick the correct document type directory.
2. Create the English source file.
3. Update links from existing docs when the new doc becomes a navigation target.
4. Run `make lint-docs` locally when adding or moving docs.

## Examples

- New setup guide:
  - `docs/guides/docker.md`
- New security guide:
  - `docs/security/security_configuration.md`

## Validation

Run:

```bash
make lint-docs
```

The local docs linter currently checks these common cases:

- no root-level translated `README` or `CONTRIBUTING` files
- no `docs/<locale>/` language buckets, regardless of case
- no nested locale buckets under typed docs directories
- no legacy `README_*.md` filenames
- no non-canonical translation-like filenames such as `_zh.md` or `.ZH.md`
- no extra Markdown files directly under `docs/` except `docs/README.md`

`make lint-docs` is a local consistency check for common naming and placement mistakes. It helps contributors stay close to the recommended layout, but it is not intended to describe every acceptable documentation pattern in the repository.

When a check fails, `make lint-docs` prints the failing path, the reason, and a suggested fix.

If you change these recommendations or want the local linter to reflect them more closely, update this file and `scripts/lint-docs.sh` together.
