# MaixCam

MaixCam is a channel dedicated to connecting Sipeed MaixCAM and MaixCAM2 AI camera devices. It uses TCP sockets for bidirectional communication, supporting edge AI deployment scenarios.

## Configuration

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "server_address": "0.0.0.0:8899",
      "allow_from": []
    }
  }
}
```

| Field          | Type   | Required | Description                               |
| -------------- | ------ | -------- | ----------------------------------------- |
| enabled        | bool   | Yes      | Whether to enable the MaixCam channel     |
| server_address | string | Yes      | TCP server listening address and port     |
| allow_from     | array  | No       | Device ID allowlist, empty means allow all|

## Use Cases

The MaixCam channel enables PicoClaw to run as an AI backend for edge devices:

- **Smart Monitoring**: MaixCAM sends image frames, and PicoClaw analyzes them using vision models.
- **IoT Control**: Devices send sensor data, and PicoClaw coordinates responses.
- **Offline AI**: Deploy PicoClaw on a local network for low-latency inference.
