# Real-time Speech Translation System

這是一個即時語音翻譯系統，後端使用 Go 語言，搭配 Docker 運行的 Whisper 模型進行語音轉文字，並支援多語系翻譯後廣播給聽眾。

## 功能
- **Speaker Mode**: 用戶說話，系統即時錄音並傳送至後端。
- **Listener Mode**: 聽眾選擇語言，接收翻譯後的文字。
- **Real-time Processing**: 使用 WebSocket 串流音訊，後端透過 Whisper 進行辨識。

## 一般執行方式 (x86_64)

確保已安裝 Go 與 Docker。

```bash
# 下載依賴
go mod tidy

# 執行伺服器
go run ./cmd/server
```
打開瀏覽器訪問 `http://localhost:8080`。

---

## ARM 架構編譯及安裝指南

若您需要在 ARM 架構的機器（例如 Apple Silicon M1/M2/M3, Raspberry Pi 4/5, NVIDIA Jetson Orin 等）上運行此系統，請參考以下步驟：

### 1. 編譯 Go 執行檔 (Cross Compilation)

在您的開發機上（假設為 Windows/Linux/Mac），透過設定 Go 的環境變數來編譯出適用於 Linux ARM64 的執行檔。

**Windows (PowerShell):**
```powershell
$env:GOOS = "linux"
$env:GOARCH = "arm64"
go build -o realtransfer-arm64 ./cmd/server
```

**Linux / macOS:**
```bash
GOOS=linux GOARCH=arm64 go build -o realtransfer-arm64 ./cmd/server
```

### 2. 準備 Docker 環境

本系統依賴 Docker 容器來運行 Whisper 模型。

*   **關鍵注意**: 請確認您使用的 Docker 映像檔 `whisper-gx10` 支援 `linux/arm64` 架構。
*   如果該映像檔僅支援 `linux/amd64` (x86)，在 ARM 機器上執行會非常緩慢（透過 qemu 模擬），嚴重影響即時性，甚至可能無法運行。
*   **強烈建議**: 若原映像檔不支援 ARM，請取得 `Dockerfile` 並在 ARM 機器上重新 `docker build` 該映像檔。

### 3. 部署至 ARM 機器

將以下檔案與目錄複製到您的 ARM 機器上：

1.  編譯好的執行檔 `realtransfer-arm64`
2.  `web/` 目錄 (包含 `templates` 與 `static`)
3.  `data/` 目錄 (若無則需建立，用於存放暫存音檔)

### 4. 執行

在 ARM 機器上，賦予執行權限並啟動：

```bash
chmod +x realtransfer-arm64
./realtransfer-arm64
```

### 5. 常見問題 (Troubleshooting)

*   **Docker 錯誤**: 若出現 `exec format error`，表示 Docker 映像檔架構不符。
*   **GPU 加速**: 若您的 ARM 機器有 NVIDIA GPU (如 Jetson)，請確保 Docker 運行時有加入 `--runtime nvidia` (需修改 `internal/docker/executor.go` 中的指令參數)。
