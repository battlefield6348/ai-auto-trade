---
name: strategy_management
description: 負責維護交易策略的核心架構，包含舊版 JSON 策略與新版計分策略 (Scoring Strategy) 的轉換與持久化。
---

# Strategy Management Skill

本 Skill 旨在協助 AI 理解並維護專案中的策略系統，特別是「舊版 JSON 條件策略」與「新版計分權重策略」並存的架構。

## 1. 核心架構 (Core Architecture)

專案目前處於從 legacy 轉向 scoring 的過渡期：

### 舊版策略 (Legacy Strategy)
- **模型**：`tradingDomain.Strategy` (`internal/domain/trading/model.go`)
- **特點**：買賣條件儲存在 `buy_conditions` 與 `sell_conditions` 的 JSONB 欄位中。
- **用途**：基礎的邏輯判斷（如 MA 交叉、RSI 等）。

### 新版計分策略 (Scoring Strategy)
- **模型**：`strategyDomain.ScoringStrategy` (`internal/domain/strategy/scoring_model.go`)
- **特點**：
    - 將每個條件抽離至 `conditions` 表。
    - 透過 `strategy_rules` 表為每個條件分配「權重 (Weight)」。
    - 設有 `threshold` (進場門檻) 與 `exit_threshold` (出場門檻)。
- **用途**：更進階的 AI 評分機制，支援不同指標的加權計算。

## 2. Source of Truth (資料來源)

雖然資料庫中 `strategies` 表是共用的，但讀取時需注意：
- **Scoring 優先**：在載入策略詳情時，應優先嘗試執行 `LoadScoringStrategyBySlug` 或 `LoadScoringStrategyByID`。
- **回退機制**：若非計分型策略，則回退至讀取 `tradingDomain.Strategy`。

## 3. 重要檔案定位 (File Mapping)

| 職責 | 檔案路徑 |
| :--- | :--- |
| 領域模型 (Scoring) | `internal/domain/strategy/scoring_model.go` |
| 領域模型 (Legacy) | `internal/domain/trading/model.go` |
| 資料庫倉儲 | `internal/infrastructure/persistence/postgres/trading_repo.go` |
| 業務邏輯 | `internal/application/trading/service.go` |
| HTTP 處理器 | `internal/interface/http/strategy_handler.go` |
| 前端載入 | `web/backtest.js` |

## 4. 常見任務流程

- **新增策略欄位**：請參考 `.agent/workflows/update_strategy_fields.md`。
- **修改計分邏輯**：需調整 `internal/interface/http/pipeline.go` 中的 `ExecuteScoringAutoTrade` 及相關評分器 (Evaluators)。

## 5. 注意事項

- **Seed 陷阱**：`internal/interface/http/strategy_seed.go` 會在啟動時檢查預設策略。修改預設策略參數時，需同步更新此處，否則重啟後可能被覆蓋（現已加上 `IF NOT EXISTS` 檢查）。
- **版本控制**：策略每次更新應自動增加 `Version`。
