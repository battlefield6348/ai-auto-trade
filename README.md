# Taiwan Stock Analyzer × Golang × 全自動開發專案

本專案是一個使用 Golang 實作的「台灣股票批次分析服務」。  
目標是：**人類只維護需求文件，所有程式碼與測試盡可能由 AI 依文件自動產生與維護**。

- 類型：後端批次服務（非 API、非前端網站）
- Domain：台灣股票（上市／上櫃）
- 主要任務：每天自動抓取資料 → 清洗 → 計算技術指標 → 儲存分析結果

更詳細的架構與 DDD/Clean Architecture 說明請見：`docs/architecture.md`。

---

## 1. 專案目標與定位

### 1.1 功能目標（Domain）

- 自動取得台灣股票每日價量資料（開、高、低、收、量）。
- 進行基本清洗與整理，確保資料可用。
- 計算常見技術指標（例如 MA5 / MA20 / 成交量平均）。
- 每天產出分析結果並儲存（例如：
  - 是否站上某條均線
  - 是否突破特定價位
  - 成交量是否放大

### 1.2 開發目標（流程）

- 使用 Golang + DDD + Clean Architecture 實作。
- 人類只負責：
  - 在 `/docs/requirements/` 新增或修改需求文件。
  - 在 `/docs` 更新架構與決策紀錄。
- AI（如 Codex / ChatGPT）負責：
  - 根據 `/docs` 內容產生與維護 `/internal` 下所有程式碼與測試。
  - 確保程式可成功編譯與測試。

---

## 2. 技術棧與設計原則（摘要）

- 語言：Golang（版本以 `go.mod` 為準）
- 架構風格：DDD + Clean Architecture
- 依賴管理：Go Modules
- 測試：`go test`（可搭配 `testify` 等）
- 格式化／檢查：
  - `go fmt ./...`
  - `go vet ./...`

詳細架構與層級依賴規則請見：`docs/architecture.md`。

---

## 3. 系統流程與模組概要（批次服務）

### 3.1 每日流程

典型流程（例如每天收盤後）：

1. 排程觸發（例如系統層的 `cron` 呼叫程式進入點）
2. Fetcher：取得指定日期的台股資料
3. Processor：
   - 清洗資料（格式轉換、排序、過濾異常值）
   - 計算技術指標（例如 MA5 / MA20 等）
   - 產出分析標記（例如突破／跌破／量能異常）
4. Storage：
   - 儲存原始價量資料
   - 儲存技術指標與分析結果
5. Reporter（可選）：
   - 產出報表（例如當日符合某些條件的股票清單，輸出為 CSV 或文字檔）

### 3.2 模組分層（概念）

- Scheduler（排程觸發）
- Fetcher（資料抓取）
- Processor（清洗與指標計算）
- Storage（資料儲存）
- Reporter（報表輸出，選用）

在程式碼實作上會對應到 DDD + Clean Architecture 的層級（domain/application/infrastructure），詳細請見 `docs/architecture.md`。

---

## 4. 專案目錄結構（摘要）

實際目錄會由 AI 依需求擴充，基本骨架如下：

```text
.
├── cmd/
│   └── api/
│       └── main.go           # 進入點：組合依賴、啟動每日批次流程
│
├── internal/
│   ├── domain/               # DDD Domain 層（不依賴其他層）
│   ├── application/          # Use Case / Application 層
│   ├── infrastructure/       # 資料庫、外部來源、設定等實作
│   └── pkg/                  # 共用工具（log, error, util...）
│
├── docs/
│   ├── architecture.md       # 架構與開發流程說明（給人類與 AI 看）
│   ├── glossary.md           # 名詞定義
│   ├── requirements/         # 業務需求與規格文件（唯一開發來源）
│   └── decisions/            # 技術與架構決策（ADR）
│
├── test/                     # 整合／E2E 測試（選用）
├── go.mod
├── go.sum
└── Makefile                  # 常用指令（fmt, vet, test, build 等）
