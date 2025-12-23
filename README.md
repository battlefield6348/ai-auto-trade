# Price Action Crypto Analyzer · Golang · Auto Trading

本專案以價格行為學（Price Action）為核心，提供 BTC 市場分析與自動交易能力，並以 monorepo 管理後端服務、排程分析、自動交易模組與前端 Web Console。

## 核心目標
- 人類只維護需求與策略文件，所有分析邏輯、排程、交易與測試盡可能由 AI 依文件自動產出與維護。

## 理念：價格行為學（Price Action）
- 分析重點：價格結構（趨勢/盤整/反轉）、高低點行為（HH/HL/LH/LL）、關鍵區域（支撐/壓力/流動性區）、K 線型態（吞噬、假突破、停損掃描）、市場狀態（趨勢盤/震盪盤）。
- 策略必須可解釋、可追溯，不做黑箱模型。

## 主要功能模組
- **定期市場判斷**：固定週期（例 1H/4H/Daily）分析市場結構、關鍵區間、潛在進場方向；結果需可追溯並推送 Telegram（格式可直接執行）。
- **自動交易（Execution）**：分析後可決定掛單、市價進場或僅觀察；每筆交易含進場/停損/停利/倉位；分析模組與下單模組嚴格分離，可停用、可前端手動接管、決策可回溯。
- **Web Console**：策略控制台與監控介面，提供登入、最新/歷史分析、交易紀錄、啟停排程分析、啟停自動交易、策略版本切換、Telegram 推送預覽；不承載分析/交易邏輯。

## 系統流程
1. 排程觸發
2. 價格行為分析（Price Action Engine）
3. 市場狀態判斷（多/空/盤整）
4. 產出交易策略建議
5. 推送 Telegram 通知；（選用）自動交易執行

## 技術與架構原則
- **後端**：Golang；DDD + Clean Architecture。模組切分：Market Analysis / Strategy Decision / Trade Execution / Notification（Telegram）。策略邏輯需有文件、有測試、可由 AI 再生。
- **前端**：技術棧由 `frontend/web-console` 自行定義（建議 TypeScript），定位為內部管理介面（非高頻交易 UI）。所有頁面流程先定義於 `docs/frontend/web-console/*.md`。

## AI 全自動開發原則
- 人類只維護 `/docs` 需求與策略/流程文件。
- AI 產生後端程式碼、測試與前端頁面。
- 後端需保證 `go test ./...` 通過，所有行為可由文件推導。

## 快速啟動
- 安裝 Docker / Docker Compose（僅用於啟動 Postgres、Swagger）。
- 啟動資料庫與 Swagger：`docker compose up -d`
- 編輯 `config.yaml`（HTTP/DB/Auth/Ingestion 設定），啟動 API：  
  ```bash
  go run ./cmd/api/main.go
  ```
- Postgres 連線資訊：`postgres://ai:ai@localhost:5432/ai_auto_trade?sslmode=disable`，初次啟動會自動套用 `db/migrations/0001_init.sql`。
- API 預設埠：`http://localhost:8080`；Swagger UI：`http://localhost:8081`
- 若 DB Volume 已存在需重跑 migration，可用：`docker compose exec db psql -U ai -d ai_auto_trade -f /docker-entrypoint-initdb.d/0001_init.sql`
- Auth：預設帳號 `admin/analyst/user@example.com`，密碼皆 `password123`。`config.yaml` 可調整 `auth.token_ttl`（access token）、`auth.refresh_ttl`（refresh token），refresh token 以 HttpOnly Cookie + `auth_sessions` 資料表保存，可在服務重啟後自動續期；`auth.secret` 請於正式環境改成安全值。
- Ingestion：`config.yaml` 可設定 `ingestion.use_synthetic`（true=合成日 K，false=實際取 Binance）。
- 自動管線：`ingestion.auto_interval` 預設每小時自動跑當日的日 K 擷取與分析，免手動呼叫 API。
- 歷史補資料：`ingestion.backfill_start_date` 可指定啟動時自動補齊的起始日期（格式 YYYY-MM-DD），只會補尚未分析的日期，避免重複。
- Telegram 推播：在 `config.yaml` 設定 `notifier.telegram`（enabled/token/chat_id/interval/門檻），API 啟動後每小時將最新分析摘要與強勢交易對推送到指定 TG chat。
- 歷史回補：可呼叫 `POST /api/admin/ingestion/backfill` 回補指定區間，預設會以 Binance 實際資料覆蓋既有日 K 並同步分析。  
  ```bash
  curl -X POST http://localhost:8080/api/admin/ingestion/backfill \
    -H "Authorization: Bearer <token>" \
    -H "Content-Type: application/json" \
    -d '{"start_date":"2025-01-01","end_date":"2025-03-01","run_analysis":true}'
  ```
- 走勢圖資料：可呼叫 `GET /api/analysis/history?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD` 取得區間內分析序列（供前端走勢圖使用）。
- 加權條件回測：`POST /api/analysis/backtest`，輸入權重與門檻（score、日漲跌加分、量能加分、總分門檻，及回測天數），回傳命中日期與後續報酬，用於前端條件回測。

## 操作手冊（前端 Web Console）

1. 啟動
   - `docker compose up -d`（DB+Swagger），`go run ./cmd/api/main.go`（API+前端）。
   - 瀏覽器開 `http://localhost:8080/`。
2. 登入
   - 預設帳號：admin/analyst/user@example.com，密碼皆 `password123`（點選帳號 chip 可快速帶入）。
   - 成功後會寫入 HttpOnly refresh token，頁面重整或服務重啟時會自動續期 access token。
3. 日常操作
   - 健康檢查：頁首狀態顯示 DB/資料來源（實際或合成）。
   - 回補：在「歷史資料回補」選區間，勾選「回補後立即分析」按「開始回補」，結果會顯示成功/失敗天數（亦可等待自動排程）。
4. 查詢與走勢
   - 分析結果：在「分析結果查詢」輸入交易日，會顯示亮點卡片與完整表格。
   - 強勢清單：在「強勢交易對篩選」輸入日期/分數/量能門檻，按查詢。
   - 走勢圖：在「BTC/USDT 走勢圖」選起迄日按「載入走勢」，滑鼠移到線上可看當日摘要，圖下方會列出 Score ≥ 50 的日期。
5. 條件回測（加權分數）
   - 填寫權重與門檻：Score 權重、日漲跌加分與漲幅門檻(%)、量能加分與量能門檻(倍率)、近 5 日報酬加分與門檻(%)、均線乖離加分與門檻(%)、總分門檻。
   - 選回測區間，按「執行回測」。結果會顯示：
     - 命中日期卡片（含總分、漲跌、量能、+3/+5/+10 日報酬）
     - 統計：各回測天數的平均報酬與勝率
     - 圖上標記：命中日期會在走勢圖上以小圓點標示
6. 提示
   - 若圖表或查詢無資料，先跑回補或等待自動排程。
   - 如果需要改用合成資料，將 `config.yaml` 的 `ingestion.use_synthetic` 設為 true。

## Monorepo 結構（預期）
```
.
├── cmd
│   ├── api                 # HTTP API 服務
│   ├── scheduler           # 排程分析服務
│   └── trader              # 自動交易服務
├── internal                # Domain / Application / Infrastructure
├── frontend
│   └── web-console         # 前端 Web Console
├── docs
│   ├── architecture.md
│   ├── strategy            # 價格行為與交易策略文件
│   ├── backend
│   └── frontend
│       └── web-console
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── README.md
```

## 核心價值
- 不追求炫技指標；不做不可解釋的黑箱策略。
- 專注市場結構、風險控制、紀律化執行。
- AI 是開發執行者，而非決策者。
