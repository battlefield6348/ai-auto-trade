# Taiwan Stock Analyzer × Golang × 前後端 × 全自動開發專案

本專案是一個以 Golang 為核心的「台灣股票批次分析服務」，採用 monorepo 管理 **後端服務** 與 **前端 Web Console**。  
目標是：**人類只維護需求文件，所有程式碼與測試盡可能由 AI 依文件自動產生與維護**。

- 類型：後端 API + 批次服務 + 前端 Web Console
- Domain：台灣股票（上市／上櫃）
- 主要任務：每天自動抓取資料 → 清洗 → 計算技術指標 → 儲存分析結果 → 由 Web Console 查詢與操作
- 架構文件：`docs/architecture.md`

---

## 0. 目前狀態與 MVP 範圍

### 0.1 後端（Backend）已實作功能

- 登入：`POST /api/auth/login`  
  - 預設帳號：`admin@example.com`、`analyst@example.com`、`user@example.com`  
  - 預設密碼：`password123`
- 手動日 K 擷取：`POST /api/admin/ingestion/daily`
- 單日分析：`POST /api/admin/analysis/daily`
- 分析結果查詢：`GET /api/analysis/daily`
- 強勢股篩選（固定條件版）：`GET /api/screener/strong-stocks`
- API 規格：`docs/api/openapi.yaml`（可用 Swagger UI 瀏覽）

### 0.2 前端（Frontend Web Console）目標（規劃方向）

前端 Web Console 主要用於：

- 以網頁介面操作登入 / 登出
- 手動觸發日 K 擷取 / 單日分析
- 瀏覽每日分析結果
- 瀏覽強勢股清單

前端需求與頁面說明將放在：

- `docs/frontend/web-console/*.md`（需求文件）
- 真實程式碼放在：`frontend/web-console/` 目錄下，由 AI 依文件生成

---

## 1. 專案目標與定位

### 1.1 功能目標（Domain）

- 自動取得台灣股票每日價量資料（開、高、低、收、量）。
- 進行基本清洗與整理，確保資料可用。
- 計算常見技術指標（例如 MA5 / MA20 / 成交量平均）。
- 每天產出分析結果並儲存（例如：站上均線、價位突破、量能放大）。
- 透過 API 與 Web Console 提供查詢與操作能力。

### 1.2 開發目標（流程）

- 使用 Golang + DDD + Clean Architecture 作為後端核心架構。
- 前端技術棧由 `frontend/web-console` 專案定義（例如 React / Vue / Next.js 等）。
- 人類僅維護 `/docs` 內的需求與架構文件，不直接修改 `/internal` 或前端程式碼。
- AI 依文件產生程式與測試，確保：
  - 後端：`go test ./...` 通過。
  - 前端：依專案設定執行對應測試／Lint／Build。

---

## 2. 技術棧與設計原則（摘要）

### 2.1 後端（Backend）

- 語言：Golang（版本以 `go.mod` 為準）
- 架構：DDD + Clean Architecture
- 測試：`go test`
- 格式/靜態檢查：`go fmt ./...`、`go vet ./...`
- API 規格來源：`docs/api/openapi.yaml`

### 2.2 前端（Frontend Web Console）

- 技術棧：由 `frontend/web-console` 專案自訂（建議：TypeScript + 主流前端框架）
- 功能定位：
  - 管理介面（Internal Console）
  - 操作分析流程與查詢結果的 GUI
- 規格來源：`docs/frontend/web-console/*.md`（頁面規格、流程、API 依賴）

---

## 3. 系統流程（批次服務與操作）

1. 排程觸發（例如每日收盤後）
2. 取得指定日期日 K 資料
3. 清洗與指標計算
4. 寫入儲存（DB）
5. 透過：
   - API 查詢與篩選
   - 前端 Web Console 進行查詢、檢視與操作

---

## 4. 專案目錄（Monorepo 結構）

```text
.
├── cmd/api                   # 後端 HTTP 服務進入點
├── internal                  # 後端 Domain / Application / Infrastructure / Interface
├── frontend
│   └── web-console           # 前端 Web Console 專案（框架由此目錄定義）
├── docs
│   ├── architecture.md       # 整體架構說明
│   ├── api
│   │   └── openapi.yaml      # API 規格（供後端 / 前端參考）
│   ├── backend               # 後端需求文件（以功能或 bounded context 切分）
│   └── frontend
│       └── web-console       # 前端頁面、流程需求文件
├── test                      # 整合／E2E 測試（選用）
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── Makefile
