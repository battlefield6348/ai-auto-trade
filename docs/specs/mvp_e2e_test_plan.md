# MVP 端到端測試計畫（E2E Test Plan）

版本：v1.0  
狀態：草稿  
最後更新：2025-12-03

---

## 1. 文件目的

本文件定義「MVP 端到端功能」的測試流程、測試案例、驗收標準，確保系統的第一條完整功能（登入 → 抓資料 → 分析 → 查詢 → 選股）能正確運作。

本文件將作為：
- Codex 自動產生 E2E 測試的依據
- 人工驗收的標準
- 未來系統迭代時 regression test 的基準

---

## 2. 測試範圍

對應 `/docs/specs/mvp_end_to_end_flow.md`，測試項目包含：

1. 使用者登入  
2. Data Ingestion（單日抓取，BTC/USDT）  
3. Daily Analysis（單日分析，BTC/USDT）  
4. 分析結果查詢 API（交易對欄位）  
5. 強勢交易對 Screener API  
6. RBAC（角色／權限）  
7. 錯誤處理  
8. 任務流程正確銜接（真正的 End-to-End）

不包含：
- 自動排程
- 多日期回補
- 完整 Screener 模型
- 策略、訂閱、通知
- 儀表板

---

## 3. 前置條件

測試前需準備：

1. DB 已建立且 migrations 已執行。
2. 系統已啟動並可接受 HTTP 請求。
3. 已存在以下帳號（由 seed 建立）：

| Email               | 密碼        | 角色    |
| ------------------- | ----------- | ------- |
| admin@example.com   | password123 | admin   |
| analyst@example.com | password123 | analyst |
| user@example.com    | password123 | user    |

4. 測試使用固定日期：  
   `2025-12-01`（可替換，但需一致）

---

## 4. 測試案例總覽

| 編號 | 名稱                                             | 類型           |
| ---- | ------------------------------------------------ | -------------- |
| T01  | 使用者登入成功                                   | Auth           |
| T02  | 使用者登入失敗                                   | Auth           |
| T03  | 無 Token 呼叫保護 API                            | RBAC           |
| T04  | 一般使用者無法呼叫 `/api/admin/*`                | RBAC           |
| T05  | Admin 觸發單日 Ingestion                         | Ingestion      |
| T06  | Analyst 觸發單日 Ingestion                       | Ingestion      |
| T07  | User 不能觸發 Ingestion                          | RBAC           |
| T08  | Admin 觸發單日 Analysis                          | Analysis       |
| T09  | Analysis 未完成前的查詢會失敗                    | Analysis       |
| T10  | 查詢指定日期分析結果                             | Analysis Query |
| T11  | Screener API 回傳正確格式                        | Screener       |
| T12  | Screener 條件正確套用                            | Screener       |
| T13  | Screener 排序正確                                | Screener       |
| T14  | 強勢股為空結果時可正常回傳                       | Screener       |
| T15  | Ingestion → Analysis → Query → Screener 完整 E2E | End-to-End     |

以下為每個測試案例的詳細內容。

---

## 5. 測試案例詳細內容

---

### T01：使用者登入成功

**前置條件**：帳號存在  
**步驟**：  
1. `POST /api/auth/login`  
2. 帶入 `user@example.com` / `password123`  

**預期結果**：  
- 回傳 `access_token`  
- token 格式正確  
- `success = true`

---

### T02：使用者登入失敗

**步驟**：  
- 使用不存在帳號／錯誤密碼登入

**預期結果**：  
- 401  
- `error_code = AUTH_INVALID_CREDENTIALS`

---

### T03：無 Token 呼叫保護 API

**步驟**：  
- `GET /api/analysis/daily?trade_date=2025-12-01` 不帶 token  

**預期結果**：  
- 回傳 401  
- `error_code = AUTH_UNAUTHORIZED`

---

### T04：一般使用者無法呼叫 `/api/admin/*`

**前置條件**：取得 user token  
**步驟**：  
- `POST /api/admin/analysis/daily`  

**預期結果**：  
- 回傳 403  
- `error_code = AUTH_FORBIDDEN`

---

### T05 / T06：Admin / Analyst 可觸發單日 Ingestion

**步驟**：  
- 使用 admin 或 analyst token 呼叫：  
  `POST /api/admin/ingestion/daily`  
  body: `{ "trade_date": "2025-12-01" }`

**預期結果**：  
- 回傳 `success = true`  
- `total_stocks > 0`  
- `failure_count >= 0`  
- `daily_prices` 表內有寫入資料

---

### T07：User 不能觸發 Ingestion

**預期結果**：403 Forbidden

---

### T08：Admin 觸發單日 Analysis

**步驟**：  
- `POST /api/admin/analysis/daily {"trade_date":"2025-12-01"}`

**預期結果**：  
- 回傳成功  
- `analysis_results` 有資料  
- 每檔股票至少有：  
  - `close_price`  
  - `return_5d`（若不足 5 日，允許記為失敗）  
  - `volume_ratio`  
  - `score`

---

### T09：Analysis 未完成前查詢會失敗

**步驟**：  
1. 先觸發 ingestion  
2. 不執行 analysis，直接呼叫：  
   `/api/analysis/daily?trade_date=2025-12-01`

**預期結果**：  
- 404 或自定義錯誤碼：`ANALYSIS_NOT_READY`

---

### T10：查詢分析結果 API 正確運作

**預期結果**：  
- 回傳 total_count  
- 每筆 item 至少包含：  
  - `stock_code`  
  - `close_price`  
  - `return_5d`  
  - `volume_ratio`  
  - `score`

---

### T11：Screener API 回傳正確格式

**步驟**：  
- `/api/screener/strong-stocks?trade_date=2025-12-01`

**預期結果**：  
- 回傳 items（可能為空）  
- 每筆 item 欄位完整

---

### T12：Screener 條件正確套用

條件：  
- `score >= score_min`  
- `return_5d > 0`  
- `volume_ratio >= volume_ratio_min`  
- `change_percent >= 0`

**測試方式**：  
- 對 DB 手動插入可控的 `analysis_results` 假資料  
- 呼叫 Screener API  

**預期結果**：  
- 僅符合條件的股票被回傳  
- 不符合條件的股票必須排除

---

### T13：Screener 排序正確

**預期結果**：  
- 預期排序：score DESC → return_5d DESC  
- 確認回傳列表排序無誤

---

### T14：強勢股為空時應回傳空陣列

**預期結果**：  
- `items: []`  
- 不可回傳錯誤

---

### T15：完整 E2E 流程測試（重點案例）

**步驟**：

1. Admin 登入 → 取得 token  
2. `POST /api/admin/ingestion/daily`  
3. `POST /api/admin/analysis/daily`  
4. user 登入 → 取得 token  
5. `GET /api/analysis/daily`  
6. `GET /api/screener/strong-stocks`

**預期結果**：

- 所有步驟成功  
- 查詢 API 有資料  
- Screener 回傳結果格式正確

---

## 6. 非功能性測試（MVP）

### 6.1 基礎效能檢查

- Screener API 第一次呼叫應在可接受延遲內（例如 < 300ms）。
- 分析結果查詢在 limit=100 時應快速回應。

### 6.2 正常錯誤處理

- 任意 API 出錯需回傳 `success = false` 與正確 `error_code`。

---

## 7. 驗收標準

MVP 被視為完成必須：

1. 所有 **T01–T15 測試案例皆通過**。  
2. 端到端流程完全可用。  
3. 數據與邏輯符合規格定義。  
4. 全部保護 API 皆依權限正確判斷。  

---
