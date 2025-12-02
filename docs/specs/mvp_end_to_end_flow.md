# MVP 端到端流程規格（End-to-End Flow）

版本：v1.0  
狀態：草稿  
最後更新：2025-12-03

---

## 1. 目的與範圍

本文件定義第一條可運作的最小端到端流程（MVP），目標是達成：

> 已登入的使用者，可以呼叫一個 API，取得某一交易日的「強勢股清單」，  
> 而這個清單是用真實台股日 K 資料、經過系統分析後產生的結果。

僅從既有規格擷取「必要子集」，形成可驗收的垂直流程。

---

## 2. MVP 整體流程概觀

1. 使用者登入：Email + 密碼取得 Access Token。  
2. 資料準備：手動觸發指定交易日的日 K 抓取，寫入 `stocks`、`daily_prices`。  
3. 每日分析：手動觸發單日分析，寫入 `analysis_results`。  
4. 分析結果查詢 API：查詢某日全市場分析結果。  
5. 強勢股 Screener API：固定條件，回傳當日強勢股清單。

---

## 3. 前置條件

1. PostgreSQL 可用；已執行 migration，至少包含：`users`、`stocks`、`daily_prices`、`analysis_results`、`roles`、`user_roles`、`permissions`、`role_permissions`。  
2. 至少有一個可登入帳號（角色 `user` 或 `analyst`）。  
3. 服務提供 HTTP API，並已套用認證 middleware。

---

## 4. 使用者故事（MVP）

> 身為已註冊並啟用的使用者，我可以：  
> 1) 用 Email + 密碼登入取得 Access Token。  
> 2) 在系統已完成資料抓取與分析後，  
> 3) 呼叫 API 取得指定交易日的「強勢股清單」，用於觀察或自用。

---

## 5. 實作範圍（擷取自既有規格）

- `authentication_authorization.md`：登入、Token 驗證、最小角色/權限檢查。  
- `data_ingestion.md`：單日日 K 抓取與寫入（手動觸發）。  
- `daily_batch_analysis.md`：單日分析，最小指標集合（漲跌幅、近 5 日報酬、成交量放大倍率、簡易 score）。  
- `query_and_export.md`：查詢某交易日全市場分析結果。  
- `stock_screener.md`：固定規則的「強勢股」 API（不實作通用條件模型）。

---

## 6. 流程分段詳述

### 6.1 登入與 Token

- **登入 API**：`POST /api/auth/login`  
  - Body：`email`, `password`  
  - 成功回傳：`access_token`, `token_type=Bearer`, `expires_in`（可選）  
  - 驗證帳號 active、密碼正確後簽發 Access Token（至少含 user_id、role 或可查角色）。

- **Token 驗證**：後續 API 需 `Authorization: Bearer <token>`；無 token 或無效 → 401。

### 6.2 資料抓取（手動）

- **API**：`POST /api/admin/ingestion/daily`  
  - 角色：`admin` 或 `analyst`；權限：`ingestion.trigger_daily`  
  - Body：`trade_date` (YYYY-MM-DD)、`market`（可選）  
  - 行為：抓取該日全市場日 K；若 `stocks` 無資料則建立；寫入/覆蓋 `daily_prices`。  
  - 回應：`status`、`trade_date`、`total_stocks`、`success_count`、`failure_count`。

### 6.3 每日分析（手動）

- **API**：`POST /api/admin/analysis/daily`  
  - 角色：`admin` 或 `analyst`；權限：`analysis.trigger_daily`  
  - Body：`trade_date` (YYYY-MM-DD)  
  - 行為：讀 `daily_prices`，計算最小指標：  
    - 價格/報酬：`close_price`、`change_percent`、`return_5d`（不足則無法計算）。  
    - 量能：`volume`、`volume_avg_5d`、`volume_ratio = volume / volume_avg_5d`（分母 0 則無效）。  
    - 簡易 `score`：固定公式，輸出穩定、值域固定（如 0–100）。  
  - 寫入 `analysis_results`：`stock_id`、`trade_date`、`analysis_version`（例 `"v1-mvp"`）、上述欄位、`status`、`error_reason`（如有）。  
  - 缺歷史資料者標記失敗，不影響其他股票。  
  - 回應：`status`、`trade_date`、`total_stocks`、`success_count`、`failure_count`。

### 6.4 分析結果查詢 API

- **API**：`GET /api/analysis/daily`  
  - 角色：`user` 以上；權限：`analysis_results.query`  
  - Query：`trade_date`(必填, YYYY-MM-DD)、`limit`(預設 100, 上限 1000)、`offset`(預設 0)  
  - 行為：查 `analysis_results` + `stocks`，依預設排序（如 `stock_code` 升冪）分頁回傳。  
  - 回應：`trade_date`、`total_count`、`items`（`stock_code`、`stock_name`、`market_type`、`close_price`、`change_percent`、`return_5d`、`volume`、`volume_ratio`、`score`）。

### 6.5 強勢股 Screener API（固定條件）

- **條件（可調預設）**：  
  - `score >= score_min`（預設 70）  
  - `return_5d > 0`  
  - `volume_ratio >= volume_ratio_min`（預設 1.5）  
  - `change_percent >= 0`

- **API**：`GET /api/screener/strong-stocks`  
  - 角色：`user` 以上；權限：`screener.use`  
  - Query：`trade_date`(必填)、`limit`(預設 50, 上限 200)、`score_min`(預設 70)、`volume_ratio_min`(預設 1.5)  
  - 行為：  
    1) 驗證 `analysis_results` 是否有該日資料。  
    2) 套用條件篩選：`score >= score_min`、`return_5d > 0`、`volume_ratio >= volume_ratio_min`、`change_percent >= 0`。  
    3) 依 `score` 由高到低排序，分數相同再依 `return_5d`。  
    4) 回傳前 `limit` 檔。  
  - 回應：`trade_date`、`params`（實際門檻）、`total_count`（不受 limit）、`items`（同上欄位集合）。

---

## 7. 回應格式與錯誤處理

- 成功建議：

```json
{
  "success": true,
  "data": { "..." : "..." }
}
```

- 常見錯誤代碼（示意）：
  - `AUTH_INVALID_CREDENTIALS`
  - `AUTH_FORBIDDEN`
  - `ANALYSIS_NOT_READY`（指定日期尚無分析結果）
  - `JOB_RUNNING`（Ingestion / Analysis 任務執行中）
  - `INTERNAL_ERROR`

---

## 8. MVP 必要權限對應

- 登入：無需特定權限（需帳號 active）。  
- `/api/analysis/daily`：`analysis_results.query`。  
- `/api/screener/strong-stocks`：`screener.use`。  
- `/api/admin/ingestion/daily`：角色 `admin/analyst` + `ingestion.trigger_daily`。  
- `/api/admin/analysis/daily`：角色 `admin/analyst` + `analysis.trigger_daily`。

---

## 9. MVP 驗收標準

1. **登入**：預建帳號可成功取得 Access Token。  
2. **資料抓取與分析**（admin/analyst 帳號）：  
   - 呼叫 `/api/admin/ingestion/daily` 抓某日。  
   - 呼叫 `/api/admin/analysis/daily` 分析同日。  
   - DB 有對應 `daily_prices` 與 `analysis_results`（含 score）。  
3. **查詢分析結果**（user 帳號）：  
   - `GET /api/analysis/daily?trade_date=YYYY-MM-DD` 成功分頁。  
4. **強勢股清單**（user 帳號）：  
   - `GET /api/screener/strong-stocks?trade_date=...` 可回傳 0~N 檔（即使 0 也要成功返回空陣列）。  
5. **權限驗證**：  
   - 未登入呼叫受保護 API → 401。  
   - user 呼叫 `/api/admin/*` → 403。

---

## 10. 未來擴充（非 MVP 範圍）

- 完整 Screener 條件模型。  
- 通知/訂閱系統、策略引擎、報表/儀表板。  
- 自動排程（每日自動 Ingestion / Analysis / Screener / 報表）。  
- 更完整的角色/權限管理 UI、完整回補機制。  
