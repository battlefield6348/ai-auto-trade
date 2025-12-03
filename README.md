# Taiwan Stock Analyzer × Golang × 全自動開發專案

本專案是一個以 Golang 實作的「台灣股票批次分析服務」。目標是：**人類只維護需求文件，所有程式碼與測試盡可能由 AI 依文件自動產生與維護**。

- 類型：後端批次服務（非 API、非前端網站）
- Domain：台灣股票（上市／上櫃）
- 主要任務：每天自動抓取資料 → 清洗 → 計算技術指標 → 儲存分析結果
- 架構文件：`docs/architecture.md`

---

## 0. 目前已實作功能（MVP 範圍）
- 登入：`POST /api/auth/login`（預設帳號：`admin@example.com`、`analyst@example.com`、`user@example.com`，密碼皆為 `password123`）
- 手動日 K 擷取：`POST /api/admin/ingestion/daily`
- 單日分析：`POST /api/admin/analysis/daily`
- 分析結果查詢：`GET /api/analysis/daily`
- 強勢股篩選（固定條件版）：`GET /api/screener/strong-stocks`
- API 規格：`docs/api/openapi.yaml`（可用 Swagger UI 瀏覽）

---

## 1. 專案目標與定位

### 1.1 功能目標（Domain）
- 自動取得台灣股票每日價量資料（開、高、低、收、量）。
- 進行基本清洗與整理，確保資料可用。
- 計算常見技術指標（例如 MA5 / MA20 / 成交量平均）。
- 每天產出分析結果並儲存（例如：站上均線、價位突破、量能放大）。

### 1.2 開發目標（流程）
- 使用 Golang + DDD + Clean Architecture。
- 人類僅維護 `/docs` 需求與架構文件，不直接改 `/internal` 程式碼。
- AI 依文件產生程式與測試，確保 `go test ./...` 通過。

---

## 2. 技術棧與設計原則（摘要）
- 語言：Golang（版本以 `go.mod` 為準）
- 架構：DDD + Clean Architecture
- 測試：`go test`
- 格式/靜態檢查：`go fmt ./...`、`go vet ./...`

---

## 3. 系統流程（批次服務）
1. 排程觸發
2. 取得指定日期日 K
3. 清洗與指標計算
4. 寫入儲存
5. 查詢與選股（API）

---

## 4. 專案目錄
```text
.
├── cmd/api                 # 進入點：啟動 HTTP 服務
├── internal                # Domain / Application / Infrastructure / Interface
├── docs                    # 架構與需求文件
├── test                    # 整合／E2E 測試（選用）
├── docs/api/openapi.yaml   # OpenAPI 規格
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── Makefile
```

---

## 5. 如何啟動與使用

### 5.1 使用 Docker Compose（建議）
```bash
docker compose up --build
```
- API：`http://localhost:8080`
- Swagger UI：`http://localhost:8081`（掛載 `docs/api/openapi.yaml`）
- DB：內建 Postgres（compose 服務 `db`，環境變數已設定）。

### 5.2 本機直接執行
```bash
# 需安裝對應的 Go 版本（見 go.mod）
make build        # 或 make run
./bin/ai-auto-trade
```
- 可自行設定 `DB_DSN`、`HTTP_ADDR` 等環境變數；未設定時會以記憶體儲存啟動。

### 5.3 MVP E2E 操作順序
1. 登入取得 token：`POST /api/auth/login`
2. 觸發日 K 擷取：`POST /api/admin/ingestion/daily`
3. 觸發單日分析：`POST /api/admin/analysis/daily`
4. 查詢分析結果：`GET /api/analysis/daily?trade_date=YYYY-MM-DD`
5. 查詢強勢股：`GET /api/screener/strong-stocks?trade_date=YYYY-MM-DD`

---

## 6. 開發與測試
- `make fmt` / `make fmt-check`
- `make vet`
- `make test` 或 `go test ./...`
- OpenAPI：`docs/api/openapi.yaml`（配合 Swagger UI）
