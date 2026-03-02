---
description: 如何在策略模型中新增或修改欄位
---

為了確保 AI 在修改交易策略 (Strategy) 相關欄位時保持高效且不遺漏任何模組，請遵循以下步驟：

### 1. 更新領域模型 (Domain Model)
檔案路徑：`internal/domain/trading/model.go`
- 在 `Strategy` 結構體中新增欄位。
- 確保包含正確的 `json:"field_name"` 標籤。

### 2. 更新資料庫持久層 (Repository)
檔案路徑：`internal/infrastructure/persistence/postgres/trading_repo.go`
必須更新以下四個方法：
a. `CreateStrategy`: 更新 SQL INSERT 語句及其參數。
b. `UpdateStrategy`: 更新 SQL UPDATE 語句及其參數。
c. `getStrategyByField`: 更新 SQL SELECT 語句及 `Scan` 接收變數。
d. `ListStrategies`: 更新 SQL SELECT 語句及 `Scan` 接收變數。

### 3. 更新服務層邏輯 (Application Service)
檔案路徑：`internal/application/trading/service.go`
- 在 `UpdateStrategy` 方法中，手動將輸入的 `input` 欄位值賦予 `current` 物件，確保更新會被儲存。

### 4. 同步更新測試程式 (Tests)
檔案路徑：`internal/infrastructure/persistence/postgres/trading_repo_test.go`
- 更新所有 `sqlmock.ExpectQuery` 與 `sqlmock.ExpectExec` 的 `WithArgs`。
- 更新 `sqlmock.NewRows` 的欄位定義與 `AddRow` 的模擬資料。

檔案路徑：`internal/application/trading/service_test.go`
- 在 `TestUpdateStrategy_VersionBump` 等測試案例中加入對新欄位的驗證。

### 5. (前端可選) 更新載入邏輯
檔案路徑：`web/backtest.js`
- 在 `loadStrategyDetails` 函數中，檢查回傳的 JSON 並將欄位對應至 UI 元素（如 `total_min` 對應至 `threshold`）。

### 備註：計分策略 (Scoring Strategy)
如果是針對計分規則的欄位（rules/threshold），請同時確認 `internal/domain/strategy/scoring_model.go` 是否需要異動。在 API 層次中，`handleGetStrategyByQuery` 與 `handleStrategyGetOrUpdate` 應優先返回載入完整的計分模型。
