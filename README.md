# AI Auto Trade: 原動力量化交易系統

本專案是一個整合行情分析、策略回測與自動化交易的完整平台，專注於加密貨幣市場（BTC/USDT）。系統結合了自動化資料管線與自定義評分策略，旨在提供紀律化的交易執行環境。

---

## 🚀 核心功能
*   **自動化資料管線**：每日自動抓取 Binance 行情並計算技術指標。
*   **AI 評分系統**：根據技術面與市場趨勢，產出多維度的 AI 原始評分。
*   **回測控制台**：直覺的 UI 介面，支援動態權重調整、門檻測試及績效分析。
*   **多環境自動交易**：
    *   **Testnet**：真實串接 Binance 測試網執行模擬交易。
    *   **Paper**：無風險虛擬模擬盤。
    *   **Live**：正式主網自動化執行。

---

## 🛠 技術棧
*   **後端**：Golang (DDD, Clean Architecture)
*   **前端**：Vanilla JS + CSS (Rich Aesthetics UI)
*   **資料庫**：Postgres (Time-series optimization)
*   **API 串接**：Binance API (K-lines, Trades)

---

## 📂 文件總覽
詳盡的系統說文件位於 `/docs` 目錄下：
*   [**文件總覽**](./docs/文件總覽.md)
*   [**功能規格**](./docs/功能規格.md)：了解系統如何運作。
*   [**系統架構**](./docs/系統架構.md)：了解程式碼目錄與設計模式。
*   [**技術規格**](./docs/技術規格.md)：資料庫模型與 API 客戶端定義。

---

## ⚡ 快速啟動

### 1. 環境建置
確保您的系統已安裝 Docker 與 Docker Compose。

```bash
docker compose up -d
```

### 2. 配置設定
複製 `config.example.yaml` 並重新命名為 `config.yaml`，填入您的資料庫 DSN、JWT Secret 及 Binance API Keys。

### 3. 啟動服務
```bash
go run ./cmd/api/main.go
```
啟動後訪問 `http://localhost:8080` 即可進入管理後台。

---

## 🔒 預設帳號
*   **Email**: `admin@example.com`
*   **Password**: `password123`
