# Golang × DDD × Clean Architecture × 全自動 Codex 開發專案

本專案旨在建立一個完全由 Codex 開發與維護的後端系統。人類只負責更新需求文件，所有程式碼變更與測試皆由 Codex 依據文件自動完成。

## 目錄
- 專案目標與定位
- 技術棧與設計原則
- 專案目錄結構
- 開發流程（人類 × Codex 分工）
- 需求文件模板
- 測試策略
- Coding Style 與慣例
- 未來擴充規範
- 給 Codex 的總結指令

## 1. 專案目標與定位
- 使用 Golang 為主要開發語言，採用 DDD 與 Clean Architecture。
- Domain 與 Application 層需具備完善、可維護的單元測試。
- 人類唯一工作：維護 /docs 內的需求與規格文件；禁止直接修改 /internal 程式碼。
- Codex 依據需求文件與本 README 進行開發、調整與測試。

## 2. 技術棧與設計原則
### 2.1 技術棧
- 語言：Golang（>= 1.22，實際版本於 repo 中更新）
- 依賴管理：Go Modules
- 測試：`go test`、`testify`（可依需求擴充）
- 架構：DDD + Clean Architecture（Hexagonal/Onion 風格）
- 格式化與靜態檢查：
  - `go fmt ./...`
  - `go vet ./...`
  - （選用）`golangci-lint run`

### 2.2 設計原則
- **Domain-First**：業務邏輯與規則皆在 domain 層定義，且不依賴 framework 或外部技術細節。
- **Clean Architecture 依賴方向**：
  - domain 不依賴任何其他層
  - application 依賴 domain
  - interface/adapter 依賴 application
  - infrastructure 實作各種介面（DB、外部 API 等），由外向內注入
- **測試優先**：
  - Domain 與 Application 層需有完善單元測試
  - 新增需求需先更新/新增對應測試
- **自動化開發流程**：
  - 禁止人類直接修改 `/internal` 下的程式碼
  - 人類僅透過更新 `/docs` 內容驅動 Codex 開發

## 3. 專案目錄結構
預期骨架，Codex 需依此維護與擴充：

```
.
├── cmd/
│   └── api/
│       └── main.go           # 進入點：組合 DI、啟動 HTTP Server 等
│
├── internal/
│   ├── domain/               # DDD Domain Layer（不依賴其他層）
│   │   ├── <bounded_context>/
│   │   │   ├── entity.go     # 實體（Aggregate, Entity, Value Object）
│   │   │   ├── vo.go         # Value Objects
│   │   │   ├── service.go    # Domain service（若需要）
│   │   │   └── repo.go       # repository 介面定義（port）
│   │   └── shared/           # 共用 domain 類型（錯誤、共用 VO）
│   │
│   ├── application/          # Use Case/Application Layer
│   │   ├── <bounded_context>/
│   │   │   ├── usecase.go    # 用例（Application services）
│   │   │   └── dto.go        # Input/Output DTO
│   │   └── shared/
│   │
│   ├── interface/            # Interface / Adapter（Controller, Presenter, CLI）
│   │   └── http/
│   │       ├── router.go     # 路由設定
│   │       └── handler/
│   │           ├── <bounded_context>_handler.go
│   │           └── response_mapper.go
│   │
│   ├── infrastructure/       # DB / 外部系統 / Framework 實作
│   │   ├── persistence/
│   │   │   ├── <bounded_context>_repository.go
│   │   ├── external/
│   │   │   └── <service>_client.go
│   │   └── config/
│   │       └── config.go
│   │
│   └── pkg/                  # 可抽取的共用工具（log, error, util...）
│
├── docs/
│   ├── README.md             # Doc 使用說明（給人類）
│   ├── architecture.md       # 架構說明（補充本檔案）
│   ├── glossary.md           # 名詞定義與 ubiquitous language
│   ├── requirements/         # 業務需求與規格文件（唯一開發來源）
│   │   ├── 0000-template.md
│   │   ├── 0001-...md
│   │   └── ...
│   └── decisions/
│       └── ADR-0001-...md    # 架構/技術決策紀錄
│
├── test/
│   ├── integration/          # 整合測試（需啟用實際 infra）
│   └── e2e/                  # 端到端測試（選用）
│
├── Makefile                  # 開發與測試指令（給 Codex 使用）
├── go.mod
├── go.sum
└── README.md                 # 本檔案
```

## 4. 開發流程（人類 × Codex 分工）
### 4.1 人類的工作流程
1) 為新功能/需求新增 Doc：`/docs/requirements/<流水號>-<簡要描述>.md`（例：`0003-create-user.md`）。  
2) 在 Doc 中填寫完整需求（可用模板）。  
3) 將 Doc 提交版本控制。  
4) 後續所有程式碼變更交由 Codex，避免直接改動 `/internal`。  

### 4.2 Codex 的工作流程
每次執行 Codex 時，依序進行：
1) 讀取本 README 理解架構與規範。  
2) 讀取 `/docs/architecture.md`、`/docs/glossary.md` 等架構與名詞說明。  
3) 掃描 `/docs/requirements/` 需求文件，分析新增或修改的需求：Bounded Context、Aggregates/Entities/Value Objects、Use Cases、外部介面。  
4) 更新 `internal/domain` 的 Domain 模型與介面；更新 `internal/application` 的 Use Case 與 DTO。  
5) 在 `internal/infrastructure` 實作/調整 Repository、Client 等。  
6) 在 `internal/interface/http` 更新 Router 與 Handler，依需求文件的 API 規格映射到 Use Case。  
7) 補齊或更新必要的單元測試（Domain 不依賴 infra；Application 使用 fake repo/client）。  
8) 確保以下指令全部成功：
   - `go fmt ./...`
   - `go vet ./...`
   - `go test ./...`

### 4.3 Commit 原則
- Codex（AI）可自行建立 commit，涵蓋依需求文件產出的程式碼與測試。
- 人類負責審閱與 `git push` 到遠端（origin），避免 AI 直接推送。
- 人類仍可就 `/docs` 內的需求或架構更新建立 commit，但請勿直接提交 `/internal` 變更。
- 建議 commit message 簡潔描述目的，例如：`docs: 新增 0003-create-user 需求`、`feat: 實作 user create usecase` 或 `chore: 更新架構說明`。

## 5. 需求文件模板（位於 `/docs/requirements/0000-template.md`）
```
# 功能名稱：<對人類友善的名稱>

## 1. 背景與目標
- 說明為什麼需要此功能
- 說明預期解決的問題與效益

## 2. Domain 與名詞定義
- Domain 名稱 / Bounded Context
- 相關名詞與定義（若與《glossary.md》衝突，以 glossary 為準）

## 3. 使用情境（Use Cases）
### 3.1 Use Case A
- 角色：
- 前置條件：
- 主流程：
- 例外流程：

### 3.2 Use Case B
...

## 4. Domain 模型草稿
- Aggregates / Entities / Value Objects 描述
- 重要欄位與不變條件（Invariants）

## 5. API / 介面規格（若有）
### 5.1 HTTP API 範例
- Method: `POST /v1/users`
- Request JSON：
```json
{
  "name": "string",
  "email": "string"
}
```
- Response JSON：
```json
{
  "id": "string",
  "name": "string",
  "email": "string"
}
```

### 5.2 其他介面（Message Queue, gRPC, Cron 等）
- Topic / Queue Name：
- Payload：
- 觸發條件：

## 6. 驗收條件（Acceptance Criteria）
- 情境 A：當 ... 時，系統必須 ...
- 情境 B：當 ... 時，系統必須 ...
- 單元測試覆蓋以下情境：
  - 成功路徑
  - 主要錯誤類型
  - 邊界條件
```

Codex 必須以需求文件作為 Domain 與 Use Case 的唯一真實來源。

## 6. 測試策略
- `domain` 與 `application` 的所有 public function 需有對應單元測試。
- 測試檔命名：`xxx_test.go`，放在同一 package。
- Domain 層測試不得依賴 DB/HTTP 等基礎建設。
- Application 層透過介面注入 fake/stub repository、client。
- 範例指令：
```bash
go test ./...
```
- 整合測試：`/test/integration`（需實際 infra）
- E2E 測試：`/test/e2e`（選用）

## 7. Coding Style 與慣例
- 函式/方法命名優先表達 domain 概念。
- 介面命名以能力或角色為主：`UserRepository`、`PaymentService`、`Clock` 等。
- 錯誤處理：使用 `error`；必要時在 domain 定義 domain-level errors，避免依賴 framework-specific error 型別。
- Context 傳遞：進入點（HTTP handler、CLI）建立 `context.Context`；Application/Infrastructure 可用 context；Domain 層避免直接持有 context。

## 8. 未來擴充規範
- 調整架構或 DDD 邊界時，人類不得直接修改程式碼。
- 必須更新 `/docs/architecture.md` 或新增 ADR（`/docs/decisions/ADR-xxxx-...md`）。
- Codex 下次執行時進行計畫性重構：調整目錄、切分 Bounded Context、更新測試。

## 9. 給 Codex 的總結指令（簡化版）
1) 讀取本 README，理解架構與規範。  
2) 讀取 `/docs` 下的架構與需求文件。  
3) 依 DDD + Clean Architecture 更新 `/internal` 內的程式碼、介面與實作。  
4) 確保下列指令全數成功：`go fmt ./...`、`go vet ./...`、`go test ./...`。  
5) 所有邏輯修改以需求文件為唯一真實來源。  
