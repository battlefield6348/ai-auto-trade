# 專案 AI 開發指南

本文件提供 AI 在本專案執行任務時的基本規範與工作流程，確保開發一致性與可追溯性。

## 核心目標
- 以 BTC/USDT 分析與自動交易為主要範圍。
- 優先完成可運作的資料擷取、分析、推播與管理介面功能。
- 避免引入多餘依賴或複雜度，維持 MVP 可用性。

## 啟動方式 (僅限使用者手動執行)
- 使用 `config.yaml` 作為唯一設定來源。
- 啟動資料庫與 Swagger：`docker compose up -d`
- 啟動 API：`go run ./cmd/api/main.go`
- 若 DB Volume 已存在需重跑 migration：`docker compose exec db psql -U ai -d ai_auto_trade -f /docker-entrypoint-initdb.d/0001_init.sql`

## AI 行為限制
- **嚴禁自行啟動服務**：AI 在修復或開發過程中，**不得**在後台啟動 API 服務（如 `go run`）或資料庫。啟動與重啟服務的權限完全保留給使用者。AI 僅負責：
    - 修改程式碼邏輯與樣式。
    - 更新開發指南與技術文件。
    - 透過日誌分析問題，但不直接干預運行狀態。

## 開發原則
- 優先修正錯誤與阻斷流程的問題，再處理體驗與優化。
- 行為以文件為準，對應需求文件位於 `docs/specs/`。
- 僅在必要時加入註解，避免噪音。
- 對外溝通使用繁體中文（台灣用語），時間以台北時區（UTC+8）描述。
- 指令/文件範例以台灣開發者習慣撰寫（例如「幾點」、「排程」、「檔名」用語）。
- 程式碼註解優先使用繁體中文（台灣用語），且保持精簡。
- API 層優先使用 Gin 框架開發；程式碼可讀性與清楚的控制流程優先於效能微調。

## 功能現況（摘要）
- 自動管線：`ingestion.auto_interval` 會定期跑日 K 擷取與分析。
- Telegram 推播：`notifier.telegram` 會定期推送最新摘要與強勢交易對。
- Ingestion：`ingestion.use_synthetic` 控制是否使用合成日 K。
- Auth：新增獨立登入頁面 (`/web/login.html`) 與全站路由守衛 (Auth Guard)。
- 回測控制台：整合手動參數與資料庫策略載入。支援連續交易模擬、績效統計與數據可視化（含完整時間軸與命中點標註）。
- 環境切換：頂部導覽列支援全站同步切換 Test/Paper/Live 執行模式。

## 主要 API（摘要）
- 身份與授權：`POST /api/auth/login`, `POST /api/auth/register`
- 回測分析：`POST /api/analysis/backtest` (手動), `POST /api/analysis/backtest/slug` (策略)
- 策略重溫：`GET /api/analysis/strategies`, `POST /api/analysis/strategies/save-scoring`
- 執行概況：`GET /api/analysis/summary`
- 環境配置：`GET/POST /api/admin/binance/config`
- 任務監控：`GET /api/admin/jobs/status`, `GET /api/admin/jobs/history`
- 模擬交易：`POST /api/admin/trading/order` (手動下單)

## 交付規範
- 變更需更新 `README.md`（若涉及使用方式或設定）。
- 每次提交以繁體中文撰寫 commit message。
- 避免修改未被要求的檔案。
- 前端修改需以專業 UI/UX 角度評估資訊架構與互動動線，先闡述設計理由與取捨再進行異動。
- 若現有功能未使用或與需求重疊，優先精簡並移除，保持 MVP 範圍與維護成本最低。
- 所有功能皆需搭配測試；除環境設定外，測試碼可讀性優先於過度 DRY，適度重複以換取清晰度。
