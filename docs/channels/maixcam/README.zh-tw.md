> 回到 [README](../../project/README.zh-tw.md)

# MaixCam

MaixCam 是用來連線至 Sipeed MaixCAM 與 MaixCAM2 AI 攝影機裝置的專用頻道。此頻道使用 TCP Socket 進行雙向通訊，並支援邊緣 AI 部署情境。

## 設定

```json
{
  "channel_list": {
    "maixcam": {
      "enabled": true,
      "type": "maixcam",
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    }
  }
}
```

| 欄位       | 型別   | 必填 | 說明                                       |
| ---------- | ------ | ---- | ------------------------------------------ |
| enabled    | bool   | 是   | 是否啟用 MaixCam 頻道                      |
| host       | string | 是   | TCP 伺服器監聽位址                         |
| port       | int    | 是   | TCP 伺服器監聽連接埠                       |
| allow_from | array  | 否   | 裝置 ID 允許清單；留空表示允許所有裝置     |

## 使用情境

MaixCam 頻道讓 PicoClaw 能作為邊緣裝置的 AI 後端服務：

- **智慧監控**：MaixCAM 傳送影像畫面，PicoClaw 使用視覺模型分析
- **IoT 控制**：裝置傳送感測器資料，PicoClaw 協調回應
- **離線 AI**：在區域網路部署 PicoClaw，提供低延遲推論
