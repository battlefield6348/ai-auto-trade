# 專案 AI 開發指南

本文件提供 AI 在本專案執行任務時的基本規範與工作流程，確保開發一致性與可追溯性。

## 核心目標
- 以 BTC/USDT 分析與自動交易為主要範圍。
- 優先完成可運作的資料擷取、分析、推播與管理介面功能。
- 避免引入多餘依賴或複雜度，維持 MVP 可用性。

## 啟動方式
- 使用 `config.yaml` 作為唯一設定來源。
- 啟動資料庫與 Swagger：`docker compose up -d`
- 啟動 API：`go run ./cmd/api/main.go`
- 若 DB Volume 已存在需重跑 migration：`docker compose exec db psql -U ai -d ai_auto_trade -f /docker-entrypoint-initdb.d/0001_init.sql`

## 開發原則
- 優先修正錯誤與阻斷流程的問題，再處理體驗與優化。
- 行為以文件為準，對應需求文件位於 `docs/specs/`。
- 僅在必要時加入註解，避免噪音。
- 對外溝通使用繁體中文（台灣用語），時間以台北時區（UTC+8）描述。
- 指令/文件範例以台灣開發者習慣撰寫（例如「幾點」、「排程」、「檔名」用語）。
- 程式碼註解優先使用繁體中文（台灣用語），且保持精簡。

## 功能現況（摘要）
- 自動管線：`ingestion.auto_interval` 會定期跑日 K 擷取與分析。
- Telegram 推播：`notifier.telegram` 會定期推送最新摘要與強勢交易對。
- Ingestion：`ingestion.use_synthetic` 控制是否使用合成日 K（true=合成，false=實際取 Binance）。
- Auth：預設帳號 `admin/analyst/user@example.com`，密碼皆 `password123`。

## 主要 API（摘要）
- 登入：`POST /api/auth/login`
- 手動擷取：`POST /api/admin/ingestion/daily`
- 手動分析：`POST /api/admin/analysis/daily`
- 分析查詢：`GET /api/analysis/daily`
- 走勢摘要：`GET /api/analysis/summary`
- 強勢交易對：`GET /api/screener/strong-stocks`

## 交付規範
- 變更需更新 `README.md`（若涉及使用方式或設定）。
- 每次提交以繁體中文撰寫 commit message。
- 提交節奏：以「適當功能/子模組」為一個 commit（避免一次巨大提交，也避免過度零碎），多個 commit 組成完整功能。
- 避免修改未被要求的檔案。
