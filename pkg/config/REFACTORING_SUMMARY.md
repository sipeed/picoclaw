# Security Configuration Refactoring Summary

## Overview

Successfully refactored `pkg/config/config.go` to support a separate `security.yml` file for storing all sensitive data (API keys, tokens, secrets, passwords).

## Changes Made

### New Files Created

1. **`pkg/config/security.go`** (New file)
   - Defines `SecurityConfig` structure for all sensitive data
   - Implements `LoadSecurityConfig()` to load from YAML
   - Implements `SaveSecurityConfig()` to save with secure permissions (0o600)
   - Implements `ResolveReference()` to resolve `ref:` prefixed strings
   - Supports all model, channel, web tool, and skills security entries

2. **`pkg/config/security_test.go`** (New file)
   - Comprehensive unit tests for security config loading
   - Tests for reference resolution (models, channels, web tools, skills)
   - Tests for file I/O operations

3. **`pkg/config/security_integration_test.go`** (New file)
   - Integration tests for full workflow
   - Tests backward compatibility with direct values
   - Tests mixed usage of references and direct values
   - Tests error handling for invalid references

4. **`security.example.yml`** (New file)
   - Template for users to copy and fill in
   - Includes all possible security entries with placeholder values
   - Located at project root

5. **`pkg/config/SECURITY_CONFIG.md`** (New file)
   - Complete documentation for the security config feature
   - Usage examples and reference format guide
   - Migration guide from old config
   - Security best practices

6. **`pkg/config/example_security_usage.go`** (New file)
   - Practical examples in Go comment format
   - Shows complete workflow from creation to usage
   - Lists all available reference paths

### Modified Files

1. **`pkg/config/config.go`**
   - Added `applySecurityConfig()` function to resolve all `ref:` references
   - Modified `LoadConfig()` to:
     - Load security config from `security.yml`
     - Apply security references to all config fields
     - Maintain backward compatibility with direct values
   - Updated warning message to suggest using `security.yml`

## Key Features

### Reference Format

Uses dot notation for referencing values:
- Models: `ref:model_list.<model_name>.api_key`
- Channels: `ref:channels.<channel_name>.<field>`
- Web Tools: `ref:web.<provider>.<field>`
- Skills: `ref:skills.<registry>.<field>`

### Supported Security Entries

**Models:**
- API keys for all model configurations

**Channels:**
- Telegram: token
- Feishu: app_secret, encrypt_key, verification_token
- Discord: token
- QQ: app_secret
- DingTalk: client_secret
- Slack: bot_token, app_token
- Matrix: access_token
- LINE: channel_secret, channel_access_token
- OneBot: access_token
- WeCom: token, encoding_aes_key
- WeComApp: corp_secret, token, encoding_aes_key
- WeComAIBot: token, encoding_aes_key
- Pico: token
- IRC: password, nickserv_password, sasl_password

**Web Tools:**
- Brave: api_key
- Tavily: api_key
- Perplexity: api_key
- GLMSearch: api_key

**Skills:**
- GitHub: token
- ClawHub: auth_token

### Backward Compatibility

- Direct values in `config.json` still work
- Mixed usage of references and direct values is supported
- Optional security file (if missing, only references fail)
- No breaking changes to existing configurations

## Testing

All tests pass successfully:

```bash
go test ./pkg/config -v
```

Test coverage includes:
- ✅ Unit tests for reference resolution
- ✅ Integration tests for full workflow
- ✅ Backward compatibility tests
- ✅ Error handling tests
- ✅ File I/O and permission tests
- ✅ All existing config tests still pass

## Usage Example

### config.json
```json
{
  "version": 1,
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api.openai.com/v1",
      "api_key": "ref:model_list.gpt-5.4.api_key"
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "ref:channels.telegram.token"
    }
  }
}
```

### security.yml
```yaml
model_list:
  gpt-5.4:
    api_key: "sk-proj-actual-key-here"

channels:
  telegram:
    token: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
```

## Migration Path

1. Copy `security.example.yml` to `~/.picoclaw/security.yml`
2. Fill in actual API keys and tokens
3. Update `config.json` to use `ref:` references
4. Set proper permissions: `chmod 600 ~/.picoclaw/security.yml`
5. Test with `picoclaw --version`

## Security Benefits

1. **Separation of concerns**: Configuration and secrets are in separate files
2. **Easier sharing**: Config can be shared without exposing secrets
3. **Better version control**: `security.yml` can be added to `.gitignore`
4. **Flexible deployment**: Different environments can use different security files
5. **Secure file permissions**: Saved with `0o600` by default

## Implementation Details

### File Loading Flow

```
LoadConfig()
  ├─ Load config.json
  ├─ Detect version
  ├─ Parse config based on version
  ├─ Load security.yml (optional)
  ├─ Apply security references
  │   └─ Resolve all "ref:" prefixes
  ├─ Parse environment variables
  ├─ Resolve API keys (file://, enc://)
  ├─ Expand multi-key models
  └─ Validate and return
```

### Reference Resolution

The `ResolveReference()` function:
1. Checks if string starts with `ref:`
2. Parses the dot-notation path
3. Navigates the security config structure
4. Returns the actual value
5. Returns error if path doesn't exist

### Error Handling

- Clear error messages with full context
- Includes the reference path and field name
- Fails early on invalid references
- Maintains backward compatibility

## Dependencies

Added dependency: `gopkg.in/yaml.v3` for YAML parsing

## Files Modified Summary

- **Created**: 6 new files (security.go, tests, docs, examples)
- **Modified**: 1 file (config.go - added security integration)
- **Lines added**: ~1000+ lines (including tests and documentation)
- **Backward compatible**: ✅ Yes
- **Breaking changes**: ❌ None

## Next Steps

1. Update main README to mention security.yml
2. Add security.yml to .gitignore
3. Update documentation with security config examples
4. Consider adding migration tool for existing users
5. Add validation for security.yml schema

## Conclusion

The refactoring successfully implements a secure, flexible, and backward-compatible way to manage sensitive configuration data. All tests pass and the feature is ready for use.
