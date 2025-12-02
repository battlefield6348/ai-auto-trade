# MVP 端到端流程規格（End-to-End Flow）

版本：v1.0  
狀態：草稿  
最後更新：2025-12-03  

---

## 1. 目的與範圍

本文件定義本專案的「第一條可運作的最小端到端流程（MVP）」規格，目標是讓系統達到「實際可用」的最小標準：

> 已登入的使用者，可以呼叫一個 API，取得某一交易日的「強勢股清單」，  
> 而這個清單是用真實台股日 K 資料、經過系統分析後產生的結果。

本文件只挑選既有各規格中的「必要子集」，整合成一條可實作、可驗收的垂直流程，供 Codex 依此實作第一版系統。

---

## 2. MVP 整體流程概觀

MVP 端到端流程包含以下步驟：

1. **使用者登入**
   - 透過 Email + 密碼取得 Access Token。

2. **資料準備（Data Ingestion，手動觸發即可）**
   - 對指定交易日抓取台股日 K 資料。
   - 將資料寫入 `stocks`、`daily_prices`。

3. **每日分析（Daily Analysis，手動觸發即可）**
   - 對指定交易日的全市場股票進行最小集合的分析計算。
   - 產生並寫入 `analysis_results`。

4. **分析結果查詢 API**
   - 提供查詢「某日全市場分析結果」的 API。

5. **強勢股 Screener API（MVP 版）**
   - 提供一個固定條件的選股 API，回傳「當日強勢股清單」。

---

## 3. 前置條件

MVP 流程啟動前需滿足：

1. 已有可運作的 PostgreSQL（依 `docker-compose.yml`）。
2. 已執行 migration，建立 `database_schema.md` 所定義的最小必要表：
   - `users`
   - `stocks`
   - `daily_prices`
   - `analysis_results`
   - `roles` / `user_roles` / `permissions` / `role_permissions`（最小可用）
3. 至少存在一個可登入的使用者帳號：
   - 角色：`user` 或 `analyst`（可由 Admin 預先建立）。
4. 服務提供 HTTP API 入口（例如：`/api/...`），並已套用認證 middleware。

---

## 4. 使用者故事（MVP）

> 身為一個已註冊並啟用的使用者  
> 我可以：
> 1. 用 Email + 密碼登入並取得 Access Token  
> 2. 在系統已完成資料抓取與分析後  
> 3. 呼叫一個 API，取得指定交易日的「強勢股清單」，用於後續觀察或自行下單  

---

## 5. 實作範圍（從既有規格擷取）

本 MVP 只實作以下規格子集：

- 來自 `/docs/specs/authentication_authorization.md`
  - 登入（Login）
  - Token 驗證（Access Token）
  - 基本角色／權限檢查（最小版）

- 來自 `/docs/specs/data_ingestion.md`
  - 單日日 K 資料抓取與寫入
  - 不需要排程，只需透過 internal API 或 CLI 觸發

- 來自 `/docs/specs/daily_batch_analysis.md`
  - 單日分析，計算最小指標集合：
    - 漲跌幅、近 5 日報酬、成交量放大倍率、簡單 score
  - 不需要排程，只需透過 internal API 或 CLI 觸發

- 來自 `/docs/specs/query_and_export.md`
  - 查詢某交易日全市場分析結果的基本 API

- 來自 `/docs/specs/stock_screener.md`
  - 一個固定規則的「強勢股」 Screener API（不實作通用 Condition Model）

---

## 6. 流程分段詳述

### 6.1 A. 登入與 Token

#### 6.1.1 API：登入

- Method：`POST`
- Path：`/api/auth/login`
- Request Body（JSON）：
  - `email`：string
  - `password`：string
- Response（成功）：
  - `access_token`：string
  - `token_type`：固定為 `Bearer`
  - `expires_in`：秒數（可選）
- 行為：
  - 驗證帳號狀態為 active。
  - 密碼比對成功後簽發 Access Token。
  - Access Token 內需至少包含 user_id 與角色資訊（或可查詢）。

#### 6.1.2 Token 驗證

- 所有後續 API（Ingestion、Analysis、Query、Screener）皆需：
  - 從 HTTP Header `Authorization: Bearer <token>` 取得 token。
  - 驗證 token 有效、未過期。
  - 取得 user_id + 角色資訊。
- 若無 token 或無效：
  - 回傳 401 Unauthorized。

---

### 6.2 B. 資料抓取（MVP 版 Data Ingestion）

MVP 中只需支援「手動觸發單日抓取」，不須完整回補機制。

#### 6.2.1 API：單日資料抓取（internal）

- Method：`POST`
- Path：`/api/admin/ingestion/daily`
- 權限：
  - 角色：`admin` 或 `analyst`
  - 需具備權限：`ingestion.trigger_daily` 或等價 internal 權限
- Request Body（JSON）：
  - `trade_date`：string，格式 `YYYY-MM-DD`
  - `market`（可選）：預設為上市 + 上櫃全部
- 行為（邏輯）：
  1. 針對指定 `trade_date` 從外部資料來源抓取全市場日 K。
  2. 若 `stocks` 無對應股票，建立新股票基本資料。
  3. 將日 K 寫入 `daily_prices`：
     - `stock_id`、`trade_date`、開高低收、成交量、成交金額、漲跌幅等。
  4. 若同一 `stock_id + trade_date` 已存在，則覆蓋或更新資料。
- Response（成功）：
  - `status`：`success`
  - `trade_date`
  - `total_stocks`
  - `success_count`
  - `failure_count`

> 註：外部 API 細節不在本文件範圍內，此處僅定義行為與結果。

---

### 6.3 C. 每日分析（MVP 版 Daily Analysis）

MVP 中只需支持「手動觸發單日分析」，不須自動排程。

#### 6.3.1 API：單日分析執行

- Method：`POST`
- Path：`/api/admin/analysis/daily`
- 權限：
  - 角色：`admin` 或 `analyst`
  - 權限：`analysis.trigger_daily`
- Request Body（JSON）：
  - `trade_date`：string，格式 `YYYY-MM-DD`
- 行為（邏輯）：
  1. 查詢指定 `trade_date` 的全市場日 K（`daily_prices`）。
  2. 依每檔股票計算以下最小集合欄位：

     - 價格與報酬：
       - `close_price`
       - `change_percent`（與前一日收盤價比較）
       - `return_5d`：以收盤價計算近 5 個交易日報酬（若不足則標記為無法計算）

     - 量能：
       - `volume`
       - `volume_avg_5d`
       - `volume_ratio` = `volume / volume_avg_5d`（若分母為 0 則處理為無效）

     - 簡單 score（示意）：
       - 舉例：`score` = 權重組合 (`return_5d`、`change_percent`、`volume_ratio`)，具體公式由實作決定，但需：
         - 同樣輸入 → 同樣輸出
         - 值域固定（例如 0–100）

  3. 將結果寫入 `analysis_results`：
     - `stock_id`
     - `trade_date`
     - `analysis_version`（例如 `"v1-mvp"`）
     - `close_price`
     - `change_percent`
     - `return_5d`
     - `volume`
     - `volume_ratio`
     - `score`
     - `status`（成功／失敗）
     - `error_reason`（如有）

  4. 若某股票缺少必要歷史資料，則該筆標記為失敗，不影響其他股票。

- Response（成功）：
  - `status`：`success`
  - `trade_date`
  - `total_stocks`
  - `success_count`
  - `failure_count`

---

### 6.4 D. 分析結果查詢 API（MVP）

#### 6.4.1 API：查詢某日全市場分析結果

- Method：`GET`
- Path：`/api/analysis/daily`
- 權限：
  - 角色：`user` 以上
  - 權限：`analysis_results.query`
- Query Parameters：
  - `trade_date`（必填）：`YYYY-MM-DD`
  - `limit`（可選）：預設 100，最大可設定（例如 1000）
  - `offset`（可選）：預設 0
- 行為：
  - 從 `analysis_results` 與 `stocks` 查出指定日期的分析結果。
  - 依預設排序（例如依 `stock_code` 升冪）回傳分頁結果。
- Response（成功，JSON）：
  - `trade_date`
  - `total_count`
  - `items`：array，每筆包含：
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

### 6.5 E. 強勢股 Screener API（MVP 固定條件）

此 API 是 MVP 的「對外主打功能」。

#### 6.5.1 強勢股定義（MVP）

MVP 版「強勢股」條件可定義為：

- `score` ≥ `score_min`（預設 70）
- 且 `return_5d` > 0
- 且 `volume_ratio` ≥ 1.5
- 且 `change_percent` ≥ 0

具體門檻可配置，但在邏輯上須為「短期表現佳、量能放大、價格不弱」。

#### 6.5.2 API：取得強勢股清單

- Method：`GET`
- Path：`/api/screener/strong-stocks`
- 權限：
  - 角色：`user` 以上
  - 權限：`screener.use`
- Query Parameters：
  - `trade_date`（必填）：`YYYY-MM-DD`
  - `limit`（可選）：預設 50，最大例如 200
  - `score_min`（可選）：預設 70
  - `volume_ratio_min`（可選）：預設 1.5
- 行為：
  1. 驗證指定 `trade_date` 在 `analysis_results` 有資料。
  2. 套用上述條件篩選：
     - `score >= score_min`
     - `return_5d > 0`
     - `volume_ratio >= volume_ratio_min`
     - `change_percent >= 0`
  3. 按 `score` 由高到低排序，若分數相同可再按 `return_5d` 排序。
  4. 回傳前 `limit` 檔股票。

- Response（成功）：
  - `trade_date`
  - `params`：實際使用的門檻（`score_min`、`volume_ratio_min` 等）
  - `total_count`：符合條件的總數（不受 limit 限制）
  - `items`：array，每筆包含：
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

## 7. 回應格式與錯誤處理（MVP）

### 7.1 通用回應格式（建議）

成功時：

```json
{
  "success": true,
  "data": { ... }
}

7.2 常見錯誤情境

登入失敗：

error_code: AUTH_INVALID_CREDENTIALS

權限不足：

error_code: AUTH_FORBIDDEN

trade_date 尚未有分析結果：

error_code: ANALYSIS_NOT_READY

Ingestion / Analysis 任務執行中：

error_code: JOB_RUNNING

系統內部錯誤：

error_code: INTERNAL_ERROR

8. 權限對應（MVP 必要子集）

MVP 中需要的最小權限：

登入：

不需權限（但需帳號 active）

/api/analysis/daily：

analysis_results.query

/api/screener/strong-stocks：

screener.use

/api/admin/ingestion/daily：

角色：admin / analyst

權限：ingestion.trigger_daily

/api/admin/analysis/daily：

角色：admin / analyst

權限：analysis.trigger_daily

9. MVP 驗收標準

MVP 被視為「可用」需滿足：

登入

使用預先建立之帳號（如 user），能成功取得 Access Token。

資料抓取與分析

用 admin 或 analyst 帳號：

呼叫 /api/admin/ingestion/daily 對某日（例如 2025-12-01）抓日 K。

呼叫 /api/admin/analysis/daily 對同一日執行分析。

DB 中：

daily_prices 對應日期有資料。

analysis_results 對應日期有資料，且 score 等欄位有值。

查詢分析結果

用一般 user 帳號：

呼叫 /api/analysis/daily?trade_date=2025-12-01 成功取得分頁列表。

取得強勢股清單

用同一 user 帳號：

呼叫 /api/screener/strong-stocks?trade_date=2025-12-01

能得到至少 0~N 檔「強勢股」結果（如果沒有符合也需回傳 items: []，不可錯誤）。

權限驗證

未登入呼叫任何上述保護 API → 得到 401。

一般 user 呼叫 /api/admin/* → 得到 403。

10. 未來擴充（不屬於 MVP）

以下功能在其他文件已有定義，但 不包含在本次 MVP 實作範圍內，可於 MVP 之後逐步加入：

完整 Screener Condition Model（任意條件組合）

通知與訂閱系統（Alert & Notification）

策略引擎（Strategy Engine）

報表與儀表板（Reports & Dashboard）

自動排程（每日自動 Ingestion / Analysis / Screener / 報表）

更完整的角色／權限管理 UI

完整回補機制（多日、多檔股票）
