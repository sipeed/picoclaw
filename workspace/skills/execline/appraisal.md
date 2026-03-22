# Appraisal: Environment Sanitization PR

## Summary

PR1 adds environment sanitization with caching to the exec tool, enabling:
1. Clean environment for child processes (no leaked secrets)
2. LLM-controlled env injection (with blocklist)
3. Cached env at startup for efficiency

## Approach

### Design Decisions

| Decision | Rationale |
|----------|-----------|
| `[]string` as cache format | Direct compatibility with `os.Environ()` and `exec.Cmd.Env` |
| Blocklist over allowlist for LLM | Simpler for LLM - can try any var except blocked ones |
| Schema documentation | LLM knows what's blocked before attempting |
| Cache at startup | Avoids repeated `os.Environ()` syscalls |

### Security Properties

**What gets through:**
- Default allowlist: PATH, HOME, USER, LANG, SHELL, TERM, PWD, etc.
- Config-defined env_set overrides
- LLM-defined extraEnv (non-blocked vars only)

**What is blocked:**
- Secret vars from parent (API keys, tokens)
- LLM override of sensitive vars: PATH, HOME, USER, LD_PRELOAD, etc.

### Trade-offs

| Pros | Cons |
|------|------|
| No secret leakage to child processes | Additional startup cost (build env once) |
| LLM can inject debug vars | Blocklist may need expansion |
| Efficient caching | Cache is static - no dynamic updates |
| Compatible with execline/mvdan paths | - |

## Future Considerations

1. **Dynamic env updates** — Currently cache is built once at startup. Could add method to rebuild cache if needed.

2. **Expand blocklist** — Current list: PATH, HOME, USER, LOGNAME, SHELL, LD_PRELOAD, LD_LIBRARY_PATH, LD_AUDIT, LD_DEBUG. May need more.

3. **Per-command env isolation** — Currently env is shared across calls. Could offer isolated mode.

4. **Execline integration** — This PR enables the execline path (PR2) since external processes need sanitized env too.

## Code Metrics

- Production code: +125 lines
- Tests: +90 lines  
- Files changed: 5
- Functions: 3 new (`BuildSanitizedEnv` modified, `EnvironToSlice` added)

## Conclusion

This PR provides a solid foundation for environment handling. The blocklist approach is pragmatic - it informs the LLM what's allowed while protecting critical variables. The caching ensures efficiency for high-frequency exec calls.

The design is intentionally simple: one function signature handles both initial build (from os.Environ) and subsequent builds (from cached slice). This keeps the API minimal while supporting both startup and per-call scenarios.
