---
name: mvp_implementation
description: Step-by-step implementation plan for the MVP of the AI Auto Trade project, based on the MVP 实作計畫.
---

# MVP Implementation Plan (MVP實作計畫)

This skill outlines the tasks required to implement the MVP for the AI Auto Trade project. Follow these steps sequentially to build the core functionality.

## 1. Goal

The goal is to complete an end-to-end flow for the **BTC Spot (BTC/USDT)** scenario:

> A logged-in user can call an API to get the "BTC Analysis and Strong Stock Results" for a specific trading date.
> This list is calculated using exchange daily K-line data (default Binance 1d; falls back to synthetic data if failed).

## 2. Relevant Specifications

Before starting, ensure you have read and understood the relevant specification documents in `docs/specs/`:

- `docs/specs/身分驗證與授權.md` (Authentication & Authorization)
- `docs/specs/角色與權限.md` (Roles & Permissions)
- `docs/specs/資料庫結構.md` (Database Schema)
- `docs/specs/資料擷取流程.md` (Data Ingestion - MVP subset)
- `docs/specs/日批次分析.md` (Daily Batch Analysis - MVP subset)
- `docs/specs/查詢與匯出.md` (Query & Export - MVP subset)
- `docs/specs/選股篩選器.md` (Stock Screener - MVP subset)
- `docs/specs/MVP端對端流程.md` (MVP E2E Flow)

## 3. Implementation Phases

Follow these phases in order:

1.  **Project Structure & DB Connection**: Setup basic structure and Postgres connection.
2.  **Auth & RBAC**: Implement minimal authentication and role-based access control.
3.  **Data Models & Repository**: Implement core repositories for `stocks`, `daily_prices`, `analysis_results`.
4.  **Data Ingestion**: Implement single-day data fetching (Binance -> Synthetic fallback).
5.  **Daily Analysis**: Implement single-day batch analysis logic.
6.  **Query API**: Implement API to query analysis results.
7.  **Strong Screener API**: Implement the "Strong Stocks" screener API.
8.  **Error Handling & Verification**: Polish error handling and verify the E2E flow.

---

## 4. Phase 1: Project Structure & DB Connection

### 4.1 Tasks

1.  **Project Setup**:
    - Ensure directories follow DDD + Clean Architecture (domain, application, infrastructure, interfaces).
    - Setup configuration loading (viper or similar).

2.  **DB Module**:
    - Implement database connection using `pgx` or `gorm` (as per project standards).
    - Setup connection pooling and health check.

3.  **Migration Setup**:
    - Create a mechanism to run migrations (e.g., `golang-migrate`).
    - Ensure migrations can be run via CLI or on startup.

---

## 5. Phase 2: Auth & RBAC (Minimal)

### 5.1 Tasks

1.  **Schema Implementation**:
    - Create tables: `users`, `roles`, `user_roles`, `permissions`, `role_permissions`.

2.  **User Model**:
    - Implement user creation (seeding admin/user).

3.  **Login Flow**:
    - Implement `POST /api/auth/login` (Email + Password -> Access Token).

4.  **Middleware**:
    - Implement JWT validation middleware.
    - Extract user ID and roles from token.

5.  **Permission Check**:
    - Implement logic to check if user has required permission for an endpoint.

6.  **Route Protection**:
    - Protect `/api/admin/*` for admin/analyst.
    - Protect `/api/analysis/*`, `/api/screener/*` for user.

---

## 6. Phase 3: Data Models & Repository

### 6.1 Tasks

1.  **Schema & Models**:
    - Create tables: `stocks` (BTC/USDT), `daily_prices`, `analysis_results`.
    - Define Domain Entities and Repository Interfaces.

2.  **Repository Implementation**:
    - `StockRepository`: Find/Save trading pairs.
    - `DailyPriceRepository`: Save/Find daily prices (history & specific date).
    - `AnalysisResultRepository`: Save/Find analysis results (history, date, screening).

---

## 7. Phase 4: Data Ingestion (MVP Single Day)

### 7.1 Tasks

1.  **API Handler**:
    - Implement `POST /api/admin/ingestion/daily` (Input: `trade_date`).

2.  **Logic**:
    - Fetch daily K-line from external source (Binance).
    - If external fails, use synthetic data (if configured).
    - Update `stocks` and `daily_prices` tables.

3.  **Auth**:
    - Require `ingestion.trigger_daily` permission.

---

## 8. Phase 5: Daily Analysis (MVP Single Day)

### 8.1 Tasks

1.  **API Handler**:
    - Implement `POST /api/admin/analysis/daily` (Input: `trade_date`).

2.  **Logic**:
    - Load daily prices for the date.
    - For each stock, load previous 5 days history.
    - Calculate: `close_price`, `change_percent`, `return_5d`, `volume`, `volume_avg_5d`, `score`.
    - Save to `analysis_results`.

3.  **Auth**:
    - Require `analysis.trigger_daily` permission.

---

## 9. Phase 6: Analysis Query API

### 9.1 Tasks

1.  **API Handler**:
    - Implement `GET /api/analysis/daily`.
    - Inputs: `trade_date`, `limit`, `offset`.

2.  **Logic**:
    - Query `analysis_results` joined with `stocks`.
    - Return list with fields: code, name, market, close, change%, return_5d, volume, ratio, score.

3.  **Auth**:
    - Require `analysis_results.query` permission.

---

## 10. Phase 7: Strong Screener API

### 10.1 Tasks

1.  **API Handler**:
    - Implement `GET /api/screener/strong-stocks`.
    - Inputs: `trade_date`, `limit`, `score_min`, `volume_ratio_min`.

2.  **Logic (Fixed Criteria)**:
    - `score >= score_min`
    - `return_5d > 0`
    - `volume_ratio >= volume_ratio_min`
    - `change_percent >= 0`
    - Sort by score desc, then return_5d desc.

3.  **Auth**:
    - Require `screener.use` permission.

---

## 11. Phase 8: Error Handling & Verification

### 11.1 Tasks

1.  **Standardized Errors**:
    - Ensure consistent error responses (code + message).

2.  **Error Cases**:
    - Handle unauthorized, forbidden, data not found, ingestion/analysis running.

3.  **Logging**:
    - Log login attempts, job execution times, screener usage.

---

## Execution Order

1.  Read `docs/specs/*.md` for context.
2.  Implement Phase 1 (Structure & DB).
3.  Implement Phase 2 (Auth).
4.  Implement Phase 3 (Repos).
5.  Implement Phase 4 (Ingestion).
6.  Implement Phase 5 (Analysis).
7.  Implement Phase 6 (Query).
8.  Implement Phase 7 (Screener).
9.  Run End-to-End Tests.
