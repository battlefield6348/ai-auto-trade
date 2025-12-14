# MVP 實作計畫（MVP Implementation Plan）

版本：v1.0  
狀態：草稿  
最後更新：2025-12-03  

---

## 1. 文件目的

本文件將 `/docs/specs/*.md` 中與 MVP 相關的規格，轉換為「可執行的開發任務清單」，供 Codex 依序實作。

目標：  
完成一條可跑通的端到端流程（詳見 `/docs/specs/mvp_end_to_end_flow.md`），現已改為 **BTC 現貨（BTC/USDT）** 場景：

> 已登入的使用者，可以呼叫 API，取得某一交易日的「BTC 分析與強勢條件結果」，  
> 並且這份清單以交易所日 K 資料（預設 Binance 1d；失敗時允許合成日 K 備援）計算而得。

---

## 2. 相關規格文件

MVP 需遵守以下規格文件（僅取子集實作）：

- `/docs/specs/authentication_authorization.md`
- `/docs/specs/roles_permissions.md`
- `/docs/specs/database_schema.md`
- `/docs/specs/data_ingestion.md`（MVP 子集，目標來源為 Binance BTC/USDT 1d K 線；允許合成備援）
- `/docs/specs/daily_batch_analysis.md`（MVP 子集，針對 BTC 單一交易對）
- `/docs/specs/query_and_export.md`（MVP 子集）
- `/docs/specs/stock_screener.md`（MVP 子集；目前僅 BTCUSDT 但預留擴充）
- `/docs/specs/mvp_end_to_end_flow.md`

---

## 3. 開發階段與優先順序

MVP 實作分為以下階段，須依照順序：

1. 基礎專案結構與 DB 連線
2. Auth ＋ RBAC 最小實作
3. 資料模型與 Repository 層（交易對=BTCUSDT）
4. Data Ingestion（單日抓取，MVP 版；Binance → 合成備援）
5. Daily Analysis（單日批次分析，MVP 版，單一交易對）
6. Query API（分析結果查詢）
7. 強勢 Screener API（固定條件版）
8. 基礎錯誤處理與驗收

以下逐一說明每個階段的任務項目。

---

## 4. 階段一：基礎專案結構與 DB 連線

### 4.1 目標

建立：

- 專案基本目錄結構（依本專案既有 README 規劃）
- 與 PostgreSQL 的連線設定
- 基礎啟動流程（例如 HTTP server 啟動）

### 4.2 任務

1. 根據專案既定架構（DDD + Clean Architecture）：
   - 建立必要的目錄（domain / application / infrastructure / interfaces 等）。
   - 初始化設定載入機制（環境變數／設定檔）。

2. 建立 DB 連線模組：
   - 讀取 DB 連線設定（對應 docker compose 中的 Postgres）。
   - 建立連線池與健康檢查。

3. 預留 migration 機制：
   - 定義一個標準執行 migration 的入口（例如啟動時自動或獨立指令）。
   - 後續由 Codex 依 `database_schema.md` 產生 migration。

---

## 5. 階段二：Auth ＋ RBAC 最小實作

### 5.1 目標

實作：

- 使用者登入（Email + 密碼）取得 Access Token。
- 針對保護 API 的 token 驗證 middleware。
- 最小角色與權限判斷（admin / analyst / user）。

### 5.2 任務

1. 依 `database_schema.md` 實作以下資料表：
   - `users`
   - `roles`
   - `user_roles`
   - `permissions`
   - `role_permissions`
   - （可選）`auth_sessions`

2. 實作基礎帳號模型：
   - 能建立初始使用者帳號（例如 seed 一個 admin、一個 user）。

3. 實作登入流程：
   - 依 `authentication_authorization.md` 與 `mvp_end_to_end_flow.md` 定義的 `/api/auth/login`。
   - 驗證 email + 密碼，簽發 Access Token。

4. 實作 token 驗證 middleware：
   - 檢查 `Authorization: Bearer <token>`。
   - 解析 token，取得 user_id 與角色資訊。
   - 將目前使用者上下文注入後續 handler。

5. 實作權限檢查機制：
   - 依 `roles_permissions.md` 設計角色 → 權限 mapping。
   - 提供統一方法檢查「目前使用者是否擁有某權限」。

6. 為後續 API 定義基本保護規則：
   - admin/analyst 專用的 `/api/admin/*` 路徑。
   - user 可用的 `/api/analysis/*`、`/api/screener/*`。

---

## 6. 階段三：資料模型與 Repository 層

### 6.1 目標

建立所有 MVP 用到的核心資料模型與 Repository 介面，以方便日後替換實作或加測試。

### 6.2 任務

1. 依 `database_schema.md` 建立以下資料表與對應 Domain Model / Repository 介面：
   - `stocks`（交易對，預設 BTC/USDT）
   - `daily_prices`
   - `analysis_results`

2. 為 MVP 流程提供必要的 Repository 方法（不必一次實作全部）：
   - `stocks`：
     - 依 `trading_pair` 查／寫。
     - 建立新交易對。
   - `daily_prices`：
     - 寫入（建立或更新）某日某交易對的日 K。
     - 依 `stock_id + 日期區間` 查歷史日 K。
     - 依 `trade_date` 查當日交易對日 K。
   - `analysis_results`：
     - 寫入／更新某日某交易對的分析結果。
     - 依 `stock_id + 日期區間` 查歷史分析結果。
     - 依 `trade_date` 查當日分析結果。
     - 依指定條件（score / return_5d / volume_ratio 等）做查詢（供 Screener 用）。

---

## 7. 階段四：Data Ingestion（MVP 單日抓取）

### 7.1 目標

實作一個「手動觸發」的單日 Ingestion API，使得：

- 對指定 `trade_date`，可以抓到日 K，並寫入 DB。

### 7.2 任務

1. 根據 `data_ingestion.md` 與 `mvp_end_to_end_flow.md`：
   - 定義 `/api/admin/ingestion/daily` 的 Handler 行為。

2. Ingestion 流程（MVP）：

   - 輸入：`trade_date`。
   - 從外部來源取得指定日期的全市場日 K 資料。
     - 外部來源存取實作由 Codex自行決定，可先用 mock / 假資料。
   - 更新 `stocks` 表：
     - 如有新股票代碼，建立 `stocks` 記錄。
   - 更新 `daily_prices` 表：
     - 以 `stock_id + trade_date` 為 key：
       - 若存在 → 更新。
       - 若不存在 → 建立。

3. 權限控制：
   - 僅 `admin` / `analyst` 且具 `ingestion.trigger_daily` 權限的使用者可呼叫。

4. 輸出：
   - 按 `mvp_end_to_end_flow.md` 定義的回應格式回傳結果統計。

---

## 8. 階段五：Daily Analysis（MVP 單日分析）

### 8.1 目標

實作一個「手動觸發」的單日分析 API，使得：

- 對指定 `trade_date`，可以計算最小集合指標，並寫入 `analysis_results`。

### 8.2 任務

1. 依 `daily_batch_analysis.md` 與 `mvp_end_to_end_flow.md`：
   - 定義 `/api/admin/analysis/daily` 行為。

2. 分析流程（MVP）：

   - 輸入：`trade_date`。
   - 從 `daily_prices` 讀取該日全市場日 K。
   - 對每檔股票：
     - 取得該股近 5 個交易日 `daily_prices`（含當日）。
     - 計算：
       - `close_price`
       - `change_percent`（與前一日收盤價比）
       - `return_5d`
       - `volume`
       - `volume_avg_5d`
       - `volume_ratio`
       - `score`（簡單可重現的公式）
     - 寫入／更新 `analysis_results`。

3. 錯誤處理：
   - 若某股缺少前一日或近 5 日資料：
     - 可選擇：跳過或標記為分析失敗，需有 status / error 記錄。
   - 不得因單一股票失敗，導致整體任務失敗。

4. 權限控制：
   - 僅 `admin` / `analyst` 且具 `analysis.trigger_daily` 權限者可呼叫。

---

## 9. 階段六：分析結果查詢 API

### 9.1 目標

提供基本查詢 API，讓一般使用者可取得某日分析結果列表。

### 9.2 任務

1. 依 `query_and_export.md` 與 `mvp_end_to_end_flow.md`：
   - 實作 `/api/analysis/daily`：

     - 欄位：
       - `trade_date`（必填）
       - `limit` / `offset`（可選）
     - 行為：
       - 連接 `analysis_results` 與 `stocks`。
       - 回傳清單與總筆數。

2. 權限控制：

   - 角色：`user` 以上。
   - 權限：`analysis_results.query`。

3. 回應內容：

   - 每筆至少包含：
     - `stock_code`
     - `stock_name`
     - `market_type`
     - `close_price`
     - `change_percent`
     - `return_5d`
     - `volume`
     - `volume_ratio`
     - `score`

---

## 10. 階段七：強勢股 Screener API（固定條件版）

### 10.1 目標

實作 MVP 主功能：  
「指定交易日 → 回傳強勢股清單」。

### 10.2 任務

1. 依 `stock_screener.md` 與 `mvp_end_to_end_flow.md`：
   - 實作 `/api/screener/strong-stocks`：

     - Query 參數：
       - `trade_date`（必填）
       - `limit`（可選）
       - `score_min`（可選，預設值）
       - `volume_ratio_min`（可選，預設值）

     - 內部邏輯條件（MVP 固定版本）：
       - `score >= score_min`
       - `return_5d > 0`
       - `volume_ratio >= volume_ratio_min`
       - `change_percent >= 0`

     - 排序：
       - 先依 `score` 由大到小。
       - 再依 `return_5d` 由大到小（若有需要）。

2. 權限控制：

   - 角色：`user` 以上。
   - 權限：`screener.use`。

3. 回應內容：

   - 同 Query API，但僅回傳符合條件之股票。

---

## 11. 階段八：錯誤處理與基本觀測

### 11.1 目標

確保 MVP 流程的錯誤回應與基本 log 完整，方便除錯與驗收。

### 11.2 任務

1. 統一錯誤回應格式（對照 `mvp_end_to_end_flow.md`）。  
2. 為以下情境提供明確錯誤碼：
   - 未登入 / token 錯誤。
   - 權限不足。
   - 指定 `trade_date` 尚未有分析結果。
   - Ingestion 或 Analysis 尚在執行中。
   - 未預期系統錯誤。

3. 實作基本 log：
   - 登入成功／失敗。
   - Ingestion / Analysis API 執行起訖與耗時。
   - Screener API 呼叫與條件摘要。

---

## 12. Codex 實作建議順序（給執行端）

以下為建議直接給 Codex 的執行順序：

1. 讀取：
   - `/docs/specs/database_schema.md`
   - `/docs/specs/authentication_authorization.md`
   - `/docs/specs/roles_permissions.md`
   - `/docs/specs/mvp_end_to_end_flow.md`

2. 先完成：
   - DB schema + migration
   - Auth + RBAC + User seed

3. 再依本文件第 4～10 節順序，完成：
   - Repository
   - Ingestion API（單日）
   - Analysis API（單日）
   - Analysis Query API
   - Strong Stocks Screener API

4. 最後執行整體端到端測試，驗證第 9 節（MVP 驗收標準）。

---
