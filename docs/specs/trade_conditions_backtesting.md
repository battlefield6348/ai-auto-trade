# 買賣條件與回測／交易功能規格

版本：v1.0  
狀態：草稿  
最後更新：2025-12-30

---

## 1. 文件目的

明確列出「買入／賣出條件設定、儲存、回測與實際交易」的完整需求，供後續 AI 開發遵循，避免因上下文過長而遺漏細節。所有實作須延續既有 BTC/USDT 日線資料管線（ingestion + daily analysis），並能被前後端共同使用。

---

## 2. 範圍與假設

- 交易對：現階段聚焦 BTC/USDT（日 K）；允許未來擴充多交易對。
- 資料來源：沿用現有 ingestion 與 daily analysis 產出的分析結果；不額外拉盤中即時資料。
- 時間粒度：以「日」為最小決策單位（信號以當日分析值決定，實際下單默認使用次一交易日開盤價或設定的執行價）。
- 帳號與權限：策略建立/啟用限 admin/analyst；查詢可授權至 user 視需求配置。
- 非範圍：高頻交易、子帳號資金分離、多層巢狀布林邏輯（先以單層 AND/OR 為主）。

---

## 3. 核心功能清單

- **條件建模**：能以結構化方式描述買入條件與賣出條件，支援 AND/OR 單層邏輯，引用分析欄位與閾值。
- **策略儲存與版本**：條件可儲存為策略，具備名稱、描述、狀態（draft/active/archived）、版本與適用環境（test/prod）。
- **回測引擎（區間）**：可指定起訖日期，依策略條件模擬「觸發買入 → 持有 → 達賣出條件才平倉」的往復交易。
- **重用與執行**：已儲存的策略可直接用於回測、測試服下單（paper trading）、正式服下單（real trading），使用相同決策邏輯避免偏移。
- **結果與追蹤**：回測與實際交易均需提供交易明細、統計指標、執行日誌；策略變更需可追溯。
- **路由命名**：沿用現有 `/api/admin/*` 風格與 router 慣例（http.ServeMux 路徑前綴固定、HTTP 動詞明確）。

---

## 4. 條件與策略模型

### 4.1 基本名詞

- **策略 (Strategy)**：包含一組買入條件、一組賣出條件、資金/風控設定、適用環境與版本資訊。
- **條件 (Condition)**：單一判斷式；條件組可用 AND 或 OR 串接（單層）。
- **觸發事件 (Signal)**：當條件滿足時產生的買入或賣出訊號。
- **環境 (Environment)**：`test`（paper，僅紀錄不實際下單）、`prod`（正式送單）。

### 4.2 可引用的欄位（最小集）

以 daily analysis 結果與日 K 為基準，條件至少可引用：

- 價格/報酬：`close`、`open`、`high`、`low`、`pct_change_1d`、`pct_change_3d`、`pct_change_5d`、`pct_change_10d`
- 量能：`volume`、`volume_ratio`（相對近 N 日平均）
- 均線與乖離：`ma_short`、`ma_long`、`ma_diff_pct`
- 分數與標籤：`score`、`tags`（量能放大、突破、震盪等）
- 其他：可擴充但須明確文件化，避免 ad-hoc 指標

### 4.3 條件表達

- 基本運算：`>`, `>=`, `<`, `<=`, `==`, `between`；`tags` 支援 `include any/all` 與 `exclude`.
- 邏輯：單層 AND/OR 組合；不支援巢狀。
- 方向：買入條件與賣出條件分開保存；買入只在空倉狀態評估，賣出只在持倉狀態評估。
- 冷卻/最小持有：可設定「最少持有天數」、「買入後冷卻 N 日不再觸發」以避免頻繁交易。
- MVP 限制：買入僅允許 1 條條件、賣出僅允許 1 條條件；新增時覆蓋上一條（方便快速切換信號）。

### 4.4 風控與資金設定

- 下單模式：`fixed_usdt`（固定金額）、`percent_of_equity`（以淨值比例）。
- 手續費與滑價：回測與實際執行皆可設定；預設使用交易所費率與 0.1% 滑價，允許策略層覆寫。
- 風控：`max_positions`（預設 1，日線單倉）、`stop_loss_pct`、`take_profit_pct`、`max_daily_loss_pct`（達成則當日停用策略）。
- 成交價採樣（可由策略設定）：`next_open`（預設）、`next_close`、`current_close`；實際與回測需一致。
- 允許配置「下單時間與價格」規則（如：以次日開盤價、市價、或指定撮合邏輯），需與回測一致。

---

## 5. 策略儲存與版本管理

- 欄位：`id`、`name`、`description`、`base_symbol`（預設 BTCUSDT）、`timeframe`（1d）、`buy_conditions`、`sell_conditions`、`risk_settings`、`status`（draft/active/archived）、`env`（test/prod/both）、`version`、`created_by`、`updated_by`、`updated_at`.
- 行為：
  - 新增/編輯時須保留歷史版本（至少紀錄 version 與變更摘要）。
  - `active` 狀態須唯一對應環境（同一環境同一交易對同時僅允許 1 個啟用策略，避免衝突）。
  - 可從既有策略複製為新版本（copy as new）。
  - 需提供「啟用/停用」操作並記錄操作者與時間。

---

## 6. 回測規格（區間內反覆買賣）

### 6.1 輸入

- 必填：`strategy_id`（或 inline 條件）、`start_date`、`end_date`.
- 選填：`initial_equity`（預設 10,000 USDT）、`order_size_mode`（fixed/percent）、`order_size_value`、`fees`、`slippage_pct`、`stop_loss_pct`、`take_profit_pct`、`cool_down_days`、`min_hold_days`、`max_positions`（預設 1）。
- 模式：單倉往復；如買入條件多日連續成立，僅在空倉時首次觸發。

### 6.2 執行邏輯

- 狀態機：`空倉` →（買入條件成立）→ `持倉` →（賣出條件成立或停損/停利）→ `空倉`，循環至區間結束。
- 價格採樣：信號判定用當日分析值；成交價預設為「次一交易日開盤價」。若區間最後一天觸發買入但無次日資料，視為未成交。
- 若策略選擇 `current_close` 或 `next_close`，需標記並於結果中顯示採樣模式。
- 風控：當日若達 `max_daily_loss_pct`，停止當日新信號；持倉仍可觸發賣出。
- 冷卻/最小持有：買入後需滿足 `min_hold_days` 才檢查賣出；賣出後 `cool_down_days` 期間不再買入。

### 6.3 輸出

- 交易明細：每筆含 `entry_date/price/reason`、`exit_date/price/reason`（賣出條件/停損/停利）、`pnl_usdt`、`pnl_pct`、`hold_days`。
- 統計：`total_return`、`cagr`（若需）、`max_drawdown`、`win_rate`、`avg_gain`、`avg_loss`、`profit_factor`、`exposure_pct`（持倉天數比例）、`trade_count`。
- 時間序列：每日淨值曲線（equity curve）與持倉狀態，供前端繪圖。
- 日誌：信號觸發、過濾原因、價格採用方式，便於對帳與追蹤。
- 保存：每次回測結果與參數需可儲存/查詢（至少留近 N 次；N 可設定，預設 20）。

---

## 7. 實際交易（測試服 / 正式服）

- 評估邏輯：與回測共用同一套條件解析與風控流程，避免行為不一致。
- 觸發時機：每日分析完成後自動評估；需提供手動觸發 API 以便立即測試。
- 環境隔離：
  - `test`：paper trading，寫入交易紀錄與日誌，不送出真實下單。
  - `prod`：真實下單（若暫無交易所串接，可先落地成 TODO stub，但流程與紀錄需完整）。
- 單一活躍策略：同一環境/交易對僅允許一個 `active` 策略；啟用新策略前須停用舊策略。
- 交易紀錄：實際執行結果需保存（策略版本、信號明細、成交價、手續費、滑價假設、失敗原因）。
- 安全機制：遇到連續失敗或 `max_daily_loss_pct` 觸發時，當日暫停策略並發送告警（可沿用 Telegram 客戶端）。

---

## 8. 資料表（最小決策）

- `strategies`
  - `id (uuid)`, `name`, `description`, `base_symbol`, `timeframe`, `env`（test/prod/both）, `status`（draft/active/archived）, `version`（int, 遞增）, `buy_conditions (jsonb)`, `sell_conditions (jsonb)`, `risk_settings (jsonb: order_size_mode/value, fees, slippage_pct, stop_loss_pct, take_profit_pct, cool_down_days, min_hold_days, max_positions, price_mode)`, `created_by`, `updated_by`, `updated_at`.
  - 約束：同一 `base_symbol + timeframe + env` 僅允許 1 筆 active。
- `strategy_backtests`
  - `id`, `strategy_id`, `strategy_version`, `start_date`, `end_date`, `params (jsonb: initial_equity, fees, slippage_pct, price_mode 等)`, `stats (jsonb)`, `equity_curve (jsonb)`, `trades (jsonb array)`, `created_by`, `created_at`.
- `strategy_trades`
  - 實際或紙本交易紀錄：`id`, `strategy_id`, `strategy_version`, `env`, `side (buy/sell)`, `entry_date`, `entry_price`, `exit_date`, `exit_price`, `pnl_usdt`, `pnl_pct`, `hold_days`, `reason`（賣出原因/停損/停利）, `params_snapshot (jsonb)`, `created_at`.
- `strategy_positions`
  - 當前持倉：`strategy_id`, `env`, `entry_date`, `entry_price`, `size`, `stop_loss`, `take_profit`, `status`。
- `strategy_logs`
  - 信號與執行日誌：`id`, `strategy_id`, `strategy_version`, `env`, `date`, `phase (signal/eval/order)`, `message`, `payload (jsonb)`, `created_at`.
- `strategy_reports`
  - 每次實際執行後可產生報告：`id`, `strategy_id`, `strategy_version`, `env`, `period_start`, `period_end`, `summary (jsonb: return, win_rate, dd, trade_count)`, `trades_ref`（引用 trades 或 snapshot json）, `created_by`, `created_at`.

---

## 9. API 需求（草案）

路徑以 `/api/admin` 為主（需 auth + RBAC），實際名稱可依既有 router 規則微調。

- 策略 CRUD：
  - `POST /api/admin/strategies`：建立策略（含買/賣條件與風控設定）。
  - `GET /api/admin/strategies`：列表（支持 status/env/name 篩選）。
  - `GET /api/admin/strategies/{id}`：查詢單筆（含版本資訊）。
  - `PUT /api/admin/strategies/{id}`：更新並自動 bump version。
  - `DELETE /api/admin/strategies/{id}`：封存（改 archived，不硬刪）。
- 啟用/停用：
  - `POST /api/admin/strategies/{id}/activate`（body: env=test|prod，校驗唯一 active）。
  - `POST /api/admin/strategies/{id}/deactivate`
- 回測：
  - `POST /api/admin/strategies/{id}/backtest`：以策略版本 + 區間回測。
  - `POST /api/admin/strategies/backtest`：傳入 inline 條件回測（不落盤策略）。
  - `GET /api/admin/strategies/{id}/backtests`：查詢策略歷史回測結果。
- 實際/紙本交易：
  - `POST /api/admin/strategies/{id}/run`：立即以當前資料評估並（視 env）paper/real 下單。
  - `GET /api/admin/trades`：查詢交易紀錄（filters: env, strategy_id, date range）。
  - `GET /api/admin/positions`：查詢當前持倉（paper/real）。
- 日誌與告警：
  - `GET /api/admin/strategies/{id}/logs`：近期信號與執行情況。
  - 若 prod 下單失敗或停損觸發，沿用 Telegram 發送摘要。
 - 報告：
   - `POST /api/admin/strategies/{id}/reports`：產生並保存報告（可指定期間與資料來源 backtest/paper/real）。
   - `GET /api/admin/strategies/{id}/reports`：列表查詢，可保存多筆報告供下載/檢視。

---

## 10. 前端 / 使用流程（Web Console）

- 登入頁：
  - 現有帳密登入流程，成功後進入策略管理；顯示環境狀態與使用者角色。
- 設定條件頁（策略管理）：
  - 建立/編輯策略：名稱、描述、交易對、環境、買/賣條件編輯（簡單表單或 JSON），風控設定。
  - 版本/狀態切換：顯示目前 test/prod 啟用的策略與版本。
  - MVP 限制：買入/賣出各僅 1 條條件（新增即覆蓋上一條），待驗證後再擴充多條。
- 回測頁：
  - 選擇策略版本 + 區間，輸入資金/費率/滑價；顯示交易明細、統計指標、淨值曲線、信號標記。
  - 支援將回測結果存為預設並快速重跑。
- 實際執行頁（測試服/正式服）：
  - 按鈕「立即評估並下單（paper/real）」。
  - 顯示當前持倉（紙本/正式）、今日停用狀態、最新告警。
  - 日誌檢視：最近 N 筆信號、過濾原因。
- 報告頁（多筆報告保存）：
  - 列表已產生的回測/紙本/正式報告，可篩選策略、環境、期間。
  - 檢視報告摘要（報酬、勝率、DD、交易明細連結）並支援匯出。
- 快速導覽（暫行單頁多分段）：可用錨點快速跳至走勢、分析查詢、強勢篩選、策略列表、交易紀錄、持倉、報告、操作紀錄；後續如需真正多頁/路由可再重構。

---

## 11. 資料持久化與可追溯性

- 所有策略、回測、交易紀錄需存 DB；記錄策略版本與操作人。
- 回測與實際交易共用同一套條件解析邏輯；若日後演進，需有版本化或遷移策略，避免舊紀錄無法重播。
- 交易紀錄需可對帳：包含信號欄位值、計算出的閾值、採用的成交價/費率/滑價。
- 日誌需可依日期/策略/環境篩選；至少保留 30 天或可設定。

---

## 12. 驗收標準（最小可用）

- 能新增一個策略（含買入、賣出條件與風控設定），成功儲存並查詢。
- 能指定策略與日期區間回測，得到交易明細、統計與淨值曲線；同條件重跑結果一致。
- 能啟用策略於 `test` 環境，於每日分析後自動產生 paper 訂單並可查詢紀錄。
- 能啟用策略於 `prod` 環境（即便暫時為 stub 送單），若達停損或每日風控即停止並產生日誌與告警。
- 前端可完成策略建立、回測、啟用/停用、日誌檢視的基本操作流程。
