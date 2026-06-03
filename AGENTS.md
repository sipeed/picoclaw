# Local Working Notes

This fork uses `bogdanovich/forgeclaw:main` as the active local/runtime
development branch. The upstream project is tracked separately as
`sipeed/picoclaw:main` through the `upstream` remote.

Branch policy:

- `main` is the public feature-rich ForgeClaw branch with all changes we
  actually run locally.
- `upstream/main` tracks `sipeed/picoclaw:main` and should never be modified
  directly.
- `upstream-mirror` is an optional local backup mirror of `sipeed/picoclaw:main`.
- Merge or rebuild `main` onto the latest `upstream/main` periodically to stay
  current.
- Do not open upstream PRs directly from `main`.
- For upstream PRs, create a clean topic branch from the latest `upstream/main`
  or `upstream-mirror`, then cherry-pick or manually port only the intended
  patch.
- Do not use a `[codex]` prefix in PR titles.
- Use conventional PR titles with a functional scope and colon, such as
  `feat(providers): add Gemini search`, `fix(telegram): handle media groups`,
  `fix(agents): preserve topic routing`, or `feat(tools): add update_plan`.
