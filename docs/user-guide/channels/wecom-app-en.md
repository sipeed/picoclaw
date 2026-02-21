# WeCom App Configuration Guide

This document explains how to configure the WeCom App (wecom-app) channel in PicoClaw.

## Features

| Feature | Status |
|---------|--------|
| Receive messages | ✅ |
| Send messages | ✅ |
| Private chat | ✅ |
| Group chat | ❌ |

## Configuration Steps

### 1. WeCom Admin Console Setup

1. Log in to [WeCom Admin Console](https://work.weixin.qq.com/wework_admin)
2. Go to "Application Management" → Select your custom app
3. Record the following information:
   - **AgentId**: Shown on the app details page
   - **Secret**: Click "View" to get it
4. Go to "My Company" page and record the **CorpID** (Enterprise ID)

### 2. Message Reception Configuration

1. On the app details page, click "Set API Receiver" under "Receive Messages"
2. Fill in the following:
   - **URL**: `http://your-server:18792/webhook/wecom-app`
   - **Token**: Randomly generated or custom (for signature verification)
   - **EncodingAESKey**: Click "Random Generate" to generate a 43-character key
3. When you click "Save", WeCom will send a verification request

### 3. PicoClaw Configuration

Add the following configuration to your `config.json`:

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",           // Enterprise ID
      "corp_secret": "xxxxxxxxxxxxxxxxxxxxxxxx", // App Secret
      "agent_id": 1000002,                        // App AgentId
      "token": "your_token",                      // Token from message reception config
      "encoding_aes_key": "your_encoding_aes_key", // EncodingAESKey from message reception config
      "webhook_host": "0.0.0.0",
      "webhook_port": 18792,
      "webhook_path": "/webhook/wecom-app",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

## Troubleshooting

### 1. Callback URL Verification Failed

**Symptom**: WeCom shows verification failed when saving API receiver settings

**Check**:
- Confirm server firewall has port 18792 open
- Confirm `corp_id`, `token`, `encoding_aes_key` are configured correctly
- Check PicoClaw logs for incoming requests

### 2. Chinese Message Decryption Failed

**Symptom**: `invalid padding size` error when sending Chinese messages

**Cause**: WeCom uses non-standard PKCS7 padding (32-byte block size)

**Solution**: Ensure you're using the latest version of PicoClaw, which has fixed this issue.

### 3. Port Conflict

**Symptom**: Port already in use error at startup

**Solution**: Change `webhook_port` to another port, e.g., 18794

## Technical Details

### Encryption Algorithm

- **Algorithm**: AES-256-CBC
- **Key**: 32 bytes from Base64-decoded EncodingAESKey
- **IV**: First 16 bytes of AESKey
- **Padding**: PKCS7 (32-byte block size, not standard 16 bytes)
- **Message Format**: XML

### Message Structure

Decrypted message format:
```
random(16B) + msg_len(4B) + msg + receiveid
```

Where `receiveid` is `corp_id` for custom apps.

## Debugging

Enable debug mode to see detailed logs:

```bash
picoclaw gateway --debug
```

Key log identifiers:
- `wecom_app`: WeCom App channel related logs
- `wecom_common`: Encryption/decryption related logs

## References

- [WeCom Official Documentation - Receiving Messages](https://developer.work.weixin.qq.com/document/path/96211)
- [WeCom Official Encryption Library](https://github.com/sbzhu/weworkapi_golang)
