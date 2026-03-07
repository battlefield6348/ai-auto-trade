# 策略參數優化器 (Strategy Optimizer) UI 設計規格

本文件旨在提供給 Stitch 或前端開發人員，用於生成「自動策略優化 (Auto-Optimization)」的介面。

## 1. 功能概述
此功能允許使用者針對特定交易對，透過後端 API 自動尋找過去一段時間內表現最優的策略參數組合（進場門檻、停利停損、條件權重等），並能一鍵儲存並部署該策略。

## 2. API 介面參考
- **Endpoint**: `POST /api/admin/strategies/optimize`
- **Payload 範例**:
  ```json
  {
    "symbol": "BTCUSDT",
    "days": 90,
    "save_top": true
  }
  ```
- **Response 範例**:
  ```json
  {
    "success": true,
    "result": {
      "best_strategy": { "name": "...", "threshold": 70, ... },
      "total_return": 45.2,
      "win_rate": 65.0,
      "total_trades": 18
    }
  }
  ```

## 3. 介面區塊建議 (UI Mockup Elements)

### A. 設定面板 (Control Panel)
- **交易對選擇 (Select Symbol)**: 下拉選單或輸入框 (預設 `BTCUSDT`)。
- **追溯天數 (Lookback Days)**: 滑動條或輸入框 (60, 90, 180 天)。
- **自動部署開關 (Direct Deployment)**: 一個 Toggle 開關，對應 `save_top` 參數。若開啟，優化完成後直接啟用策略。
- **開始優化按鈕**: 點擊後觸發 API。

### B. 優化中狀態 (Processing State)
- **進度條/讀取動畫**: 由於優化需進行數百次排列組合的回測，建議顯示「正在運算最優參數組合...」的動畫。

### C. 結果展示區 (Results Display) - *最重要區塊*
當 API 回傳後，顯示最優策略的「成績單」：
- **核心指標**:
  - 總報酬率 (Total Return) - 醒目的百分比顯示。
  - 勝率 (Win Rate)。
  - 總交易次數 (Total Trades)。
- **建議參數**:
  - 進場分數門檻 (Entry Threshold)。
  - 出場分數門檻 (Exit Threshold)。
  - 停利點 (Take Profit)。
  - 停損點 (Stop Loss)。
- **視覺化**: 建議使用簡易的卡片 (Card) 設計，區分「當前設定」與「優化建議設定」的差異。

## 4. 設計風格建議
- **科技感**: 由於是人工智慧/自動優化功能，建議使用帶有微光效果 (Glow) 的卡片設計。
- **動態回饋**: 優化成功時，顯示一個「Successfully Optimized & Deployed」的 Toast 訊息。

## 5. 互動邏輯
1. 使用者選擇 BTCUSDT 並設定 90 天，按下「Run Optimizer」。
2. API 呼叫期間，按鈕進入 Loading 狀態。
3. 成功後，下方滑出 (Slide up) 結果面板，顯示報酬率 45.2%。
4. 若 `save_top` 為真，自動提示「新策略已儲存並啟動」。
