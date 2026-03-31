> Back to [README](../../../README.md)

# MaixCam

MaixCam is a dedicated channel for connecting to Sipeed MaixCAM and MaixCAM2 AI camera devices. It uses TCP sockets for bidirectional communication and supports edge AI deployment scenarios.

## Configuration

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    }
  }
}
```

| Field      | Type   | Required | Description                                                      |
| ---------- | ------ | -------- | ---------------------------------------------------------------- |
| enabled    | bool   | Yes      | Whether to enable the MaixCam channel                            |
| host       | string | Yes      | TCP server listening address                                     |
| port       | int    | Yes      | TCP server listening port                                        |
| allow_from | array  | No       | Allowlist of device IDs; empty means all devices are allowed     |

## Use Cases

The MaixCam channel enables PicoClaw to act as an AI backend for edge devices:

- **Smart Surveillance**: MaixCAM sends image frames; PicoClaw analyzes them using vision models
- **IoT Control**: Devices send sensor data; PicoClaw coordinates responses
- **Offline AI**: Deploy PicoClaw on a local network for low-latency inference
