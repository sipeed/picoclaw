> [README](../../project/README.ja.md) に戻る

# MaixCam

MaixCam は、Sipeed MaixCAM および MaixCAM2 AI カメラデバイスへの接続専用チャンネルです。TCP ソケットを使用した双方向通信を実装し、エッジ AI デプロイメントシナリオをサポートします。

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

| フィールド | 型     | 必須   | 説明                                                          |
| ---------- | ------ | ------ | ------------------------------------------------------------- |
| enabled    | bool   | はい   | MaixCam チャンネルを有効にするかどうか                        |
| host       | string | はい   | TCP サーバーのリッスンアドレス                                |
| port       | int    | はい   | TCP サーバーのリッスンポート                                  |
| allow_from | array  | いいえ | 許可するデバイスIDのリスト。空の場合はすべてのデバイスを許可 |

## ユースケース

MaixCam チャンネルにより、PicoClaw はエッジデバイスの AI バックエンドとして機能できます：

- **スマート監視**：MaixCAM が画像フレームを送信し、PicoClaw がビジョンモデルで分析する
- **IoT 制御**：デバイスがセンサーデータを送信し、PicoClaw がレスポンスを調整する
- **オフライン AI**：ローカルネットワークに PicoClaw をデプロイして低遅延推論を実現する
