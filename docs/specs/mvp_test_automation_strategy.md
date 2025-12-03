# MVP 自動化測試策略（MVP Test Automation Strategy）

版本：v1.0  
狀態：草稿  
最後更新：2025-12-03  

---

## 1. 文件目的

本文件說明在 MVP 階段，整個系統的自動化測試策略，包含：

- 要有哪些層級的測試（Unit / Integration / E2E）
- 每個層級負責驗證哪些規格
- 測試資料與測試環境如何建置
- 如何在 CI 流程中執行這些測試

本文件配合：

- `/docs/specs/mvp_end_to_end_flow.md`
- `/docs/tests/mvp_e2e_test_plan.md`
- `/docs/specs/database_schema.md`
- `/docs/specs/authentication_authorization.md`
- `/docs/specs/roles_permissions.md`

---

## 2. 測試層級總覽

MVP 階段至少包含三層：

1. 單元測試（Unit Tests）
2. 整合測試（Integration Tests）
3. 端到端測試（End-to-End Tests）

目標分工：

- **Unit**：保證「一小塊邏輯」本身正確  
- **Integration**：保證「模組與 DB / 外部服務的互動」正確  
- **E2E**：保證「使用者故事完整流程」可用  

---

## 3. 單元測試（Unit Tests）

### 3.1 範圍

MVP 階段，單元測試至少涵蓋：

1. **認證邏輯**
   - 密碼驗證
   - Token 產生與解析
   - 角色 → 權限 mapping 判斷

2. **分析計算邏輯（Daily Analysis 核心）**
   - `return_5d` 計算（給定一組價格序列 → 應得報酬）
   - `volume_avg_5d`、`volume_ratio`
   - `score` 計算公式（給定一組輸入 → 穩定輸出）

3. **Screener 條件判斷**
   - 給定一筆分析結果，判斷是否符合「強勢股」條件：
     - score 門檻
     - return_5d 正值
     - volume_ratio 門檻
     - change_percent 非負

### 3.2 測試原則

- 單元測試不連線真實 DB  
- 對 Repository 介面使用 mock / stub  
- 對純邏輯函式（分析公式）做「輸入 → 輸出」驗證

---

## 4. 整合測試（Integration Tests）

### 4.1 範圍

整合測試需涵蓋：

1. **Repository 與 DB 的互動**
   - `stocks` / `daily_prices` / `analysis_results` 寫入與查詢
   - 主鍵與唯一約束是否符合預期
   - 依日期／股票查詢是否正確

2. **Ingestion → DB 寫入**
   - 用假資料模擬外部來源，檢查寫入 `daily_prices` 的結果

3. **Analysis → DB 寫入**
   - 給定 `daily_prices` 資料，執行分析：
     - 應產生正確的 `analysis_results` 記錄
     - 欄位值是否與單元測試公式一致

4. **Auth + RBAC 與 DB**
   - 登入流程實際讀取 `users`、`roles`、`user_roles` 等表
   - 權限查詢與判斷實際走 DB 資料

### 4.2 測試環境

- 使用「測試專用 DB」：
  - 可使用 docker 啟動 PostgreSQL（可共用目前本地 docker compose，但使用測試 database）
- 每個整合測試：
  - 需在開始前清理相關資料表（或跑 migration + seeding）
  - 測試結束後可選擇回滾或 truncate

---

## 5. 端到端測試（E2E Tests）

### 5.1 範圍

完整 E2E 測試需對應：

- `/docs/tests/mvp_e2e_test_plan.md` 中 T01–T15  
- 包含實際 HTTP 請求，從入口 API 一路跑完整服務層與 DB

### 5.2 實作方式建議

- 啟動整個服務（可用 docker compose + app service）  
- 使用 HTTP 客戶端（測試框架）對 API 發請求  
- 在測試前：
  - 執行 DB migration
  - Seed 初始使用者、角色、權限資料
- 在測試中：
  - 嚴格按 test plan 執行登入 → ingestion → analysis → query → screener 流程
- E2E 測試不 mock DB，也不 mock 核心服務，只可 mock 外部資料來源（市場所屬、券商 API 等）

---

## 6. 測試資料策略

### 6.1 基礎 Seed 資料

在測試環境中，應在 migration 後執行一次 seed，至少包含：

- 角色：
  - `admin` / `analyst` / `user`
- 權限：
  - MVP 所需的最小權限，如：
    - `analysis_results.query`
    - `screener.use`
    - `ingestion.trigger_daily`
    - `analysis.trigger_daily`
- 使用者：
  - `admin@example.com`
  - `analyst@example.com`
  - `user@example.com`
- 角色綁定：
  - admin → admin 角色
  - analyst → analyst 角色
  - user → user 角色

### 6.2 市場資料測試樣本

為避免測試依賴真實外部 API，MVP 測試可採：

1. **Unit / Integration：**
   - 直接插入小量 `stocks` / `daily_prices` 假資料，例如 3–5 檔股票，10 個交易日。

2. **E2E：**
   - Ingestion 流程可使用「可切換實做」：
     - 正式環境：呼叫真實外部 API
     - 測試環境：從固定 JSON / 檔案讀取模擬資料

---

## 7. 測試環境與 CI 流程

### 7.1 本地開發測試流程

建議標準流程：

1. 啟動 Postgres（依 `docker-compose.yml`）  
2. 執行 migrations + seed  
3. 執行測試：
   - 單元測試
   - 整合測試（可選）
   - E2E 測試（可選）

### 7.2 CI 產線前流程

在 CI pipeline 中，建議步驟順序：

1. Checkout 專案  
2. 建立測試用 Postgres（使用 docker service）  
3. 執行 migrations  
4. 執行 seed（建立角色、權限、測試使用者）  
5. 執行單元測試  
6. 執行整合測試  
7. 執行 E2E 測試（可標記為較重的 stage，但 MVP 建議仍然要跑）

所有測試通過才允許：

- 合併到主要分支  
- 進行部署

---

## 8. 失敗處理與除錯

### 8.1 測試失敗的基本要求

當任一層測試失敗時，需能：

- 從測試輸出中知道：
  - 失敗案例編號（對應 test plan，例如 T08）
  - 失敗請求的 API 路徑與參數
  - 實際回應內容（status code + body）
- 有基本 log（至少在 console / 檔案）對應該次請求與 server error

### 8.2 最小 Log 要求

- 登入成功／失敗
- Ingestion / Analysis API 的開始與結束、目標日期、成功／失敗統計
- Screener API 的呼叫，包含：
  - trade_date
  - 實際使用的 `score_min`、`volume_ratio_min`
  - 命中筆數

---

## 9. 覆蓋率目標（MVP）

MVP 階段不追求極端高的覆蓋率，但需有合理保障：

- Unit Test：核心分析邏輯與認證邏輯覆蓋率應 ≥ 70%（行為建議，可在報表中觀察）  
- Integration + E2E：對「MVP 最關鍵路徑」必須有測試覆蓋，即 `/docs/tests/mvp_e2e_test_plan.md` 所列全數案例  

---

## 10. 與未來測試擴充的關係

本文件僅針對：

- MVP 端到端流程  
- 資料抓取／分析／查詢／選股  

未來新增模組（如通知、策略、Dashboard）時，需再新增：

- 各模組對應的 Unit / Integration / E2E 測試規劃文件  
- 並沿用本文件定義的測試層級與資料策略

---
