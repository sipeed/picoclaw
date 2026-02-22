# MaixCam

MaixCam 是专用于连接矽速科技 MaixCAM 与 MaixCAM2 AI 摄像设备的通道。它采用 TCP 套接字实现双向通信，支持边缘 AI 部署场景。

## 配置

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

| 字段           | 类型   | 必填 | 描述                             |
| -------------- | ------ | ---- | -------------------------------- |
| enabled        | bool   | 是   | 是否启用 MaixCam 频道            |
| server_address | string | 是   | TCP 服务器监听地址和端口         |
| allow_from     | array  | 否   | 设备ID白名单，空表示允许所有设备 |

## 使用场景

MaixCam 通道使 PicoClaw 能够作为边缘设备的 AI 后端运行：

- **智能监控** ：MaixCAM 发送图像帧，PicoClaw 通过视觉模型进行分析
- **物联网控制** ：设备发送传感器数据，PicoClaw 协调响应
- **离线AI** ：在本地网络部署 PicoClaw 实现低延迟推理
