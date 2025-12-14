# 資料庫 Schema 定義規格（Database Schema）

版本：v1.0  
狀態：草稿  
最後更新：2025-12-03  

---

## 1. 文件目的與範圍

本文件定義「BTC 現貨分析服務（預設交易對：BTC/USDT）」後端系統所需的**關聯式資料模型**，包含：

- 資料表（Tables）與主要欄位
- 主鍵與關聯關係（Relationships）
- 索引需求（Index Requirements）
- 資料一致性與約束（Constraints）
- 資料生命週期／保留策略（Retention）

本文件不包含任何程式碼或實際 DDL，只提供邏輯層規格，供 Codex 產生實際資料庫 schema 與 migration。

預設目標資料庫：PostgreSQL。

---

## 2. 共通設計原則

### 2.1 一般欄位慣例

所有主要資料表（非純關聯表）建議具備以下共通欄位：

- id：整數或 UUID，作為主鍵（Primary Key）
- created_at：建立時間（UTC 時間）
- updated_at：最後更新時間（UTC 時間）
- created_by：建立者使用者 ID（可選）
- updated_by：最後更新者使用者 ID（可選）

時間欄位：

- 系統內部儲存一律使用 UTC
- 與台灣交易日相關的日期欄位使用「純日期」（不含時間）

### 2.2 命名慣例（邏輯）

- 資料表名稱採小寫加底線：例如 `users`、`daily_prices`、`analysis_results`
- 欄位名稱盡量語意清楚：例如 `trade_date`、`close_price`
- 外鍵欄位命名為 `{表名單數}_id`：例如 `user_id`、`stock_id`

### 2.3 主鍵與關聯

- 多數表使用單一欄位主鍵 `id`
- 針對天然複合鍵（例如 `stock_id + trade_date`），可使用唯一約束（Unique Constraint），而非取代主鍵
- 關聯關係透過外鍵欄位表達，並視情況建立外鍵約束（可由實作階段決定是否強制）

---

## 3. 核心市場資料（Market Data）

### 3.1 交易對基本資料表：`stocks`

用途：儲存交易對基本資料（目前僅 BTC/USDT，預留多交易對），供其他模組參照。

主要欄位（邏輯層）：

- id：主鍵
- stock_code：交易對代碼（例 "BTCUSDT"）
- market_type：市場別（此處使用 `CRYPTO`，預留多交易所）
- name_zh / name_en：名稱（可空）
- industry：可填入 "Crypto"（或保留未來分類）
- listing_date / delisting_date：此場景可空
- status：狀態（active 等）
- category：種類（現貨）

約束與索引：

- `stock_code` + `market_type` 應為唯一（BTC/USDT + CRYPTO）
- 常用查詢需針對 `stock_code` 建索引
- 其他模組多透過 `id` 與其關聯

---

### 3.2 日 K 資料表：`daily_prices`

用途：儲存每筆交易對的日 K 原始資料，由 Data Ingestion 寫入（預設 Binance 1d K 線）。

主要欄位：

- id：主鍵
- stock_id：外鍵指向 `stocks.id`
- trade_date：交易日期（純日期）
- open_price：開盤價
- high_price：最高價
- low_price：最低價
- close_price：收盤價
- volume：成交量（張數）
- turnover：成交金額（整數或小數）
- trade_count：成交筆數（如來源提供）
- change：漲跌價差
- change_percent：漲跌幅（例如小數）
- is_limit_up：是否漲停（可空）
- is_limit_down：是否跌停（可空）
- is_dividend_date：是否除權息日（可空，未來用）

關聯與約束：

- 同一 `stock_id + trade_date` 必須唯一
- 不允許有負價格／負量（由 Data Ingestion 驗證）

索引：

- 主查詢模式：
  - 給定 `stock_id` + 日期區間 → 查日 K 序列
  - 給定 `trade_date` + 多檔股票 → 查當日全市場
- 建議建立：
  - `stock_id, trade_date` 複合索引
  - `trade_date` 單欄索引（供全市場查詢）

---

## 4. 分析結果與批次任務（Analysis & Jobs）

### 4.1 分析結果表：`analysis_results`

用途：儲存「日批次分析核心」為每檔股票 × 每個交易日產生的分析結果。

主要欄位：

- id：主鍵
- stock_id：外鍵，指向 `stocks.id`
- trade_date：交易日期
- analysis_version：分析模型版本（字串或整數）
- close_price：收盤價（冗餘欄位，用於快速查詢）
- change：漲跌價差
- change_percent：漲跌幅
- return_5d：近 5 日報酬
- return_20d：近 20 日報酬
- return_60d：近 60 日報酬
- high_20d：近 20 日最高價
- low_20d：近 20 日最低價
- price_position_20d：收盤價在 20 日區間的位置（百分比）
- ma_5 / ma_10 / ma_20 / ma_60：移動平均價
- ma_trend_flag：均線排列狀態（例如：多頭、空頭、其他）
- volume：當日成交量（冗餘）
- volume_avg_5d / volume_avg_20d：均量
- volume_ratio：成交量放大倍率
- volatility_20d / volatility_60d：波動度指標（如適用）
- score：綜合分數（0–100 或其他定義）
- tags：標籤集合（可用文字陣列或 JSON 儲存）
- status：分析狀態（成功、失敗）
- error_reason：若失敗，錯誤原因文字（可空）

關聯與約束：

- `stock_id + trade_date + analysis_version` 應唯一
- 若未來只保留最新版本，可定義 `stock_id + trade_date` 唯一

索引：

- `trade_date`（用於查某日全市場分析結果）
- `stock_id, trade_date`（用於查個股歷史）
- 針對常用排序欄位（如 `score`、`return_5d`、`volume_ratio`）可視情況建立複合索引，例如：
  - `trade_date, score DESC`
  - `trade_date, return_5d DESC`

---

### 4.2 Data Ingestion 任務表：`ingestion_jobs`

用途：記錄每次資料抓取任務（例行、回補、重抓）之執行情況。

主要欄位：

- id：主鍵
- job_type：任務類型（每日例行、回補、重抓）
- target_start_date：目標日期起始
- target_end_date：目標日期結束
- options：額外參數（例如限定市場、限定股票清單），建議用 JSON
- status：任務狀態（排程中、執行中、成功、部分成功、失敗）
- started_at / finished_at：開始／結束時間
- success_count：成功股票數
- failure_count：失敗股票數
- error_summary：錯誤摘要（文字）

索引：

- `job_type, created_at`
- `status, created_at`

---

### 4.3 Data Ingestion 子項表：`ingestion_job_items`

用途：記錄每次 Ingestion 任務中每檔股票或每個日期執行狀態。

主要欄位：

- id：主鍵
- ingestion_job_id：外鍵指向 `ingestion_jobs.id`
- stock_id：外鍵（可空，依任務型態）
- trade_date：目標日期（可空）
- status：成功、失敗、略過等
- error_reason：錯誤原因（可空）

索引：

- `ingestion_job_id`
- `stock_id, trade_date`

---

### 4.4 日批次分析任務表：`analysis_jobs`

用途：記錄每次日批次分析任務之執行狀態。

主要欄位：

- id：主鍵
- job_type：任務類型（每日例行、重跑某日、重跑區間）
- target_date：主要分析日期（必要）
- options：額外參數（例如限制市場／股票）
- analysis_version：分析版本
- status：任務狀態
- started_at / finished_at
- total_stocks：總股票數
- success_count / failure_count
- error_summary

索引：

- `target_date`
- `status, created_at`

---

### 4.5 日批次分析子項表：`analysis_job_items`

用途：記錄每檔股票在本次分析任務中的處理結果。

主要欄位：

- id：主鍵
- analysis_job_id：外鍵指向 `analysis_jobs.id`
- stock_id：外鍵
- status：成功／失敗
- error_reason：錯誤原因
- duration_ms：處理耗時（非必填）

索引：

- `analysis_job_id`
- `stock_id, analysis_job_id`

---

## 5. 身分驗證與 RBAC（Auth & RBAC）

### 5.1 使用者帳號表：`users`

用途：儲存系統使用者基本資料。

主要欄位：

- id：主鍵
- email：登入用 Email（唯一）
- password_hash：密碼雜湊值
- display_name：顯示名稱
- status：帳號狀態（active / disabled / locked）
- last_login_at：最後登入時間（可空）
- is_service_account：是否為 Service Account 帳號（布林）

索引與約束：

- `email` 必須唯一
- `status` + `email` 常用於登入判斷

---

### 5.2 角色表：`roles`

用途：定義 `admin` / `analyst` / `user` / 其他自訂角色。

主要欄位：

- id：主鍵
- name：角色名稱（如 "admin"、"analyst"）
- description：描述
- is_system_role：是否為系統預設角色（避免被刪除）

約束：

- `name` 唯一

---

### 5.3 使用者角色關聯表：`user_roles`

用途：表示使用者與角色為多對多關係（即使 v1 只讓一人一角，結構仍保留彈性）。

主要欄位：

- id：主鍵
- user_id：外鍵指向 `users.id`
- role_id：外鍵指向 `roles.id`

約束：

- 相同 `user_id + role_id` 不得重複

索引：

- `user_id`
- `role_id`

---

### 5.4 權限表：`permissions`

用途：定義所有權限名稱（例如 `screener.use`、`analysis_results.query` 等）。

主要欄位：

- id：主鍵
- name：權限名稱（字串，唯一）
- description：描述

---

### 5.5 角色–權限關聯表：`role_permissions`

用途：定義某角色擁有哪些權限。

主要欄位：

- id：主鍵
- role_id：外鍵
- permission_id：外鍵

約束：

- 相同 `role_id + permission_id` 不得重複

索引：

- `role_id`
- `permission_id`

---

### 5.6 Refresh Token / Session 表：`auth_sessions`（可選）

用途：支援 Refresh Token 與強制登出。

主要欄位：

- id：主鍵
- user_id：外鍵
- refresh_token_id：token 識別用字串（不儲存原文）
- expires_at：過期時間
- revoked_at：撤銷時間（可空）
- user_agent：裝置資訊（可空）
- ip_address：登入 IP（可空）

索引：

- `user_id`
- `refresh_token_id`

---

## 6. 選股器與條件模板（Screener）

### 6.1 選股條件模板表：`screener_presets`

用途：儲存系統預設與使用者自訂的選股條件組合。

主要欄位：

- id：主鍵
- owner_user_id：外鍵指向 `users.id`，若為系統預設可為空或特定系統帳號
- name：模板名稱
- description：描述
- is_public：是否為公開模板（所有人可見）
- is_system_preset：是否為系統預設模板
- condition_definition：條件定義（JSON，對應 `stock_screener.md` 的 Condition Model）
- sort_definition：排序規則（JSON）
- status：啟用／停用

索引：

- `owner_user_id`
- `is_public, is_system_preset`

---

### 6.2 選股查詢紀錄表（可選）：`screener_queries`

用途：紀錄使用者執行選股的歷史，支援統計與快取。

主要欄位：

- id：主鍵
- user_id：外鍵
- screener_preset_id：外鍵（可空，若使用模板）
- condition_definition：當時使用的條件（JSON）
- sort_definition：當時排序（JSON）
- trade_date：查詢目標日期
- result_count：命中股票數
- executed_at：實際執行時間

索引：

- `user_id, executed_at`
- `trade_date`

---

## 7. 通知與訂閱（Alerts & Notifications）

### 7.1 訂閱表：`subscriptions`

用途：使用者對「選股條件、單股條件、系統事件」的訂閱設定。

主要欄位：

- id：主鍵
- user_id：外鍵
- name：訂閱名稱
- subscription_type：訂閱類型（選股型、單股型、系統警報等）
- condition_definition：條件定義（JSON，與 Screener 或 Strategy 條件模型相容）
- min_hit_count：最少命中數量門檻（如 <3 檔就不通知）
- channels：通知管道列表（如 email、webhook，JSON）
- webhook_url：如使用 webhook，則儲存 URL（可空）
- is_active：是否啟用
- last_triggered_at：最近觸發時間（可空）

索引：

- `user_id`
- `subscription_type`
- `is_active`

---

### 7.2 通知紀錄表：`notifications`

用途：儲存實際發出的通知紀錄。

主要欄位：

- id：主鍵
- subscription_id：外鍵指向 `subscriptions.id`（系統警報可選擇使用特殊訂閱或空）
- user_id：外鍵（若有對應使用者）
- notification_type：類型（選股結果、股票監控、系統警報）
- trade_date：相關交易日（可空）
- payload_summary：內容摘要（文字或 JSON 摘要）
- channel：實際發送管道
- sent_at：發送時間
- status：成功／失敗
- error_reason：若失敗的原因（可空）

索引：

- `user_id, sent_at`
- `subscription_id, sent_at`
- `notification_type, sent_at`

---

## 8. 策略引擎（Strategies）

### 8.1 策略表：`strategies`

用途：儲存使用者定義的策略設定。

主要欄位：

- id：主鍵
- user_id：外鍵（策略擁有者）
- name：策略名稱
- description：策略描述
- strategy_type：策略類型（如單股策略、選股策略、產業策略）
- condition_definition：策略條件（JSON，可包含跨日邏輯）
- action_definition：策略動作（JSON，例如通知、匯出）
- schedule_definition：執行排程設定（JSON）
- is_active：是否啟用
- last_executed_at：最近執行時間
- last_triggered_at：最近觸發時間（有命中時）

索引：

- `user_id`
- `is_active`

---

### 8.2 策略執行紀錄表：`strategy_runs`

用途：紀錄每次策略執行的狀態。

主要欄位：

- id：主鍵
- strategy_id：外鍵
- run_date：執行對應的交易日（或邏輯日期）
- started_at / finished_at
- status：成功／部分成功／失敗
- total_candidates：總評估標的數
- hit_count：命中標的數
- error_summary：錯誤摘要（可空）

索引：

- `strategy_id, run_date`
- `status, started_at`

---

### 8.3 策略觸發結果表：`strategy_hits`

用途：儲存某次策略執行下命中的標的列表。

主要欄位：

- id：主鍵
- strategy_run_id：外鍵
- stock_id：外鍵（若策略命中的是股票）
- trade_date：相關交易日
- hit_detail：命中原因／指標快照（JSON，可包含當日關鍵指標）

索引：

- `strategy_run_id`
- `stock_id, trade_date`

---

## 9. 報表與匯出（Reports & Exports）

### 9.1 匯出任務表：`export_jobs`

用途：紀錄報表或清單匯出的請求與狀態。

主要欄位：

- id：主鍵
- user_id：外鍵（發起匯出者）
- export_type：匯出類型（市場報告、產業報告、策略報告等）
- parameters：匯出參數（JSON，例如日期、產業、策略 id）
- status：排程中、處理中、成功、失敗
- file_path：匯出檔案在系統內的儲存位置（如路徑或 key）
- created_at / started_at / finished_at
- error_reason：失敗原因（可空）

索引：

- `user_id, created_at`
- `export_type, created_at`
- `status, created_at`

---

## 10. 系統健康與監控（System Health）

### 10.1 系統事件／錯誤紀錄表：`system_events`

用途：儲存與系統健康相關的重要事件（例如大量失敗、來源異常等）。

主要欄位：

- id：主鍵
- event_type：事件類型（ingestion_error, analysis_error, alert_failure 等）
- severity：嚴重程度（info / warning / error / critical）
- message：事件訊息摘要
- details：詳細資訊（JSON）
- occurred_at：發生時間
- related_job_id：關聯的 job id（可空）
- related_stock_id：關聯股票 id（可空）

索引：

- `event_type, occurred_at`
- `severity, occurred_at`

---

## 11. 資料保留與容量考量（高層規格）

### 11.1 保留策略（初版）

- `daily_prices`：預計長期保留（多年），不建議刪除。
- `analysis_results`：
  - 保留至少 5 年以上，視空間與需求調整。
  - 若版本眾多，可只保留最新版本分析結果。
- `ingestion_jobs`、`analysis_jobs`、`system_events`：
  - 詳細紀錄可保留 1–2 年；再以前的資料可考慮歸檔或刪除。
- `notifications`、`strategy_runs`、`strategy_hits`：
  - 至少保留 1 年，以便回溯與除錯。

### 11.2 分割與優化（未來）

- 當資料量巨大時，可對：
  - `daily_prices`
  - `analysis_results`
  - `strategy_hits`
  
  採用分區表（例如依年份分區）以提升效能。

---

## 12. Migration 與版本管理（行為）

- 所有 schema 變更需透過 migration（版本化腳本）實作。
- 每次 schema 調整需更新本文件版本號與日期。
- 在 production 環境執行 migration 前，需：
  - 測試環境驗證
  - 明確描述相容性（是否破壞舊資料）

---

## 13. 與其他規格文件之關聯

- `data_ingestion.md`：
  - 對應 `stocks`、`daily_prices`、`ingestion_jobs`、`ingestion_job_items`
- `daily_batch_analysis.md`：
  - 對應 `analysis_results`、`analysis_jobs`、`analysis_job_items`
- `query_and_export.md`：
  - 主要查詢 `analysis_results`、`stocks`，並寫入 `export_jobs`
- `stock_screener.md`：
  - 使用 `analysis_results`、`stocks`，以及 `screener_presets`、`screener_queries`
- `alert_notification_engine.md`：
  - 對應 `subscriptions`、`notifications`
- `strategy_engine.md`：
  - 對應 `strategies`、`strategy_runs`、`strategy_hits`
- `reports_dashboard.md`：
  - 聚合來自多個表做統計與視覺化
- `authentication_authorization.md`、`roles_permissions.md`：
  - 對應 `users`、`roles`、`user_roles`、`permissions`、`role_permissions`、`auth_sessions`

---
