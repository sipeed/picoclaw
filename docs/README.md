# PicoClaw Documentation

PicoClaw documentation is organized by document type first and language second.

This file describes the recommended documentation layout, how translated files should be named, and what `make lint-docs` currently checks locally.

These conventions are intended as contributor guidance for new or moved docs. Existing docs may still have historical exceptions, and `make lint-docs` only checks a common subset of the patterns described here.

## Reader Navigation

If you are browsing docs rather than reorganizing them, start with these directory indexes:

- [Guides](guides/README.md): setup, configuration, provider, and workflow guides.
- [Reference](reference/README.md): precise configuration and behavior reference.
- [Operations](operations/README.md): debugging and troubleshooting material.
- [Security](security/README.md): security-focused guides and controls.
- [Architecture](architecture/README.md): implementation notes and internal design docs.
- [Migration](migration/README.md): upgrade and migration notes.

For channel-specific setup, start with [Chat Apps Configuration](guides/chat-apps.md) and then drill into `docs/channels/<name>/README.md` as needed.

## Principles

- Choose the document type directory first. Do not create language buckets such as `docs/zh/` or `docs/fr/`.
- Keep each translated document next to its English source document.
- Use English as the base filename with no locale suffix.
- Use lowercase locale suffixes for translations, for example `configuration.zh.md` or `README.pt-br.md`.
- Keep module-specific docs next to the code they describe instead of moving them into `docs/`.

## Recommended Directories

- `README.md`: English project entry document at the repository root.
- `docs/project/`: translated project entry documents such as `README.zh.md` and `CONTRIBUTING.zh.md`.
- `docs/guides/`: setup and usage guides.
- `docs/reference/`: reference material and detailed configuration docs.
- `docs/operations/`: debugging and troubleshooting docs.
- `docs/security/`: security-related documentation.
- `docs/architecture/`: architecture and internal design notes.
- `docs/channels/`: channel-specific integration guides.
- `docs/design/`: design proposals and investigations.
- `docs/migration/`: migration notes.

## Recommended Naming

- English documents use the base filename:
  - `README.md`
  - `configuration.md`
- Translations use `.<locale>.md`:
  - `README.zh.md`
  - `configuration.fr.md`
  - `README.pt-br.md`
- Code-adjacent translated READMEs follow the same rule:
  - `pkg/audio/asr/README.zh.md`
  - `pkg/isolation/README.zh.md`

## Common Patterns To Avoid

- Root-level translated entry docs such as `README.zh.md` or `CONTRIBUTING.fr.md`
  - Use `docs/project/README.zh.md` or `docs/project/CONTRIBUTING.fr.md` instead.
- Language directories under `docs/` such as `docs/zh/`, `docs/ZH/`, `docs/ja/`, or `docs/fr/`
  - Use `docs/<type>/<name>.<locale>.md` instead.
- Nested locale buckets such as `docs/guides/zh/configuration.md` or `docs/channels/telegram/zh/README.md`
  - Keep translations beside the English source file instead.
- Legacy translation filenames such as `README_zh.md` or `README_CN.md`
  - Use `README.zh.md`.
- Non-canonical locale suffixes such as `configuration_zh.md` or `configuration.ZH.md`
  - Use lowercase `.<locale>.md`, for example `configuration.zh.md`.

## Translation Placement

- For docs under `docs/guides`, `docs/reference`, `docs/operations`, `docs/security`, `docs/architecture`, `docs/channels`, and `docs/migration`, keep translations beside the English source file.
- For project entry translations, keep translated files in `docs/project/` and keep the English source in the repository root.
- In most cases, each translated file should have an English source document:
  - `docs/guides/configuration.zh.md` usually sits beside `docs/guides/configuration.md`
  - `docs/project/README.zh.md` usually corresponds to `README.md`
- Exception: `docs/design/` may contain locale-specific working notes without an English source document. The naming rules still apply there.

## Code-Adjacent Docs

Keep documentation next to the implementation when it primarily describes a package, command, example, or subproject.

Examples:

- `pkg/**/README.md`
- `cmd/**/README.md`
- `web/README.md`
- `examples/**/README.md`

These files still follow the same translation naming rules.

## Adding a New Document

1. Pick the correct document type directory.
2. Create the English source file first.
3. Add translated siblings after the English source exists when that source is part of the same docs set.
4. Update links from existing docs when the new doc becomes a navigation target.
5. Run `make lint-docs` locally when adding or moving docs.

## Examples

- New setup guide:
  - `docs/guides/launcher-setup.md`
  - `docs/guides/launcher-setup.zh.md`
- New security guide:
  - `docs/security/token-rotation.md`
- New translated package README:
  - `pkg/channels/README.zh.md`

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
- every translated Markdown file has a matching English source file
  - except for locale-specific working notes under `docs/design/`

`make lint-docs` is a local consistency check for common naming and placement mistakes. It helps contributors stay close to the recommended layout, but it is not intended to describe every acceptable documentation pattern in the repository.

When a check fails, `make lint-docs` prints the failing path, the reason, and a suggested fix.

If you change these recommendations or want the local linter to reflect them more closely, update this file and `scripts/lint-docs.sh` together.
