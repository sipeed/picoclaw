# AGENTS.md

## 專案部署與自動化 (Project Deployment and Automation)

本專案支持使用 Google Cloud Build 進行自動化部署。

### 1. 關鍵部署文件 (Key Deployment Files)
- `Dockerfile`: 使用多階段構建（Multi-stage build）的 Go 1.24 環境，最終鏡像基於 Alpine Linux。
- `cloudbuild.yaml`: 配置了 Google Cloud Build 的步驟，包括構建鏡像並推送到 Google Artifact Registry。

### 2. 代理人維護任務 (Agent Maintenance Tasks)
身為開發代理人，在維護此專案時應遵守以下原則：

- **同步更新**:
  - 當修改 `Makefile` 中的編譯參數（如 `LDFLAGS`）時，必須同步更新 `Dockerfile` 中的 `go build` 命令。
  - 當新增系統級依賴（如 C 庫）時，需在 `Dockerfile` 的構建或運行階段安裝對應的 `apk` 包。

- **構建驗證**:
  - 在提交涉及構建邏輯的代碼前，建議先在本地嘗試構建 Docker 鏡像：
    ```bash
    docker build -t picoclaw-local .
    ```

- **部署配置**:
  - 如果部署環境發生變化（例如從 GCR 遷移到 Artifact Registry，或更改區域），請更新 `cloudbuild.yaml` 中的 `substitutions`。
  - 默認配置：
    - `_LOCATION`: `us-central1`
    - `_REPOSITORY`: `picoclaw`
    - `_IMAGE_NAME`: `picoclaw`

### 3. 手動觸發構建 (Manual Build Trigger)
如果自動觸發失效或需要即時部署，請執行：
```bash
gcloud builds submit --config cloudbuild.yaml .
```

### 4. 運行時注意事項 (Runtime Notes)
- 鏡像默認工作目錄為 `/app`。
- `PICOCLAW_HOME` 環境變量設為 `/app/.picoclaw`。
- 內置 Skills 已打包進 `/app/skills`，但在運行時程序會優先查找工作目錄下的 `skills` 或 `WORKSPACE_DIR/skills`。
