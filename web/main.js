const state = {
  token: "",
  currentSection: "overview",
  health: null,
  lastIngestion: null,
  lastAnalysis: null,
  lastSummary: null,
  lastBackfill: null,
  lastChart: null,
  strategies: [],
  strategyBacktests: [],
  strategyBacktestsAll: [],
  strategyBacktestPage: { page: 1, size: 10 },
  lastStrategyBacktest: null,
  lastRunTrades: [],
  reports: [],
  selectedReport: null,
  activity: [],
  updatedAt: null,
};

const elements = {
  status: document.getElementById("status"),
  healthStatus: document.getElementById("healthStatus"),
  overviewTime: document.getElementById("overviewTime"),
  overviewMode: document.getElementById("overviewMode"),
  loginMessage: document.getElementById("loginMessage"),
  resetZoomBtn: document.getElementById("resetZoomBtn"),
  chartForm: document.getElementById("chartForm"),
  chartStart: document.getElementById("chartStart"),
  chartEnd: document.getElementById("chartEnd"),
  chartMeta: document.getElementById("chartMeta"),
  chartCanvas: document.getElementById("chartCanvas"),
  chartTooltip: document.getElementById("chartTooltip"),
  backtestForm: document.getElementById("backtestForm"),
  backtestSummary: document.getElementById("backtestSummary"),
  backtestEvents: document.getElementById("backtestEvents"),
  btConditionSelect: document.getElementById("btConditionSelect"),
  btSelectedConditions: document.getElementById("btSelectedConditions"),
  summaryView: document.getElementById("summaryView"),
  activityList: document.getElementById("activityList"),
  strategyForm: document.getElementById("strategyForm"),
  strategyStatus: document.getElementById("strategyStatus"),
  strategyEnv: document.getElementById("strategyEnv"),
  strategyName: document.getElementById("strategyName"),
  strategyTable: document.getElementById("strategyTable"),
  strategyMeta: document.getElementById("strategyMeta"),
  strategyRunResult: document.getElementById("strategyRunResult"),
  tradeForm: document.getElementById("tradeForm"),
  tradeStrategyId: document.getElementById("tradeStrategyId"),
  tradeEnv: document.getElementById("tradeEnv"),
  tradeStart: document.getElementById("tradeStart"),
  tradeEnd: document.getElementById("tradeEnd"),
  tradeTable: document.getElementById("tradeTable"),
  tradeMeta: document.getElementById("tradeMeta"),
  reportForm: document.getElementById("reportForm"),
  reportStrategyId: document.getElementById("reportStrategyId"),
  reportTable: document.getElementById("reportTable"),
  reportMeta: document.getElementById("reportMeta"),
  reportDetail: document.getElementById("reportDetail"),
  positionForm: document.getElementById("positionForm"),
  positionEnv: document.getElementById("positionEnv"),
  positionTable: document.getElementById("positionTable"),
  positionMeta: document.getElementById("positionMeta"),
  logForm: document.getElementById("logForm"),
  logStrategyId: document.getElementById("logStrategyId"),
  logEnv: document.getElementById("logEnv"),
  logLimit: document.getElementById("logLimit"),
  logTable: document.getElementById("logTable"),
  logMeta: document.getElementById("logMeta"),
  strategyBacktestForm: document.getElementById("strategyBacktestForm"),
  strategyBacktestSelect: document.getElementById("strategyBacktestSelect"),
  strategyBacktestId: document.getElementById("strategyBacktestId"),
  strategyBtStart: document.getElementById("strategyBtStart"),
  strategyBtEnd: document.getElementById("strategyBtEnd"),
  strategyBtEquity: document.getElementById("strategyBtEquity"),
  strategyBtPriceMode: document.getElementById("strategyBtPriceMode"),
  strategyBtFees: document.getElementById("strategyBtFees"),
  strategyBtSlippage: document.getElementById("strategyBtSlippage"),
  strategyBtStop: document.getElementById("strategyBtStop"),
  strategyBtTake: document.getElementById("strategyBtTake"),
  strategyBtDailyLoss: document.getElementById("strategyBtDailyLoss"),
  strategyBtCoolDown: document.getElementById("strategyBtCoolDown"),
  strategyBtMinHold: document.getElementById("strategyBtMinHold"),
  strategyBtMaxPos: document.getElementById("strategyBtMaxPos"),
  strategyBacktestReload: document.getElementById("strategyBacktestReload"),
  strategyBacktestMessage: document.getElementById("strategyBacktestMessage"),
  strategyBacktestSummary: document.getElementById("strategyBacktestSummary"),
  strategyBacktestTrades: document.getElementById("strategyBacktestTrades"),
  strategyBacktestHistory: document.getElementById("strategyBacktestHistory"),
  strategyBtHistStart: document.getElementById("strategyBtHistStart"),
  strategyBtHistEnd: document.getElementById("strategyBtHistEnd"),
  strategyBtHistVersion: document.getElementById("strategyBtHistVersion"),
  strategyBtHistPageSize: document.getElementById("strategyBtHistPageSize"),
  strategyBtHistPrev: document.getElementById("strategyBtHistPrev"),
  strategyBtHistNext: document.getElementById("strategyBtHistNext"),
  strategyBtHistPageInfo: document.getElementById("strategyBtHistPageInfo"),
  strategyBacktestFilterForm: document.getElementById("strategyBacktestFilterForm"),
  exportBtTrades: document.getElementById("exportBtTrades"),
  exportBtEquity: document.getElementById("exportBtEquity"),
  strategyBtGoReports: document.getElementById("strategyBtGoReports"),
  strategyEquityChart: document.getElementById("strategyEquityChart"),
  createStrategyForm: document.getElementById("createStrategyForm"),
  createStrategyName: document.getElementById("createStrategyName"),
  createStrategySymbol: document.getElementById("createStrategySymbol"),
  createStrategyTimeframe: document.getElementById("createStrategyTimeframe"),
  createStrategyEnv: document.getElementById("createStrategyEnv"),
  buyLogic: document.getElementById("buyLogic"),
  sellLogic: document.getElementById("sellLogic"),
  buyConditions: document.getElementById("buyConditions"),
  sellConditions: document.getElementById("sellConditions"),
  addBuyCondition: document.getElementById("addBuyCondition"),
  addSellCondition: document.getElementById("addSellCondition"),
  orderSizeMode: document.getElementById("orderSizeMode"),
  orderSizeValue: document.getElementById("orderSizeValue"),
  priceMode: document.getElementById("priceMode"),
  feesPct: document.getElementById("feesPct"),
  slippagePct: document.getElementById("slippagePct"),
  stopLossPct: document.getElementById("stopLossPct"),
  takeProfitPct: document.getElementById("takeProfitPct"),
  coolDownDays: document.getElementById("coolDownDays"),
  minHoldDays: document.getElementById("minHoldDays"),
  maxPositions: document.getElementById("maxPositions"),
  createStrategyPreview: document.getElementById("createStrategyPreview"),
  createStrategyMessage: document.getElementById("createStrategyMessage"),
  loadStrategyTemplate: document.getElementById("loadStrategyTemplate"),
};

const sections = Array.from(document.querySelectorAll("[data-section]"));
const navLinks = Array.from(document.querySelectorAll("[data-section-target]"));

const numberFormat = new Intl.NumberFormat("zh-TW", { maximumFractionDigits: 3 });
const intFormat = new Intl.NumberFormat("zh-TW");
const priceFormat = new Intl.NumberFormat("zh-TW", {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});
const percentFormat = new Intl.NumberFormat("zh-TW", {
  style: "percent",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});
const scoreFormat = new Intl.NumberFormat("zh-TW", {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
});
const timeFormat = new Intl.DateTimeFormat("zh-TW", {
  dateStyle: "medium",
  timeStyle: "short",
  timeZone: "Asia/Taipei",
});

const api = async (path, options = {}) => {
  const headers = { ...(options.headers || {}) };
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  if (options.body && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }
  const res = await fetch(path, { ...options, headers, credentials: "include" });
  const data = await res.json().catch(() => ({}));
  if (res.status === 401) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      return api(path, options);
    }
  }
  if (!res.ok || data.success === false) {
    const msg = data.error || res.statusText;
    throw new Error(`${res.status} ${data.error_code || ""} ${msg}`.trim());
  }
  return data;
};

const fmtText = (v) => (v === null || v === undefined || v === "" ? "—" : v);
const fmtNumber = (v) => (v === null || v === undefined ? "—" : numberFormat.format(v));
const fmtInt = (v) => (v === null || v === undefined ? "—" : intFormat.format(v));
const fmtPrice = (v) => (v === null || v === undefined ? "—" : priceFormat.format(v));
const fmtPercent = (v) => (v === null || v === undefined ? "—" : percentFormat.format(v));
const fmtScore = (v) => (v === null || v === undefined ? "—" : scoreFormat.format(v));
const fmtRatio = (v) => (v === null || v === undefined ? "—" : `${fmtNumber(v)}x`);

const deltaClass = (v) => (v > 0 ? "up" : v < 0 ? "down" : "flat");

const chartState = {
  rows: [],
  points: [],
  fullRows: [],
  fullRes: null,
  window: { start: 0, end: 0 },
  events: [],
  padding: null,
  plotWidth: 0,
  plotHeight: 0,
  step: 0,
  focusLine: null,
  focusDot: null,
};

const backtestSelections = {
  conditions: ["change", "volume", "return", "ma"],
};

const refreshConditionOptions = () => {
  if (!elements.btConditionSelect) return;
  const options = [
    { value: "change", label: "日漲跌條件" },
    { value: "volume", label: "量能條件" },
    { value: "return", label: "近 5 日報酬條件" },
    { value: "ma", label: "均線乖離條件" },
  ];
  const available = options.filter((opt) => !backtestSelections.conditions.includes(opt.value));
  const placeholder = available.length ? "選擇條件…" : "已套用所有條件";
  elements.btConditionSelect.innerHTML = `<option value="">${placeholder}</option>${available
    .map((opt) => `<option value="${opt.value}">${opt.label}</option>`)
    .join("")}`;
  elements.btConditionSelect.disabled = available.length === 0;
};

const mapTrend = (trend) => {
  if (trend === "bullish") return { label: "偏多", className: "up", tone: "good" };
  if (trend === "bearish") return { label: "偏空", className: "down", tone: "warn" };
  return { label: "中性", className: "flat", tone: "warn" };
};

const setStatus = (msg, tone) => {
  elements.status.textContent = msg;
  elements.status.classList.remove("good", "warn");
  if (tone) elements.status.classList.add(tone);
};

const toggleResetZoom = (show) => {
  if (elements.resetZoomBtn) elements.resetZoomBtn.classList.toggle("hidden", !show);
};

const refreshAccessToken = async () => {
  try {
    const res = await fetch("/api/auth/refresh", { method: "POST", credentials: "include" });
    const data = await res.json().catch(() => ({}));
    if (!res.ok || data.success === false || !data.access_token) {
      throw new Error(data.error || res.statusText);
    }
    state.token = data.access_token;
    setStatus("已登入（自動續期）", "good");
    toggleProtectedSections(true);
    return true;
  } catch (_) {
    state.token = "";
    toggleProtectedSections(false);
    return false;
  }
};

const setHealthStatus = (msg, tone) => {
  elements.healthStatus.textContent = msg;
  elements.healthStatus.classList.remove("good", "warn");
  if (tone) elements.healthStatus.classList.add(tone);
};

const setMessage = (el, msg, tone) => {
  el.textContent = msg || "";
  el.classList.remove("good", "error");
  if (tone) el.classList.add(tone);
};

const touchUpdatedAt = () => {
  state.updatedAt = new Date();
  if (elements.overviewTime) {
    elements.overviewTime.textContent = `最後更新：${timeFormat.format(state.updatedAt)}`;
  }
};

const toggleProtectedSections = (isAuth) => {
  const loginPage = document.getElementById("loginPage");
  const appShell = document.getElementById("appShell");
  if (loginPage) loginPage.classList.toggle("hidden", isAuth);
  if (appShell) appShell.classList.toggle("hidden", !isAuth);
};

const showSection = (section) => {
  state.currentSection = section;
  sections.forEach((el) => {
    const target = el.dataset.section;
    el.classList.toggle("section-hidden", target !== section);
  });
  navLinks.forEach((link) => {
    link.classList.toggle("active", link.dataset.sectionTarget === section);
  });
  if (section === "strategy" && state.strategies.length === 0 && elements.strategyForm) {
    loadStrategies().catch((err) => setStatus(`載入策略失敗：${err.message}`, "warn"));
  }
};

const updateOverviewMode = () => {
  if (!elements.overviewMode) return;
  if (!state.health) {
    elements.overviewMode.textContent = "資料來源：--";
    return;
  }
  const source = state.health.use_synthetic ? "合成日 K" : "Binance 日 K";
  elements.overviewMode.textContent = `資料來源：${source}`;
};

const setKpi = (id, { title, value, note, tone }) => {
  const el = document.getElementById(id);
  if (!el) return;
  el.innerHTML = `
    <div class="kpi-title">${title}</div>
    <div class="kpi-value ${tone || ""}">${value}</div>
    <div class="kpi-note">${note}</div>
  `;
};

const renderKpis = () => {
  if (state.health) {
    const dbStatus = String(state.health.db || "").toLowerCase();
    const ok = dbStatus === "ok" || dbStatus === "healthy";
    setKpi("kpiHealth", {
      title: "系統狀態",
      value: ok ? "運作中" : "需注意",
      note: `DB ${state.health.db || "?"} · 合成 ${state.health.use_synthetic ? "ON" : "OFF"}`,
      tone: ok ? "good" : "warn",
    });
  } else {
    setKpi("kpiHealth", {
      title: "系統狀態",
      value: "尚未連線",
      note: "等待健康檢查",
    });
  }

  if (state.lastIngestion) {
    const failed = Number(state.lastIngestion.failure_count || 0) > 0;
    const total = state.lastIngestion.total_stocks ?? state.lastIngestion.success_count;
    const success = state.lastIngestion.success_count ?? "—";
    setKpi("kpiIngestion", {
      title: "日 K 擷取",
      value: `${fmtInt(success)} / ${fmtInt(total)}`,
      note: `交易日 ${fmtText(state.lastIngestion.trade_date)} · 失敗 ${fmtInt(
        state.lastIngestion.failure_count
      )}`,
      tone: failed ? "warn" : "good",
    });
  } else {
    setKpi("kpiIngestion", {
      title: "日 K 擷取",
      value: "自動排程/回補",
      note: "由自動排程或歷史回補處理",
    });
  }

  if (state.lastAnalysis) {
    const failed = Number(state.lastAnalysis.failure_count || 0) > 0;
    const total = state.lastAnalysis.total_stocks ?? state.lastAnalysis.success_count;
    const success = state.lastAnalysis.success_count ?? "—";
    setKpi("kpiAnalysis", {
      title: "日批次分析",
      value: `${fmtInt(success)} / ${fmtInt(total)}`,
      note: `交易日 ${fmtText(state.lastAnalysis.trade_date)} · 失敗 ${fmtInt(
        state.lastAnalysis.failure_count
      )}`,
      tone: failed ? "warn" : "good",
    });
  } else {
    setKpi("kpiAnalysis", {
      title: "日批次分析",
      value: "尚未執行",
      note: "交易日 --",
    });
  }

  if (state.lastSummary) {
    const trend = mapTrend(state.lastSummary.trend);
    setKpi("kpiSummary", {
      title: "趨勢摘要",
      value: trend.label,
      note: `交易日 ${fmtText(state.lastSummary.trade_date)} · ${fmtText(
        state.lastSummary.trading_pair
      )}`,
      tone: trend.tone,
    });
  } else {
    setKpi("kpiSummary", {
      title: "趨勢摘要",
      value: "等待更新",
      note: "--",
    });
  }

  if (state.lastScreener) {
    const count = state.lastScreener.total_count ?? state.lastScreener.items?.length ?? 0;
    setKpi("kpiScreener", {
      title: "強勢交易對",
      value: `${fmtInt(count)} 檔`,
      note: `交易日 ${fmtText(state.lastScreener.trade_date)}`,
      tone: count > 0 ? "good" : "warn",
    });
  } else {
    setKpi("kpiScreener", {
      title: "強勢交易對",
      value: "尚未查詢",
      note: "--",
    });
  }
};

const renderJobPlaceholder = (container, label) => {
  container.innerHTML = `
    <div class="result-header">
      <div>
        <div class="result-title">${label}</div>
        <div class="result-sub">尚未執行</div>
      </div>
      <span class="badge warn">待執行</span>
    </div>
    <div class="result-message">請先登入並選擇交易日。</div>
  `;
};

const renderJobLoading = (container, label) => {
  container.innerHTML = `
    <div class="result-header">
      <div>
        <div class="result-title">${label}</div>
        <div class="result-sub">執行中</div>
      </div>
      <span class="badge">處理中</span>
    </div>
    <div class="result-message">正在送出請求，請稍候。</div>
  `;
};

const renderJobResult = (container, label, res) => {
  const statusText = res.success ? "完成" : "未完成";
  const badgeClass = res.success ? "good" : "warn";
  container.innerHTML = `
    <div class="result-header">
      <div>
        <div class="result-title">${label}</div>
        <div class="result-sub">交易日 ${fmtText(res.trade_date)}</div>
      </div>
      <span class="badge ${badgeClass}">${statusText}</span>
    </div>
    <div class="result-metrics">
      <div><span>處理筆數</span><strong>${fmtInt(res.total_stocks)}</strong></div>
      <div><span>成功</span><strong>${fmtInt(res.success_count)}</strong></div>
      <div><span>失敗</span><strong>${fmtInt(res.failure_count)}</strong></div>
    </div>
  `;
};

const renderJobError = (container, label, message) => {
  container.innerHTML = `
    <div class="result-header">
      <div>
        <div class="result-title">${label}</div>
        <div class="result-sub">未完成</div>
      </div>
      <span class="badge warn">注意</span>
    </div>
    <div class="result-message error">${message}</div>
  `;
};

const renderChartPlaceholder = (message) => {
  if (!elements.chartCanvas) return;
  renderEmptyState(elements.chartCanvas, message || "尚未載入走勢資料");
  if (elements.chartMeta) elements.chartMeta.innerHTML = "";
  if (elements.chartTooltip) elements.chartTooltip.classList.remove("show");
  if (elements.backtestSummary) elements.backtestSummary.innerHTML = "";
  if (elements.backtestEvents) renderEmptyState(elements.backtestEvents, "尚無回測結果");
};

const renderChartLoading = () => {
  if (!elements.chartCanvas) return;
  elements.chartCanvas.innerHTML = `<div class="empty-state">讀取中...</div>`;
  if (elements.chartTooltip) elements.chartTooltip.classList.remove("show");
};

const hideChartTooltip = () => {
  if (!elements.chartTooltip) return;
  elements.chartTooltip.classList.remove("show");
};

function applyZoomRange(startIdx, endIdx) {
  const baseRows = chartState.rows && chartState.rows.length ? chartState.rows : chartState.fullRows;
  const fullRows = chartState.fullRows && chartState.fullRows.length ? chartState.fullRows : baseRows;
  if (!baseRows || !fullRows || !baseRows.length) return;
  const baseStart = chartState.window?.start || 0;
  const loIdx = Math.max(0, Math.min(startIdx, endIdx));
  const hiIdx = Math.min(baseRows.length - 1, Math.max(startIdx, endIdx));
  if (hiIdx - loIdx < 1) return;
  const absStart = baseStart + loIdx;
  const absEnd = baseStart + hiIdx;
  zoomToWindow(absStart, absEnd);
}

function resetZoom() {
  if (!chartState.fullRes || !chartState.fullRows.length) return;
  const fullLen = chartState.fullRows.length;
  zoomToWindow(0, fullLen - 1);
}

function zoomToWindow(startAbs, endAbs) {
  const fullRows = chartState.fullRows;
  if (!fullRows || !fullRows.length) return;
  const loAbs = Math.max(0, Math.min(startAbs, endAbs));
  const hiAbs = Math.min(fullRows.length - 1, Math.max(startAbs, endAbs));
  if (hiAbs - loAbs < 1) return;
  const subRows = fullRows.slice(loAbs, hiAbs + 1);
  const base = chartState.fullRes || {
    start_date: subRows[0].trade_date,
    end_date: subRows[subRows.length - 1].trade_date,
    items: chartState.fullRows,
    total_count: chartState.fullRows.length,
  };
  const next = {
    ...base,
    items: subRows,
    start_date: subRows[0].trade_date,
    end_date: subRows[subRows.length - 1].trade_date,
    total_count: subRows.length,
  };
  renderHistoryChart(next, chartState.events || [], { preserveFull: true, absStart: loAbs, absEnd: hiAbs });
  toggleResetZoom(subRows.length < fullRows.length);
}

const renderHistoryChart = (res, backtestEvents = [], options = {}) => {
  const { preserveFull = false, absStart, absEnd } = options;
  if (!elements.chartCanvas) return;
  const rows = res.items || [];
  if (!rows.length) {
    renderChartPlaceholder("尚無走勢資料");
    return;
  }

  const closes = rows.map((row) => row.close_price);
  const maxClose = Math.max(...closes);
  const minClose = Math.min(...closes);
  if (!preserveFull) {
    chartState.fullRows = rows;
    chartState.fullRes = res;
    chartState.window = { start: 0, end: rows.length ? rows.length - 1 : 0 };
    toggleResetZoom(false);
  } else {
    const fullLen = chartState.fullRows?.length || rows.length;
    const startIdx = typeof absStart === "number" ? absStart : 0;
    const endIdx = typeof absEnd === "number" ? absEnd : startIdx + rows.length - 1;
    chartState.window = { start: startIdx, end: endIdx };
    const showReset = endIdx - startIdx + 1 < fullLen;
    toggleResetZoom(showReset);
  }
  if (elements.chartMeta) {
    renderMeta(elements.chartMeta, [
      `區間：${fmtText(res.start_date)} ~ ${fmtText(res.end_date)}`,
      `筆數：${fmtInt(res.total_count)}`,
      `最高收盤：${fmtPrice(maxClose)}`,
      `最低收盤：${fmtPrice(minClose)}`,
    ]);
  }
  const canvas = elements.chartCanvas;
  const width = canvas.clientWidth || 640;
  const height = canvas.clientHeight || 320;
  const padding = { top: 20, right: 24, bottom: 32, left: 48 };
  const plotWidth = Math.max(width - padding.left - padding.right, 1);
  const plotHeight = Math.max(height - padding.top - padding.bottom, 1);

  const range = maxClose - minClose || 1;
  const step = rows.length > 1 ? plotWidth / (rows.length - 1) : plotWidth;

  const points = rows.map((row, idx) => {
    const x = padding.left + (rows.length > 1 ? idx * step : plotWidth / 2);
    const ratio = (row.close_price - minClose) / range;
    const y = padding.top + (1 - ratio) * plotHeight;
    return { x, y, row };
  });

  const linePath = points
    .map((pt, idx) => `${idx === 0 ? "M" : "L"}${pt.x},${pt.y}`)
    .join(" ");
  const areaPath = `${linePath} L ${padding.left + plotWidth},${padding.top + plotHeight} L ${
    padding.left
  },${padding.top + plotHeight} Z`;

  const tickCount = 4;
  const gridLines = [];
  const axisLabels = [];
  for (let i = 0; i <= tickCount; i++) {
    const y = padding.top + (plotHeight / tickCount) * i;
    const value = maxClose - (range / tickCount) * i;
    gridLines.push(`<line x1="${padding.left}" y1="${y}" x2="${padding.left + plotWidth}" y2="${y}" />`);
    axisLabels.push(
      `<text x="${padding.left - 6}" y="${y + 4}" text-anchor="end">${fmtPrice(value)}</text>`
    );
  }

  const xLabels = [];
  const labelIndexes = [0, Math.floor((rows.length - 1) / 2), rows.length - 1].filter(
    (value, index, self) => self.indexOf(value) === index
  );
  labelIndexes.forEach((idx) => {
    const pt = points[idx];
    if (!pt) return;
    xLabels.push(
      `<text x="${pt.x}" y="${padding.top + plotHeight + 20}" text-anchor="middle">${pt.row.trade_date}</text>`
    );
  });

  canvas.innerHTML = `
    <svg viewBox="0 0 ${width} ${height}" preserveAspectRatio="none" aria-label="BTC/USDT 走勢圖">
      <g class="chart-grid">${gridLines.join("")}</g>
      <g class="chart-axis">${axisLabels.join("")}${xLabels.join("")}</g>
      <path class="chart-area" d="${areaPath}"></path>
      <path class="chart-line" d="${linePath}"></path>
      <line class="chart-focus-line" data-role="focus-line" x1="0" x2="0" y1="${padding.top}" y2="${
        padding.top + plotHeight
      }" style="opacity:0"></line>
      <circle class="chart-focus-dot" data-role="focus-dot" cx="0" cy="0" r="4" style="opacity:0"></circle>
    </svg>
    <div class="chart-overlay" data-role="overlay"></div>
    <div class="chart-selection hidden" data-role="selection"></div>
  `;

  const svg = canvas.querySelector("svg");
  chartState.rows = rows;
  chartState.points = points;
  chartState.events = backtestEvents;
  chartState.padding = padding;
  chartState.plotWidth = plotWidth;
  chartState.plotHeight = plotHeight;
  chartState.step = step;
  chartState.focusLine = svg.querySelector("[data-role='focus-line']");
  chartState.focusDot = svg.querySelector("[data-role='focus-dot']");
  const overlay = canvas.querySelector("[data-role='overlay']");
  const selectionBox = canvas.querySelector("[data-role='selection']");

  if (backtestEvents && backtestEvents.length) {
    const eventDates = new Set(
      backtestEvents
        .map((e) => {
          if (!e.trade_date) return null;
          const d = new Date(e.trade_date);
          return Number.isNaN(d.getTime()) ? String(e.trade_date).slice(0, 10) : d.toISOString().slice(0, 10);
        })
        .filter(Boolean)
    );
    const markers = points
      .filter((pt) => {
        const dateStr = pt.row.trade_date
          ? String(pt.row.trade_date).slice(0, 10)
          : pt.row.date
          ? String(pt.row.date).slice(0, 10)
          : "";
        return eventDates.has(dateStr);
      })
      .map(
        (pt) =>
          `<circle cx="${pt.x}" cy="${pt.y}" r="4.5" fill="var(--accent)" stroke="#fff" stroke-width="1.5" />`
      )
      .join("");
    svg.insertAdjacentHTML("beforeend", markers);
  }

  let dragging = false;
  const toIndex = (clientX) => {
    if (!overlay) return 0;
    const rect = overlay.getBoundingClientRect();
    const x = Math.max(0, Math.min(clientX - rect.left, rect.width));
    const plotX = Math.max(0, Math.min(x - padding.left, plotWidth));
    return chartState.rows.length > 1 ? Math.round(plotX / chartState.step) : 0;
  };

  const handlePointer = (event) => {
    if (dragging) return;
    if (!overlay) return;
    if (!chartState.rows.length) return;
    const rect = overlay.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const plotX = Math.max(0, Math.min(x - padding.left, plotWidth));
    const idx = chartState.rows.length > 1 ? Math.round(plotX / chartState.step) : 0;
    const point = chartState.points[Math.max(0, Math.min(idx, chartState.points.length - 1))];
    if (!point) return;

    chartState.focusLine.setAttribute("x1", point.x);
    chartState.focusLine.setAttribute("x2", point.x);
    chartState.focusLine.style.opacity = "1";
    chartState.focusDot.setAttribute("cx", point.x);
    chartState.focusDot.setAttribute("cy", point.y);
    chartState.focusDot.style.opacity = "1";

    const tooltip = elements.chartTooltip;
    if (!tooltip) return;
    tooltip.innerHTML = `
      <div class="tooltip-title">${point.row.trade_date}</div>
      <div class="tooltip-row"><span>收盤價</span><strong>${fmtPrice(point.row.close_price)}</strong></div>
      <div class="tooltip-row"><span>日漲跌</span><strong>${fmtPercent(
        point.row.change_percent
      )}</strong></div>
      <div class="tooltip-row"><span>近 5 日</span><strong>${fmtPercent(
        point.row.return_5d
      )}</strong></div>
      <div class="tooltip-row"><span>量能倍率</span><strong>${fmtRatio(
        point.row.volume_ratio
      )}</strong></div>
      <div class="tooltip-row"><span>Score</span><strong>${fmtScore(point.row.score)}</strong></div>
    `;

    const tooltipRect = tooltip.getBoundingClientRect();
    let left = point.x + 12;
    let top = point.y - tooltipRect.height - 12;
    const maxLeft = canvas.clientWidth - tooltipRect.width - 8;
    if (left > maxLeft) left = maxLeft;
    if (left < 8) left = 8;
    if (top < 8) top = point.y + 12;
    tooltip.style.left = `${left}px`;
    tooltip.style.top = `${top}px`;
    tooltip.classList.add("show");
  };

  const handleWheel = (event) => {
    if (!overlay || !chartState.rows.length) return;
    const fullRows = chartState.fullRows || chartState.rows;
    if (!fullRows.length) return;
    event.preventDefault();
    const rect = overlay.getBoundingClientRect();
    const x = Math.max(0, Math.min(event.clientX - rect.left, rect.width));
    const centerIdxCurrent = toIndex(x);
    const baseStart = chartState.window?.start || 0;
    const centerAbs = baseStart + centerIdxCurrent;
    const currentLen = (chartState.window?.end || chartState.rows.length - 1) - baseStart + 1;
    const fullLen = fullRows.length;
    const scale = event.deltaY > 0 ? 1.2 : 0.8; // down = zoom out, up = zoom in
    const targetLen = Math.max(2, Math.min(fullLen, Math.round(currentLen * scale)));
    const half = Math.floor(targetLen / 2);
    let startAbs = centerAbs - half;
    let endAbs = centerAbs + (targetLen - half - 1);
    if (startAbs < 0) {
      endAbs = Math.min(fullLen - 1, endAbs - startAbs);
      startAbs = 0;
    }
    if (endAbs > fullLen - 1) {
      const diff = endAbs - (fullLen - 1);
      startAbs = Math.max(0, startAbs - diff);
      endAbs = fullLen - 1;
    }
    zoomToWindow(startAbs, endAbs);
  };

  if (overlay) {
    overlay.addEventListener("mousemove", handlePointer);
    overlay.addEventListener("click", handlePointer);
    overlay.addEventListener("mouseleave", () => {
      hideChartTooltip();
      if (chartState.focusLine) chartState.focusLine.style.opacity = "0";
      if (chartState.focusDot) chartState.focusDot.style.opacity = "0";
    });
    overlay.addEventListener("wheel", handleWheel, { passive: false });
    overlay.addEventListener("mousedown", (e) => {
      if (!selectionBox) return;
      dragging = true;
      hideChartTooltip();
      const rect = overlay.getBoundingClientRect();
      const startX = Math.max(0, Math.min(e.clientX - rect.left, rect.width));
      selectionBox.classList.remove("hidden");
      selectionBox.style.left = `${startX}px`;
      selectionBox.style.width = "0px";
      const onMove = (ev) => {
        const currX = Math.max(0, Math.min(ev.clientX - rect.left, rect.width));
        const left = Math.min(startX, currX);
        const width = Math.abs(currX - startX);
        selectionBox.style.left = `${left}px`;
        selectionBox.style.width = `${width}px`;
      };
      const onUp = (ev) => {
        window.removeEventListener("mousemove", onMove);
        window.removeEventListener("mouseup", onUp);
        selectionBox.classList.add("hidden");
        const endX = Math.max(0, Math.min(ev.clientX - rect.left, rect.width));
        dragging = false;
        if (Math.abs(endX - startX) < 6) return;
        const startIdx = toIndex(startX);
        const endIdx = toIndex(endX);
        applyZoomRange(startIdx, endIdx);
      };
      window.addEventListener("mousemove", onMove);
      window.addEventListener("mouseup", onUp);
    });
  }
};

const renderSummary = (el, res) => {
  const trendInfo = mapTrend(res.trend);
  const metrics = res.metrics || {};
  const signals = buildSignals(metrics);
  const changeText =
    metrics.change_percent == null
      ? "—"
      : `<span class="delta ${deltaClass(metrics.change_percent)}">${fmtPercent(
          metrics.change_percent
        )}</span>`;
  const returnText =
    metrics.return_5d == null
      ? "—"
      : `<span class="delta ${deltaClass(metrics.return_5d)}">${fmtPercent(
          metrics.return_5d
        )}</span>`;
  el.innerHTML = `
    <div class="summary-head">
      <div>
        <div class="summary-title">${fmtText(res.trading_pair)} · ${fmtText(
          res.trade_date
        )}</div>
        <div class="summary-sub">趨勢：<span class="trend ${trendInfo.className}">${
          trendInfo.label
        }</span></div>
      </div>
      <div class="badge ${trendInfo.tone}">${trendInfo.label}</div>
    </div>
    <div class="summary-metrics">
      <div class="metric-card"><span>收盤價</span><strong>${fmtPrice(
        metrics.close_price
      )}</strong></div>
      <div class="metric-card"><span>日漲跌</span><strong>${changeText}</strong></div>
      <div class="metric-card"><span>近 5 日報酬</span><strong>${returnText}</strong></div>
      <div class="metric-card"><span>量能倍率</span><strong>${fmtRatio(
        metrics.volume_ratio
      )}</strong></div>
      <div class="metric-card"><span>Score</span><strong>${fmtScore(metrics.score)}</strong></div>
    </div>
    <div class="summary-signal">
      <div class="signal-block">
        <strong>觀察重點</strong>
        <ul>${signals.focus.map((item) => `<li>${item}</li>`).join("")}</ul>
      </div>
      <div class="signal-block">
        <strong>風險提醒</strong>
        <ul>${signals.risk.map((item) => `<li>${item}</li>`).join("")}</ul>
      </div>
    </div>
    <div class="advice">${fmtText(res.advice)}</div>
  `;
};

const buildSignals = (metrics) => {
  const focus = [];
  const risk = [];

  if (typeof metrics.change_percent === "number") {
    if (metrics.change_percent >= 0.02) {
      focus.push("日內動能偏強，注意追價節奏");
    } else if (metrics.change_percent <= -0.02) {
      focus.push("日內回檔幅度偏大，觀察支撐區");
    } else {
      focus.push("日內波動收斂，等待方向確認");
    }
  } else {
    focus.push("尚無日內變動資訊");
  }

  if (typeof metrics.return_5d === "number") {
    if (metrics.return_5d >= 0.05) {
      focus.push("近 5 日仍偏多，趨勢延續機率較高");
    } else if (metrics.return_5d <= -0.05) {
      risk.push("近 5 日偏空，留意續跌風險");
    }
  }

  if (typeof metrics.volume_ratio === "number") {
    if (metrics.volume_ratio >= 2) {
      focus.push("量能明顯放大，關注突破延續");
    } else if (metrics.volume_ratio <= 0.8) {
      risk.push("量能偏弱，訊號可靠度降低");
    }
  }

  if (typeof metrics.score === "number") {
    if (metrics.score >= 80) {
      focus.push("分數進入強勢區間，可持續追蹤");
    } else if (metrics.score <= 50) {
      risk.push("分數偏低，留意趨勢轉弱");
    }
  }

  if (!risk.length) {
    risk.push("目前未見明顯風險訊號");
  }

  return {
    focus: focus.slice(0, 3),
    risk: risk.slice(0, 3),
  };
};

const renderMeta = (container, items) => {
  container.innerHTML = items.map((item) => `<div class="meta-item">${item}</div>`).join("");
};

const renderEmptyState = (container, message) => {
  container.innerHTML = `<div class="empty-state">${message}</div>`;
};

const renderTable = (container, rows, columns) => {
  if (!rows.length) {
    renderEmptyState(container, "尚無資料");
    return;
  }
  const thead = columns.map((col) => `<th>${col.label}</th>`).join("");
  const tbody = rows
    .map((row) => {
      const tds = columns
        .map((col) => {
          const raw = row[col.key];
          let content = col.format ? col.format(raw, row) : fmtText(raw);
          if (col.delta) {
            content = `<span class="delta ${deltaClass(raw)}">${content}</span>`;
          }
          return `<td class="${col.className || ""}">${content}</td>`;
        })
        .join("");
      return `<tr>${tds}</tr>`;
    })
    .join("");
  container.innerHTML = `<table><thead><tr>${thead}</tr></thead><tbody>${tbody}</tbody></table>`;
};

const buildHighlightCard = (title, items, detailFn) => {
  if (!items || !items.length) return "";
  const list = items
    .map((item) => {
      const name = fmtText(item.trading_pair);
      const detail = detailFn ? detailFn(item) : "";
      return `<div class="highlight-item"><span class="mono">${name}</span><span>${
        detail || ""
      }</span></div>`;
    })
    .join("");
  return `<article class="highlight-card"><h4>${title}</h4>${list}</article>`;
};

const renderHighlights = (container, rows) => {
  if (!rows.length) {
    renderEmptyState(container, "尚無亮點資料");
    return;
  }
  const byScore = [...rows]
    .sort((a, b) => (b.score ?? -Infinity) - (a.score ?? -Infinity))
    .slice(0, 3);
  const byChange = [...rows]
    .filter((item) => typeof item.change_percent === "number")
    .sort((a, b) => Math.abs(b.change_percent) - Math.abs(a.change_percent))[0];
  const byVolume = [...rows]
    .filter((item) => typeof item.volume_ratio === "number")
    .sort((a, b) => (b.volume_ratio ?? 0) - (a.volume_ratio ?? 0))[0];

  const cards = [
    buildHighlightCard("高分排行", byScore, (item) => `Score ${fmtScore(item.score)}`),
  ];
  if (byChange) {
    cards.push(
      buildHighlightCard("波動焦點", [byChange], (item) => `日漲跌 ${fmtPercent(item.change_percent)}`)
    );
  }
  if (byVolume) {
    cards.push(
      buildHighlightCard("量能焦點", [byVolume], (item) => `量能 ${fmtRatio(item.volume_ratio)}`)
    );
  }
  container.innerHTML = cards.join("");
};

const renderBacktestSummary = (res) => {
  if (!elements.backtestSummary) return;
  const returns = res.stats?.returns || {};
  const statCards =
    Object.entries(returns).length === 0
      ? `<div class="meta-item">尚無統計</div>`
      : Object.entries(returns)
          .map(
            ([key, val]) => `
              <div class="stat-card">
                <div class="stat-title">${key.toUpperCase()}</div>
                <div class="stat-value">${fmtPercent(val.avg_return)}</div>
                <div class="stat-sub">平均報酬</div>
                <div class="stat-sub">勝率 ${fmtPercent(val.win_rate)}</div>
              </div>
            `
          )
          .join("");
  elements.backtestSummary.innerHTML = `
    <div class="result-header">
      <div>
        <div class="result-title">回測結果</div>
        <div class="result-sub">${res.start_date} ~ ${res.end_date}</div>
      </div>
      <span class="badge ${res.total_events ? "good" : "warn"}">命中 ${fmtInt(res.total_events)}</span>
    </div>
    <div class="stat-grid">${statCards}</div>
  `;
};

const renderBacktestEvents = (res) => {
  if (!elements.backtestEvents) return;
  const rows = res.events || [];
  if (!rows.length) {
    renderEmptyState(elements.backtestEvents, "回測條件未命中任何日期");
    return;
  }
  const cards = rows
    .slice(0, 12)
    .map(
      (row) => `
      <article class="highlight-card">
        <h4>${row.trade_date}</h4>
        <div class="highlight-item"><span>總分</span><span>${fmtScore(row.total_score)}</span></div>
        <div class="highlight-item"><span>Score</span><span>${fmtScore(row.score)}</span></div>
        <div class="highlight-item"><span>日漲跌</span><span class="delta ${deltaClass(
          row.change_percent
        )}">${fmtPercent(row.change_percent)}</span></div>
        <div class="highlight-item"><span>量能倍率</span><span>${fmtRatio(row.volume_ratio)}</span></div>
        <div class="highlight-item"><span>近 5 日</span><span class="delta ${deltaClass(
          row.return_5d
        )}">${fmtPercent(row.return_5d)}</span></div>
        <div class="highlight-item"><span>MA20 乖離</span><span class="delta ${deltaClass(
          row.ma_gap
        )}">${fmtPercent(row.ma_gap)}</span></div>
        <div class="highlight-item"><span>收盤</span><span>${fmtPrice(row.close_price)}</span></div>
        <div class="highlight-item"><span>+3日</span><span>${fmtPercent(row.forward_returns?.d3)}</span></div>
        <div class="highlight-item"><span>+5日</span><span>${fmtPercent(row.forward_returns?.d5)}</span></div>
        <div class="highlight-item"><span>+10日</span><span>${fmtPercent(row.forward_returns?.d10)}</span></div>
      </article>
    `
    )
    .join("");
  elements.backtestEvents.innerHTML = cards;
};

const numInput = (id, fallback = 0) => {
  const raw = document.getElementById(id)?.value;
  const n = Number(raw);
  return Number.isFinite(n) ? n : fallback;
};

const readWeightConfig = () => ({
  change_weight: backtestSelections.conditions.includes("change") ? numInput("btChangeWeight", 1) : 0,
  volume_weight: backtestSelections.conditions.includes("volume") ? numInput("btVolWeight", 1) : 0,
  return_weight: backtestSelections.conditions.includes("return") ? numInput("btReturnWeight", 1) : 0,
  ma_weight: backtestSelections.conditions.includes("ma") ? numInput("btMaWeight", 1) : 0,
});

const readThresholds = () => ({
  total_min: numInput("btTotalMin", 0),
  change_min: backtestSelections.conditions.includes("change") ? numInput("btChangeMin", 0) / 100 : 0,
  volume_ratio_min: backtestSelections.conditions.includes("volume") ? numInput("btVolMin", 0) : 0,
  return5_min: backtestSelections.conditions.includes("return") ? numInput("btReturnMin", 0) / 100 : 0,
  ma_gap_min: backtestSelections.conditions.includes("ma") ? numInput("btMaGap", 0) / 100 : 0,
});

const applyWeightedScoring = (res) => {
  if (!res || !res.events) return res;
  const weights = readWeightConfig();
  const thresholds = readThresholds();
  const flags = {
    change: backtestSelections.conditions.includes("change"),
    volume: backtestSelections.conditions.includes("volume"),
    return: backtestSelections.conditions.includes("return"),
    ma: backtestSelections.conditions.includes("ma"),
  };
  const events = (res.events || []).map((row) => {
    let total = 0;
    const comp = { ...(row.components || {}) };
    if (flags.change && row.change_percent >= thresholds.change_min) {
      const v = weights.change_weight || 0;
      comp.change = v;
      total += v;
    }
    if (flags.volume && row.volume_ratio >= thresholds.volume_ratio_min) {
      const v = weights.volume_weight || 0;
      comp.volume = v;
      total += v;
    }
    if (flags.return && (row.return_5d || 0) >= thresholds.return5_min) {
      const v = weights.return_weight || 0;
      comp.return = v;
      total += v;
    }
    if (flags.ma && (row.ma_gap || 0) >= thresholds.ma_gap_min) {
      const v = weights.ma_weight || 0;
      comp.ma = v;
      total += v;
    }
    return { ...row, total_score: total, components: comp };
  });
  const filtered = events.filter((ev) => ev.total_score >= thresholds.total_min);
  const stats = computeReturnStats(filtered);
  return { ...res, events: filtered, total_events: filtered.length, params: { weights, thresholds }, stats };
};

const computeReturnStats = (events) => {
  const horizons = ["d3", "d5", "d10"];
  const stats = {};
  horizons.forEach((h) => {
    let sum = 0;
    let count = 0;
    let wins = 0;
    events.forEach((ev) => {
      const v = ev.forward_returns?.[h];
      if (v === null || v === undefined) return;
      sum += v;
      count += 1;
      if (v > 0) wins += 1;
    });
    const avg = count ? sum / count : 0;
    const win = count ? wins / count : 0;
    stats[h] = { avg_return: avg, win_rate: win };
  });
  return { returns: stats };
};

const buildBacktestPayload = () => {
  const weights = readWeightConfig();
  const start_date = elements.chartStart?.value;
  const end_date = elements.chartEnd?.value;
  const thresholds = readThresholds();
  return {
    symbol: "BTCUSDT",
    start_date,
    end_date,
    weights,
    thresholds,
    flags: {
      use_change: backtestSelections.conditions.includes("change"),
      use_volume: backtestSelections.conditions.includes("volume"),
      use_return: backtestSelections.conditions.includes("return"),
      use_ma: backtestSelections.conditions.includes("ma"),
    },
    conditions: [...backtestSelections.conditions],
    horizons: [3, 5, 10],
  };
};

const renderBacktestConditions = () => {
  if (!elements.btSelectedConditions) return;
  if (!backtestSelections.conditions.length) {
    renderEmptyState(elements.btSelectedConditions, "尚未選擇條件");
    return;
  }
  const current = (id, fallback) => document.getElementById(id)?.value ?? fallback;
  const cards = backtestSelections.conditions
    .map((cond) => {
      if (cond === "change") {
        return `
          <div class="condition-card" data-cond="change">
            <div class="condition-card-head">
              <span>日漲跌條件</span>
              <button type="button" class="condition-remove" data-remove="change">移除</button>
            </div>
            <div class="optional-fields">
              <label>漲幅門檻(%) <input type="number" step="0.1" id="btChangeMin" value="${current(
                "btChangeMin",
                0.5
              )}"></label>
              <label>條件加權 <input type="number" step="0.1" id="btChangeWeight" value="${current(
                "btChangeWeight",
                1
              )}"></label>
            </div>
          </div>
        `;
      }
      if (cond === "volume") {
        return `
          <div class="condition-card" data-cond="volume">
            <div class="condition-card-head">
              <span>量能條件</span>
              <button type="button" class="condition-remove" data-remove="volume">移除</button>
            </div>
            <div class="optional-fields">
              <label>量能門檻(倍率) <input type="number" step="0.1" id="btVolMin" value="${current(
                "btVolMin",
                1.2
              )}"></label>
              <label>條件加權 <input type="number" step="0.1" id="btVolWeight" value="${current(
                "btVolWeight",
                1
              )}"></label>
            </div>
          </div>
        `;
      }
      if (cond === "return") {
        return `
          <div class="condition-card" data-cond="return">
            <div class="condition-card-head">
              <span>近 5 日報酬條件</span>
              <button type="button" class="condition-remove" data-remove="return">移除</button>
            </div>
            <div class="optional-fields">
              <label>報酬門檻(%) <input type="number" step="0.1" id="btReturnMin" value="${current(
                "btReturnMin",
                1.0
              )}"></label>
              <label>條件加權 <input type="number" step="0.1" id="btReturnWeight" value="${current(
                "btReturnWeight",
                1
              )}"></label>
            </div>
          </div>
        `;
      }
      if (cond === "ma") {
        return `
          <div class="condition-card" data-cond="ma">
            <div class="condition-card-head">
              <span>均線乖離條件</span>
              <button type="button" class="condition-remove" data-remove="ma">移除</button>
            </div>
            <div class="optional-fields">
              <label>乖離門檻(%) <input type="number" step="0.1" id="btMaGap" value="${current(
                "btMaGap",
                1.0
              )}"></label>
              <label>條件加權 <input type="number" step="0.1" id="btMaWeight" value="${current(
                "btMaWeight",
                1
              )}"></label>
            </div>
          </div>
        `;
      }
      return "";
    })
    .join("");
  elements.btSelectedConditions.innerHTML = cards;
  elements.btSelectedConditions.querySelectorAll("[data-remove]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const cond = btn.dataset.remove;
      backtestSelections.conditions = backtestSelections.conditions.filter((c) => c !== cond);
      renderBacktestConditions();
      refreshConditionOptions();
    });
  });
};

const renderActivity = () => {
  if (!state.activity.length) {
    elements.activityList.innerHTML = `<li class="empty-state">尚無操作紀錄</li>`;
    return;
  }
  elements.activityList.innerHTML = state.activity
    .map(
      (item) => `
      <li>
        <div class="activity-title">${item.title}</div>
        <div class="activity-detail">${item.detail}</div>
        <div class="activity-time">${item.time}</div>
      </li>
    `
    )
    .join("");
};

const logActivity = (title, detail) => {
  const entry = {
    title,
    detail,
    time: timeFormat.format(new Date()),
  };
  state.activity.unshift(entry);
  state.activity = state.activity.slice(0, 6);
  renderActivity();
  touchUpdatedAt();
};

const requireLogin = () => {
  if (!state.token) throw new Error("請先登入後再操作");
};

async function initHealth() {
  try {
    const res = await fetch("/api/health");
    const data = await res.json();
    if (data.success) {
      state.health = data;
      const ok = String(data.db || "").toLowerCase() === "ok";
      setHealthStatus(
        `系統檢查 OK ｜ DB: ${data.db} ｜ 合成資料: ${data.use_synthetic ? "ON" : "OFF"}`,
        ok ? "good" : "warn"
      );
      updateOverviewMode();
      renderKpis();
    }
  } catch (_) {
    setHealthStatus("系統檢查失敗", "warn");
  }
}

initHealth();
renderChartPlaceholder();
renderEmptyState(elements.strategyTable, "尚未載入策略");
renderEmptyState(elements.tradeTable, "尚未查詢");
renderEmptyState(elements.reportTable, "尚未查詢");
renderEmptyState(elements.positionTable, "尚未查詢");
renderEmptyState(elements.logTable, "尚未查詢");
renderActivity();
renderKpis();
updateOverviewMode();
renderBacktestConditions();
refreshConditionOptions();
showSection("overview");
toggleProtectedSections(false);
refreshAccessToken().then((ok) => {
  if (ok) {
    setMessage(elements.loginMessage, "已自動登入，Token 已更新", "good");
    logActivity("自動登入", "沿用前一次的登入狀態");
    showSection("overview");
    loadChartHistory(true).catch(() => {});
    fetchCombos();
    loadStrategies().catch(() => {});
    loadTrades().catch(() => {});
    loadPositions().catch(() => {});
    if (elements.reportStrategyId?.value) {
      loadReports().catch(() => {});
    }
  } else {
    setStatus("未登入", "warn");
  }
});

const today = new Date().toISOString().slice(0, 10);
const startOfYear = new Date(Date.UTC(new Date().getUTCFullYear(), 0, 1)).toISOString().slice(0, 10);
["chartEnd"].forEach((id) => {
  const el = document.getElementById(id);
  if (el) el.value = today;
});
if (elements.chartStart) elements.chartStart.value = startOfYear;
setStrategyBacktestDefaults();
updateOptionalFields();
loadPreset();
elements.btConditionSelect?.addEventListener("change", (e) => {
  const val = e.target.value;
  if (val && !backtestSelections.conditions.includes(val)) {
    backtestSelections.conditions.push(val);
    e.target.value = "";
    updateOptionalFields();
    renderBacktestConditions();
    scheduleAutoBacktest();
  }
});
document.getElementById("savePresetBtn")?.addEventListener("click", savePreset);
document.getElementById("loadPresetBtn")?.addEventListener("click", loadPreset);

Array.from(document.querySelectorAll(".chip[data-email]")).forEach((chip) => {
  chip.addEventListener("click", () => {
    const email = chip.dataset.email;
    document.getElementById("email").value = email;
    document.getElementById("password").value = "password123";
  });
});

const applyBacktestPreset = (preset) => {
  if (!preset) return;
  const c = preset;
  document.getElementById("btTotalMin").value = c.thresholds?.total_min ?? 1;
  if (c.thresholds) {
    document.getElementById("btChangeMin").value = (c.thresholds.change_min || 0) * 100;
    document.getElementById("btVolMin").value = c.thresholds.volume_ratio_min || 0;
    document.getElementById("btReturnMin").value = (c.thresholds.return5_min || 0) * 100;
    document.getElementById("btMaGap").value = (c.thresholds.ma_gap_min || 0) * 100;
  }
  if (c.weights) {
    document.getElementById("btChangeWeight").value = c.weights.change_weight || 1;
    document.getElementById("btVolWeight").value = c.weights.volume_weight || 1;
    document.getElementById("btReturnWeight").value = c.weights.return_weight || 1;
    document.getElementById("btMaWeight").value = c.weights.ma_weight || 1;
  }
  const nextConds = [];
  if (c.flags?.use_change) nextConds.push("change");
  if (c.flags?.use_volume) nextConds.push("volume");
  if (c.flags?.use_return) nextConds.push("return");
  if (c.flags?.use_ma) nextConds.push("ma");
  if (!nextConds.length) nextConds.push("change", "volume");
  backtestSelections.conditions = nextConds;
  updateOptionalFields();
  scheduleAutoBacktest();
};

async function loadPreset() {
  if (!state.token) return;
  try {
    const res = await api("/api/analysis/backtest/preset");
    if (res.success && res.preset) {
      applyBacktestPreset(res.preset);
      setStatus("已載入回測預設", "good");
    }
  } catch (err) {
    console.warn("load preset failed", err);
  }
}

async function savePreset() {
  try {
    requireLogin();
    const payload = buildBacktestPayload();
    await api("/api/analysis/backtest/preset/save", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    setStatus("已儲存回測預設", "good");
    logActivity("儲存回測設定", `${backtestSelections.conditions.join(",")} 條件`);
  } catch (err) {
    setMessage(elements.loginMessage, `儲存失敗：${err.message}`, "error");
  }
}

const renderStrategyTable = (items = []) => {
  if (!elements.strategyTable) return;
  if (!items.length) {
    renderEmptyState(elements.strategyTable, "尚未載入策略");
    if (elements.strategyMeta) elements.strategyMeta.innerHTML = "";
    renderRunTrades([]);
    return;
  }
  const rows = items.map((s) => ({
    name: fmtText(s.name),
    env: fmtText(s.env),
    status: `<span class="badge">${fmtText(s.status)}</span>`,
    version: fmtInt(s.version),
    base_symbol: fmtText(s.base_symbol),
    updated_at: s.updated_at ? timeFormat.format(new Date(s.updated_at)) : "—",
    actions: `
      <div class="action-row">
        <button class="ghost btn-sm" data-action="run" data-env="test" data-id="${fmtText(s.id)}">試跑 test</button>
        <button class="ghost btn-sm" data-action="run" data-env="prod" data-id="${fmtText(s.id)}">試跑 prod</button>
        <button class="ghost btn-sm" data-action="activate" data-env="test" data-id="${fmtText(s.id)}">啟用 test</button>
        <button class="ghost btn-sm" data-action="activate" data-env="prod" data-id="${fmtText(s.id)}">啟用 prod</button>
        <button class="ghost btn-sm" data-action="deactivate" data-id="${fmtText(s.id)}">停用</button>
      </div>
    `,
  }));
  const cols = [
    { key: "name", label: "名稱" },
    { key: "env", label: "環境" },
    { key: "status", label: "狀態" },
    { key: "version", label: "版次" },
    { key: "base_symbol", label: "交易對" },
    { key: "updated_at", label: "更新時間" },
    { key: "actions", label: "操作" },
  ];
  renderTable(elements.strategyTable, rows, cols);
  if (elements.strategyMeta) {
    elements.strategyMeta.innerHTML = `<div class="meta-item">共 ${fmtInt(items.length)} 筆策略</div>`;
  }
  bindStrategyActions();
  syncStrategyOptions();
};

const renderRunTrades = (trades = [], env = "") => {
  if (!elements.strategyRunResult) return;
  if (!trades.length) {
    renderEmptyState(elements.strategyRunResult, "尚未試跑");
    return;
  }
  const rows = trades.map((t) => ({
    entry_date: fmtDate(t.entry_date),
    exit_date: fmtDate(t.exit_date),
    entry_price: t.entry_price,
    exit_price: t.exit_price,
    pnl_usdt: t.pnl_usdt,
    pnl_pct: t.pnl_pct,
    reason: t.reason,
  }));
  const cols = [
    { key: "entry_date", label: "進場日" },
    { key: "exit_date", label: "出場日" },
    { key: "entry_price", label: "進場價", format: fmtPrice },
    { key: "exit_price", label: "出場價", format: fmtPrice },
    { key: "pnl_usdt", label: "PNL (USDT)", format: fmtNumber, delta: true },
    { key: "pnl_pct", label: "PNL%", format: fmtPercent, delta: true },
    { key: "reason", label: "原因" },
  ];
  const envLabel = env ? `（${env}）` : "";
  elements.strategyRunResult.innerHTML = `<div class="meta-row"><div class="meta-item">試跑筆數：${fmtInt(
    trades.length
  )}${envLabel}</div></div>`;
  const tableDiv = document.createElement("div");
  renderTable(tableDiv, rows, cols);
  elements.strategyRunResult.appendChild(tableDiv);
};

const loadStrategies = async () => {
  if (!elements.strategyForm) return;
  renderEmptyState(elements.strategyTable, "載入策略中...");
  const status = elements.strategyStatus?.value || "";
  const env = elements.strategyEnv?.value || "";
  const name = (elements.strategyName?.value || "").trim();
  const qs = new URLSearchParams();
  if (status) qs.append("status", status);
  if (env) qs.append("env", env);
  if (name) qs.append("name", name);
  const res = await api(`/api/admin/strategies${qs.toString() ? `?${qs.toString()}` : ""}`);
  state.strategies = res.strategies || [];
  renderStrategyTable(state.strategies);
  logActivity("查詢策略列表", `筆數 ${fmtInt(state.strategies.length)}`);
  touchUpdatedAt();
  setStatus("策略列表已更新", "good");
};

const syncStrategyOptions = () => {
  if (!elements.strategyBacktestSelect) return;
  const current = elements.strategyBacktestSelect.value;
  const options = state.strategies.map(
    (s) =>
      `<option value="${s.id}">${fmtText(s.name || s.id)}（v${s.version || "-"}｜${s.env || "-"}）</option>`
  );
  elements.strategyBacktestSelect.innerHTML = `<option value="">選擇策略…</option>${options.join("")}`;
  if (current) {
    elements.strategyBacktestSelect.value = current;
  }
};

const syncBacktestSelectedStrategy = () => {
  if (elements.strategyBacktestSelect && elements.strategyBacktestSelect.value) {
    elements.strategyBacktestId.value = elements.strategyBacktestSelect.value;
  }
};

const fmtDate = (v) => {
  if (!v) return "—";
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return fmtText(v);
  return d.toISOString().slice(0, 10);
};

function setStrategyBacktestDefaults() {
  const today = new Date();
  const start = new Date();
  start.setDate(today.getDate() - 180);
  if (elements.strategyBtStart) elements.strategyBtStart.value = start.toISOString().slice(0, 10);
  if (elements.strategyBtEnd) elements.strategyBtEnd.value = today.toISOString().slice(0, 10);
  if (elements.strategyBtPriceMode) elements.strategyBtPriceMode.value = "next_open";
}

const collectStrategyBacktestPayload = () => {
  const strategyID = (elements.strategyBacktestSelect?.value || elements.strategyBacktestId?.value || "").trim();
  if (!strategyID) {
    return { ok: false, error: "請選擇或輸入策略 ID" };
  }
  const start = elements.strategyBtStart?.value;
  const end = elements.strategyBtEnd?.value;
  if (!start || !end) return { ok: false, error: "請填寫回測起訖日" };
  if (new Date(end) < new Date(start)) return { ok: false, error: "結束日需晚於起始日" };
  const initialEquity = Number(elements.strategyBtEquity?.value || 0);
  if (!initialEquity || Number.isNaN(initialEquity) || initialEquity <= 0) {
    return { ok: false, error: "初始資金需為正數" };
  }

  const fees = toPercent(elements.strategyBtFees?.value, "手續費", true, 0);
  if (!fees.ok) return { ok: false, error: fees.error };
  const slippage = toPercent(elements.strategyBtSlippage?.value, "滑價", true, 0);
  if (!slippage.ok) return { ok: false, error: slippage.error };
  const stop = toPercent(elements.strategyBtStop?.value, "停損", true, null);
  if (!stop.ok) return { ok: false, error: stop.error };
  const take = toPercent(elements.strategyBtTake?.value, "停利", true, null);
  if (!take.ok) return { ok: false, error: take.error };
  const dailyLoss = toPercent(elements.strategyBtDailyLoss?.value, "單日最大虧損", true, null);
  if (!dailyLoss.ok) return { ok: false, error: dailyLoss.error };

  const coolDown = elements.strategyBtCoolDown?.value ? Number(elements.strategyBtCoolDown.value) : null;
  const minHold = elements.strategyBtMinHold?.value ? Number(elements.strategyBtMinHold.value) : null;
  const maxPos = elements.strategyBtMaxPos?.value ? Number(elements.strategyBtMaxPos.value) : null;
  if (coolDown !== null && (Number.isNaN(coolDown) || coolDown < 0)) {
    return { ok: false, error: "冷卻天數需為 0 或正整數" };
  }
  if (minHold !== null && (Number.isNaN(minHold) || minHold < 0)) {
    return { ok: false, error: "最少持有天數需為 0 或正整數" };
  }
  if (maxPos !== null && (Number.isNaN(maxPos) || maxPos < 1)) {
    return { ok: false, error: "最多持倉需為大於 0 的整數" };
  }

  const payload = {
    start_date: start,
    end_date: end,
    initial_equity: initialEquity,
    price_mode: elements.strategyBtPriceMode?.value || "next_open",
    fees_pct: fees.value,
    slippage_pct: slippage.value,
    stop_loss_pct: stop.value,
    take_profit_pct: take.value,
    max_daily_loss_pct: dailyLoss.value,
    cool_down_days: coolDown,
    min_hold_days: minHold,
    max_positions: maxPos,
  };

  return { ok: true, strategyID, payload };
};

const renderStrategyBacktestTrades = (trades = []) => {
  if (!elements.strategyBacktestTrades) return;
  state.lastStrategyTrades = trades;
  const rows = trades.map((t) => ({
    entry: fmtDate(t.entry_date),
    exit: fmtDate(t.exit_date),
    entry_price: t.entry_price,
    exit_price: t.exit_price,
    pnl: t.pnl_usdt,
    pnl_pct: t.pnl_pct,
    hold: t.hold_days,
    reason: t.reason,
  }));
  const cols = [
    { key: "entry", label: "買入日" },
    { key: "exit", label: "賣出日" },
    { key: "entry_price", label: "買入價", format: fmtPrice },
    { key: "exit_price", label: "賣出價", format: fmtPrice },
    { key: "pnl", label: "PNL (USDT)", format: fmtNumber, delta: true },
    { key: "pnl_pct", label: "PNL%", format: fmtPercent, delta: true },
    { key: "hold", label: "持有天數", format: fmtInt },
    { key: "reason", label: "原因", format: fmtText },
  ];
  renderTable(elements.strategyBacktestTrades, rows, cols);
};

const renderEquityChart = (equity = [], trades = []) => {
  if (!elements.strategyEquityChart) return;
  state.lastStrategyEquity = equity;
  if (!equity.length) {
    renderEmptyState(elements.strategyEquityChart, "尚無淨值資料");
    return;
  }
  const width = elements.strategyEquityChart.clientWidth || 640;
  const height = elements.strategyEquityChart.clientHeight || 240;
  const padding = { top: 16, right: 16, bottom: 28, left: 48 };
  const plotWidth = Math.max(width - padding.left - padding.right, 1);
  const plotHeight = Math.max(height - padding.top - padding.bottom, 1);
  const values = equity.map((p) => p.equity);
  const minV = Math.min(...values);
  const maxV = Math.max(...values);
  const range = maxV - minV || 1;
  const step = equity.length > 1 ? plotWidth / (equity.length - 1) : plotWidth;
  const points = equity.map((p, idx) => {
    const x = padding.left + (equity.length > 1 ? idx * step : plotWidth / 2);
    const y = padding.top + (1 - (p.equity - minV) / range) * plotHeight;
    return { x, y, date: p.date, equity: p.equity };
  });
  const linePath = points.map((pt, idx) => `${idx === 0 ? "M" : "L"}${pt.x},${pt.y}`).join(" ");
  const areaPath = `${linePath} L ${padding.left + plotWidth},${padding.top + plotHeight} L ${
    padding.left
  },${padding.top + plotHeight} Z`;

  const gridLines = [];
  const axisLabels = [];
  const tickCount = 4;
  for (let i = 0; i <= tickCount; i++) {
    const y = padding.top + (plotHeight / tickCount) * i;
    const val = maxV - (range / tickCount) * i;
    gridLines.push(`<line x1="${padding.left}" y1="${y}" x2="${padding.left + plotWidth}" y2="${y}" />`);
    axisLabels.push(`<text x="${padding.left - 6}" y="${y + 4}" text-anchor="end">${fmtPrice(val)}</text>`);
  }
  const xLabels = [];
  const labelIndexes = [0, Math.floor((equity.length - 1) / 2), equity.length - 1].filter(
    (v, i, arr) => arr.indexOf(v) === i
  );
  labelIndexes.forEach((idx) => {
    const pt = points[idx];
    if (!pt) return;
    xLabels.push(
      `<text x="${pt.x}" y="${padding.top + plotHeight + 18}" text-anchor="middle">${fmtDate(pt.date)}</text>`
    );
  });

  const markerDates = new Set();
  trades.forEach((t) => {
    if (t.entry_date) markerDates.add(fmtDate(t.entry_date));
    if (t.exit_date) markerDates.add(fmtDate(t.exit_date));
  });
  const markers = points
    .filter((pt) => markerDates.has(fmtDate(pt.date)))
    .map(
      (pt) =>
        `<circle cx="${pt.x}" cy="${pt.y}" r="4" fill="var(--accent)" stroke="#fff" stroke-width="1.2" />`
    )
    .join("");

  elements.strategyEquityChart.innerHTML = `
    <svg viewBox="0 0 ${width} ${height}" preserveAspectRatio="none" aria-label="回測淨值曲線">
      <g class="chart-grid">${gridLines.join("")}</g>
      <g class="chart-axis">${axisLabels.join("")}${xLabels.join("")}</g>
      <path class="chart-area" d="${areaPath}"></path>
      <path class="chart-line" d="${linePath}"></path>
      ${markers}
    </svg>
  `;
};

const renderStrategyBacktestSummary = (rec) => {
  if (!elements.strategyBacktestSummary) return;
  if (!rec) {
    renderEmptyState(elements.strategyBacktestSummary, "尚未執行回測");
    renderStrategyBacktestTrades([]);
    renderEquityChart([]);
    renderStrategyBacktestHistory([]);
    return;
  }
  const params = rec.params || {};
  const result = rec.result || {};
  const stats = result.stats || {};
  const metaItems = [
    `策略：${fmtText(rec.strategy_id)}（v${rec.strategy_version || "-"})`,
    `區間：${fmtDate(params.start_date)} ~ ${fmtDate(params.end_date)}`,
    `成交價模式：${fmtText(params.price_mode)}`,
    `手續費：${fmtPercent(params.fees_pct)}`,
    `滑價：${fmtPercent(params.slippage_pct)}`,
  ];
  elements.strategyBacktestSummary.innerHTML = `
    <div class="meta-row">${metaItems.map((m) => `<div class="meta-item">${m}</div>`).join("")}</div>
    <div class="kpi-grid">
      <div class="kpi-card"><div class="kpi-title">總報酬</div><div class="kpi-value">${fmtPercent(
        stats.total_return
      )}</div></div>
      <div class="kpi-card"><div class="kpi-title">最大回撤</div><div class="kpi-value warn">${fmtPercent(
        stats.max_drawdown
      )}</div></div>
      <div class="kpi-card"><div class="kpi-title">勝率</div><div class="kpi-value">${fmtPercent(
        stats.win_rate
      )}</div></div>
      <div class="kpi-card"><div class="kpi-title">筆數</div><div class="kpi-value">${fmtInt(
        stats.trade_count
      )}</div></div>
      <div class="kpi-card"><div class="kpi-title">平均獲利</div><div class="kpi-value">${fmtPercent(
        stats.avg_gain
      )}</div></div>
      <div class="kpi-card"><div class="kpi-title">平均虧損</div><div class="kpi-value warn">${fmtPercent(
        stats.avg_loss ? -Math.abs(stats.avg_loss) : stats.avg_loss
      )}</div></div>
      <div class="kpi-card"><div class="kpi-title">盈虧比</div><div class="kpi-value">${fmtNumber(
        stats.profit_factor
      )}</div></div>
    </div>
  `;
  renderStrategyBacktestTrades(result.trades || []);
  renderEquityChart(result.equity_curve || [], result.trades || []);
  state.lastStrategyBacktest = rec;
};

const runStrategyBacktest = async () => {
  const res = collectStrategyBacktestPayload();
  if (!res.ok) {
    setMessage(elements.strategyBacktestMessage, res.error, "error");
    return;
  }
  try {
    requireLogin();
    setMessage(elements.strategyBacktestMessage, "回測執行中...", "info");
    const data = await api(`/api/admin/strategies/${res.strategyID}/backtest`, {
      method: "POST",
      body: JSON.stringify(res.payload),
    });
    const record = data.result;
    setMessage(elements.strategyBacktestMessage, "回測完成並已保存", "good");
    renderStrategyBacktestSummary(record);
    state.lastStrategyBacktest = record;
    state.strategyBacktestPage.page = 1;
    await loadStrategyBacktests(record.strategy_id);
    logActivity("策略回測", `策略 ${res.strategyID} · ${res.payload.start_date}~${res.payload.end_date}`);
  } catch (err) {
    setMessage(elements.strategyBacktestMessage, `回測失敗：${err.message}`, "error");
    renderStrategyBacktestSummary(null);
  }
};

const applyBacktestParams = (params = {}) => {
  if (elements.strategyBtStart && params.start_date) elements.strategyBtStart.value = fmtDate(params.start_date);
  if (elements.strategyBtEnd && params.end_date) elements.strategyBtEnd.value = fmtDate(params.end_date);
  if (elements.strategyBtEquity && params.initial_equity != null) elements.strategyBtEquity.value = params.initial_equity;
  if (elements.strategyBtPriceMode && params.price_mode) elements.strategyBtPriceMode.value = params.price_mode;
  if (elements.strategyBtFees && params.fees_pct != null) elements.strategyBtFees.value = (params.fees_pct * 100).toFixed(2);
  if (elements.strategyBtSlippage && params.slippage_pct != null) elements.strategyBtSlippage.value = (params.slippage_pct * 100).toFixed(2);
  if (elements.strategyBtStop && params.stop_loss_pct != null) elements.strategyBtStop.value = (params.stop_loss_pct * 100).toFixed(2);
  if (elements.strategyBtTake && params.take_profit_pct != null) elements.strategyBtTake.value = (params.take_profit_pct * 100).toFixed(2);
  if (elements.strategyBtDailyLoss && params.max_daily_loss_pct != null) elements.strategyBtDailyLoss.value = (params.max_daily_loss_pct * 100).toFixed(2);
  if (elements.strategyBtCoolDown && params.cool_down_days != null) elements.strategyBtCoolDown.value = params.cool_down_days;
  if (elements.strategyBtMinHold && params.min_hold_days != null) elements.strategyBtMinHold.value = params.min_hold_days;
  if (elements.strategyBtMaxPos && params.max_positions != null) elements.strategyBtMaxPos.value = params.max_positions;
};

const getHistoryFilter = () => {
  const start = elements.strategyBtHistStart?.value;
  const end = elements.strategyBtHistEnd?.value;
  const version = elements.strategyBtHistVersion?.value;
  const pageSize = Number(elements.strategyBtHistPageSize?.value || 10);
  return {
    start: start ? new Date(start) : null,
    end: end ? new Date(end) : null,
    version: version ? Number(version) : null,
    pageSize: !Number.isNaN(pageSize) && pageSize > 0 ? pageSize : 10,
  };
};

const applyHistoryFilters = () => {
  const { start, end, version, pageSize } = getHistoryFilter();
  state.strategyBacktestPage.size = pageSize;
  let list = state.strategyBacktestsAll || [];
  if (version) {
    list = list.filter((r) => r.strategy_version === version);
  }
  if (start || end) {
    list = list.filter((r) => {
      const sd = r.params?.start_date ? new Date(r.params.start_date) : null;
      const ed = r.params?.end_date ? new Date(r.params.end_date) : null;
      if (start && sd && sd < start) return false;
      if (end && ed && ed > end) return false;
      return true;
    });
  }
  state.strategyBacktests = list;
};

const renderStrategyBacktestHistory = () => {
  if (!elements.strategyBacktestHistory) return;
  if (!state.strategyBacktestsAll.length) {
    renderEmptyState(elements.strategyBacktestHistory, "尚無歷史回測紀錄");
    if (elements.strategyBtHistPageInfo) elements.strategyBtHistPageInfo.textContent = "";
    return;
  }
  applyHistoryFilters();
  const { page, size } = state.strategyBacktestPage;
  const total = state.strategyBacktests.length;
  const totalPages = Math.max(1, Math.ceil(total / size));
  const currentPage = Math.min(page, totalPages);
  state.strategyBacktestPage.page = currentPage;
  const startIdx = (currentPage - 1) * size;
  const pageRows = state.strategyBacktests.slice(startIdx, startIdx + size);

  if (!pageRows.length) {
    renderEmptyState(elements.strategyBacktestHistory, "範圍內無回測紀錄");
  } else {
    const rows = pageRows.map((r, idx) => {
      const params = r.params || {};
      const stats = r.result?.stats || {};
      return {
        idx: startIdx + idx,
        created_at: r.created_at ? timeFormat.format(new Date(r.created_at)) : "—",
        period: `${fmtDate(params.start_date)} ~ ${fmtDate(params.end_date)}`,
        version: r.strategy_version || "-",
        total_return: stats.total_return,
        max_dd: stats.max_drawdown,
        win_rate: stats.win_rate,
        trades: stats.trade_count,
      };
    });
    const cols = [
      { key: "created_at", label: "建立時間" },
      { key: "period", label: "區間" },
      { key: "version", label: "版次" },
      { key: "total_return", label: "總報酬", format: fmtPercent, delta: true },
      { key: "max_dd", label: "最大回撤", format: fmtPercent, delta: true },
      { key: "win_rate", label: "勝率", format: fmtPercent },
      { key: "trades", label: "筆數", format: fmtInt },
      {
        key: "actions",
        label: "操作",
        format: (_, row) =>
          `<div class="btn-group">
            <button class="ghost btn-sm" data-action="preview" data-idx="${row.idx}">查看</button>
            <button class="ghost btn-sm" data-action="apply" data-idx="${row.idx}">套用參數</button>
            <button class="ghost btn-sm" data-action="export_trades" data-idx="${row.idx}">匯出交易</button>
            <button class="ghost btn-sm" data-action="export_equity" data-idx="${row.idx}">匯出淨值</button>
            <button class="ghost btn-sm" data-action="report" data-idx="${row.idx}">產生報告</button>
          </div>`,
      },
    ];
    const tableHTML = renderTableHTML(rows, cols);
    elements.strategyBacktestHistory.innerHTML = tableHTML;
    bindBacktestHistoryActions();
  }

  if (elements.strategyBtHistPageInfo) {
    elements.strategyBtHistPageInfo.textContent = `第 ${currentPage} / ${totalPages} 頁（共 ${fmtInt(
      total
    )} 筆）`;
  }
};

const renderTableHTML = (rows, cols) => {
  if (!rows.length) return `<div class="empty-state">尚無資料</div>`;
  const thead = cols.map((c) => `<th>${c.label}</th>`).join("");
  const tbody = rows
    .map((row) => {
      const tds = cols
        .map((c) => {
          const val = row[c.key];
          let content = c.format ? c.format(val, row) : fmtText(val);
          if (c.delta) content = `<span class="delta ${deltaClass(val)}">${content}</span>`;
          return `<td>${content}</td>`;
        })
        .join("");
      return `<tr>${tds}</tr>`;
    })
    .join("");
  return `<table><thead><tr>${thead}</tr></thead><tbody>${tbody}</tbody></table>`;
};

const bindBacktestHistoryActions = () => {
  if (!elements.strategyBacktestHistory) return;
  elements.strategyBacktestHistory.querySelectorAll("[data-action]").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      const idx = Number(btn.dataset.idx || -1);
      const record = state.strategyBacktestsAll[idx];
      if (!record) return;
      const action = btn.dataset.action;
      if (action === "preview") {
        renderStrategyBacktestSummary(record);
        state.lastStrategyBacktest = record;
      }
      if (action === "apply") {
        applyBacktestParams(record.params || {});
        setMessage(elements.strategyBacktestMessage, "已套用回測參數至表單", "info");
      }
      if (action === "export_trades") {
        if (!record.result?.trades || !record.result.trades.length) {
          setMessage(elements.strategyBacktestMessage, "該筆回測無交易資料可匯出", "warn");
          return;
        }
        const rows = record.result.trades.map((t) => ({
          entry_date: fmtDate(t.entry_date),
          exit_date: fmtDate(t.exit_date),
          entry_price: t.entry_price,
          exit_price: t.exit_price,
          pnl_usdt: t.pnl_usdt,
          pnl_pct: t.pnl_pct,
          hold_days: t.hold_days,
          reason: t.reason,
        }));
        const cols = [
          { key: "entry_date", label: "entry_date" },
          { key: "exit_date", label: "exit_date" },
          { key: "entry_price", label: "entry_price" },
          { key: "exit_price", label: "exit_price" },
          { key: "pnl_usdt", label: "pnl_usdt" },
          { key: "pnl_pct", label: "pnl_pct" },
          { key: "hold_days", label: "hold_days" },
          { key: "reason", label: "reason" },
        ];
        const csv = toCsv(rows, cols);
        downloadCsv(`strategy_${record.strategy_id || "unknown"}_trades.csv`, csv);
        setMessage(elements.strategyBacktestMessage, "已匯出該筆交易 CSV", "good");
      }
      if (action === "export_equity") {
        if (!record.result?.equity_curve || !record.result.equity_curve.length) {
          setMessage(elements.strategyBacktestMessage, "該筆回測無淨值資料可匯出", "warn");
          return;
        }
        const rows = record.result.equity_curve.map((p) => ({
          date: fmtDate(p.date),
          equity: p.equity,
        }));
        const cols = [
          { key: "date", label: "date" },
          { key: "equity", label: "equity" },
        ];
        const csv = toCsv(rows, cols);
        downloadCsv(`strategy_${record.strategy_id || "unknown"}_equity.csv`, csv);
        setMessage(elements.strategyBacktestMessage, "已匯出該筆淨值 CSV", "good");
      }
      if (action === "report") {
        const params = record.params || {};
        if (!params.start_date || !params.end_date) {
          setMessage(elements.strategyBacktestMessage, "缺少期間，無法產生報告", "warn");
          return;
        }
        const stats = record.result?.stats || {};
        const env = params.strategy?.env || "test";
        const payload = {
          env,
          period_start: fmtDate(params.start_date),
          period_end: fmtDate(params.end_date),
          summary: {
            total_return: stats.total_return,
            max_drawdown: stats.max_drawdown,
            win_rate: stats.win_rate,
            trade_count: stats.trade_count,
            profit_factor: stats.profit_factor,
          },
          trades_ref: record.result?.trades || [],
        };
        requireLogin();
        api(`/api/admin/strategies/${record.strategy_id}/reports`, {
          method: "POST",
          body: JSON.stringify(payload),
        })
          .then(() => {
            setMessage(elements.strategyBacktestMessage, "已產生報告", "good");
          })
          .catch((err) => {
            setMessage(elements.strategyBacktestMessage, `產生報告失敗：${err.message}`, "error");
          });
      }
    });
  });
};

const loadStrategyBacktests = async (strategyID) => {
  if (!elements.strategyBacktestHistory || !strategyID) return;
  try {
    renderEmptyState(elements.strategyBacktestHistory, "載入回測紀錄中...");
    const res = await api(`/api/admin/strategies/${strategyID}/backtests`);
    state.strategyBacktestsAll = res.backtests || [];
    state.strategyBacktestPage.page = 1;
    renderStrategyBacktestHistory();
  } catch (err) {
    renderEmptyState(elements.strategyBacktestHistory, `載入失敗：${err.message}`);
  }
};

const toCsv = (rows, columns) => {
  const headers = columns.map((c) => c.label);
  const lines = [headers.join(",")];
  rows.forEach((row) => {
    const cells = columns.map((c) => {
      const val = row[c.key];
      if (val === null || val === undefined) return "";
      if (typeof val === "string" && (val.includes(",") || val.includes("\"") || val.includes("\n"))) {
        return `"${val.replace(/"/g, '""')}"`;
      }
      return val;
    });
    lines.push(cells.join(","));
  });
  return lines.join("\n");
};

const downloadCsv = (filename, content) => {
  const blob = new Blob([content], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
};

const exportBacktestTrades = () => {
  const trades = state.lastStrategyTrades || [];
  if (!trades.length) {
    setMessage(elements.strategyBacktestMessage, "無交易資料可匯出", "warn");
    return;
  }
  const rows = trades.map((t) => ({
    entry_date: fmtDate(t.entry_date),
    exit_date: fmtDate(t.exit_date),
    entry_price: t.entry_price,
    exit_price: t.exit_price,
    pnl_usdt: t.pnl_usdt,
    pnl_pct: t.pnl_pct,
    hold_days: t.hold_days,
    reason: t.reason,
  }));
  const cols = [
    { key: "entry_date", label: "entry_date" },
    { key: "exit_date", label: "exit_date" },
    { key: "entry_price", label: "entry_price" },
    { key: "exit_price", label: "exit_price" },
    { key: "pnl_usdt", label: "pnl_usdt" },
    { key: "pnl_pct", label: "pnl_pct" },
    { key: "hold_days", label: "hold_days" },
    { key: "reason", label: "reason" },
  ];
  const csv = toCsv(rows, cols);
  downloadCsv("strategy_trades.csv", csv);
  setMessage(elements.strategyBacktestMessage, "已匯出交易 CSV", "good");
};

const exportBacktestEquity = () => {
  const equity = state.lastStrategyEquity || [];
  if (!equity.length) {
    setMessage(elements.strategyBacktestMessage, "無淨值資料可匯出", "warn");
    return;
  }
  const rows = equity.map((p) => ({
    date: fmtDate(p.date),
    equity: p.equity,
  }));
  const cols = [
    { key: "date", label: "date" },
    { key: "equity", label: "equity" },
  ];
  const csv = toCsv(rows, cols);
  downloadCsv("strategy_equity.csv", csv);
  setMessage(elements.strategyBacktestMessage, "已匯出淨值 CSV", "good");
};


const conditionTypeOptions = [
  { value: "numeric", label: "數值" },
  { value: "tags", label: "標籤" },
  { value: "category", label: "分類" },
  { value: "symbols", label: "代碼" },
];

const numericFieldOptions = [
  { value: "score", label: "score" },
  { value: "return5", label: "return5" },
  { value: "return20", label: "return20" },
  { value: "return60", label: "return60" },
  { value: "volume_multiple", label: "volume_multiple" },
  { value: "deviation20", label: "deviation20" },
  { value: "range_pos20", label: "range_pos20" },
  { value: "amplitude", label: "amplitude" },
  { value: "avg_amplitude20", label: "avg_amplitude20" },
  { value: "ma5", label: "ma5" },
  { value: "ma10", label: "ma10" },
  { value: "ma20", label: "ma20" },
  { value: "ma60", label: "ma60" },
  { value: "close", label: "close" },
];

const conditionOps = [
  { value: "gte", label: "≥" },
  { value: "lte", label: "≤" },
  { value: "gt", label: ">" },
  { value: "lt", label: "<" },
  { value: "eq", label: "=" },
  { value: "between", label: "區間" },
];

const categoryFieldOptions = [
  { value: "market", label: "市場 (TWSE/TPEx)" },
  { value: "industry", label: "產業" },
];

const tagOptions = ["短期強勢", "量能放大", "接近前高", "接近前低", "高波動", "低波動"];

function toPercent(raw, label, allowEmpty = true, defaultValue = null) {
  if (raw === "" || raw === null || raw === undefined) {
    return { ok: true, value: allowEmpty ? defaultValue : null };
  }
  const num = Number(raw);
  if (Number.isNaN(num) || num < 0) {
    return { ok: false, error: `${label}需為非負數字` };
  }
  return { ok: true, value: num / 100 };
}

function createOptions(select, options, selected) {
  select.innerHTML = "";
  options.forEach((opt) => {
    const option = document.createElement("option");
    option.value = opt.value;
    option.textContent = opt.label;
    select.appendChild(option);
  });
  if (selected) {
    select.value = selected;
  }
}

function parseList(text) {
  return (text || "")
    .split(/[,，\n]/)
    .map((s) => s.trim())
    .filter(Boolean);
}

function addConditionRow(container, defaults = {}) {
  if (!container) return;
  const row = document.createElement("div");
  row.className = "condition-card";
  const head = document.createElement("div");
  head.className = "condition-card-head";
  const typeSelect = document.createElement("select");
  typeSelect.dataset.type = "true";
  createOptions(typeSelect, conditionTypeOptions, defaults.type || "numeric");
  const removeBtn = document.createElement("button");
  removeBtn.type = "button";
  removeBtn.className = "ghost btn-sm";
  removeBtn.textContent = "移除";
  head.appendChild(typeSelect);
  head.appendChild(removeBtn);

  const body = document.createElement("div");
  body.className = "condition-body";
  body.dataset.conditionBody = "true";
  row.appendChild(head);
  row.appendChild(body);

  const renderBody = (type, dft = {}) => {
    body.innerHTML = "";
    if (type === "numeric") {
      const fieldLabel = document.createElement("label");
      fieldLabel.textContent = "欄位";
      const fieldSelect = document.createElement("select");
      fieldSelect.dataset.field = "true";
      createOptions(fieldSelect, numericFieldOptions, dft.field || "score");
      fieldLabel.appendChild(fieldSelect);

      const opLabel = document.createElement("label");
      opLabel.textContent = "運算子";
      const opSelect = document.createElement("select");
      opSelect.dataset.op = "true";
      createOptions(opSelect, conditionOps, dft.op || "gte");
      opLabel.appendChild(opSelect);

      const valueLabel = document.createElement("label");
      valueLabel.textContent = "數值";
      const valueInput = document.createElement("input");
      valueInput.type = "number";
      valueInput.step = "0.01";
      valueInput.dataset.value = "true";
      valueInput.value = dft.value ?? "";
      valueLabel.appendChild(valueInput);

      const rangeWrapper = document.createElement("div");
      rangeWrapper.className = "inline";
      const minLabel = document.createElement("label");
      minLabel.textContent = "最小值";
      const minInput = document.createElement("input");
      minInput.type = "number";
      minInput.step = "0.01";
      minInput.dataset.min = "true";
      minInput.value = dft.min ?? "";
      minLabel.appendChild(minInput);
      const maxLabel = document.createElement("label");
      maxLabel.textContent = "最大值";
      const maxInput = document.createElement("input");
      maxInput.type = "number";
      maxInput.step = "0.01";
      maxInput.dataset.max = "true";
      maxInput.value = dft.max ?? "";
      maxLabel.appendChild(maxInput);
      rangeWrapper.appendChild(minLabel);
      rangeWrapper.appendChild(maxLabel);

      body.appendChild(fieldLabel);
      body.appendChild(opLabel);
      body.appendChild(valueLabel);
      body.appendChild(rangeWrapper);

      const toggleRange = () => {
        const isBetween = opSelect.value === "between";
        valueLabel.style.display = isBetween ? "none" : "block";
        rangeWrapper.style.display = isBetween ? "flex" : "none";
      };
      toggleRange();
      opSelect.addEventListener("change", toggleRange);
    } else if (type === "category") {
      const fieldLabel = document.createElement("label");
      fieldLabel.textContent = "欄位";
      const fieldSelect = document.createElement("select");
      fieldSelect.dataset.categoryField = "true";
      createOptions(fieldSelect, categoryFieldOptions, dft.field || "market");
      fieldLabel.appendChild(fieldSelect);

      const valuesLabel = document.createElement("label");
      valuesLabel.textContent = "值（以逗號分隔）";
      const valuesInput = document.createElement("input");
      valuesInput.type = "text";
      valuesInput.placeholder = "TWSE,TPEx 或產業名稱";
      valuesInput.dataset.categoryValues = "true";
      valuesInput.value = dft.values?.join(",") || "";
      valuesLabel.appendChild(valuesInput);

      body.appendChild(fieldLabel);
      body.appendChild(valuesLabel);
    } else if (type === "tags") {
      const includeAnyLabel = document.createElement("label");
      includeAnyLabel.textContent = "包含任一標籤（逗號分隔）";
      const includeAnyInput = document.createElement("input");
      includeAnyInput.type = "text";
      includeAnyInput.placeholder = tagOptions.join("，");
      includeAnyInput.dataset.includeAny = "true";
      includeAnyInput.value = (dft.includeAny || []).join(",");
      includeAnyLabel.appendChild(includeAnyInput);

      const includeAllLabel = document.createElement("label");
      includeAllLabel.textContent = "需同時包含（逗號分隔，可留空）";
      const includeAllInput = document.createElement("input");
      includeAllInput.type = "text";
      includeAllInput.placeholder = "短期強勢,量能放大";
      includeAllInput.dataset.includeAll = "true";
      includeAllInput.value = (dft.includeAll || []).join(",");
      includeAllLabel.appendChild(includeAllInput);

      const excludeLabel = document.createElement("label");
      excludeLabel.textContent = "需排除（逗號分隔，可留空）";
      const excludeInput = document.createElement("input");
      excludeInput.type = "text";
      excludeInput.placeholder = "高波動";
      excludeInput.dataset.excludeAny = "true";
      excludeInput.value = (dft.excludeAny || []).join(",");
      excludeLabel.appendChild(excludeInput);

      body.appendChild(includeAnyLabel);
      body.appendChild(includeAllLabel);
      body.appendChild(excludeLabel);
    } else if (type === "symbols") {
      const includeLabel = document.createElement("label");
      includeLabel.textContent = "限定代碼（逗號分隔）";
      const includeInput = document.createElement("input");
      includeInput.type = "text";
      includeInput.placeholder = "2330, 2317";
      includeInput.dataset.includeSymbols = "true";
      includeInput.value = (dft.include || []).join(",");
      includeLabel.appendChild(includeInput);

      const excludeLabel = document.createElement("label");
      excludeLabel.textContent = "排除代碼（逗號分隔，可留空）";
      const excludeInput = document.createElement("input");
      excludeInput.type = "text";
      excludeInput.placeholder = "0050";
      excludeInput.dataset.excludeSymbols = "true";
      excludeInput.value = (dft.exclude || []).join(",");
      excludeLabel.appendChild(excludeInput);

      body.appendChild(includeLabel);
      body.appendChild(excludeLabel);
    }

    body.querySelectorAll("input, select").forEach((el) => {
      el.addEventListener("input", updateStrategyPreview);
    });
  };

  renderBody(typeSelect.value, defaults);

  typeSelect.addEventListener("change", () => {
    renderBody(typeSelect.value, {});
    updateStrategyPreview();
  });
  removeBtn.addEventListener("click", () => {
    row.remove();
    updateStrategyPreview();
  });

  container.appendChild(row);
}

function resetConditionRows(container, defaults = []) {
  if (!container) return;
  container.innerHTML = "";
  defaults.forEach((d) => addConditionRow(container, d));
}

function collectConditions(container, logicValue, strict = true) {
  if (!container) return { ok: false, error: "條件容器遺失" };
  const logic = logicValue || "AND";
  if (logic !== "AND" && logic !== "OR") {
    return { ok: false, error: "邏輯需為 AND 或 OR" };
  }
  const conditions = [];
  const rows = Array.from(container.querySelectorAll(".condition-card"));
  if (rows.length === 0) {
    return { ok: false, error: "請至少新增一條條件" };
  }

  for (const row of rows) {
    const type = row.querySelector("select[data-type]")?.value || "numeric";
    if (type === "numeric") {
      const field = (row.querySelector("[data-field]")?.value || "").trim();
      const op = row.querySelector("[data-op]")?.value || "";
      if (!field) return { ok: false, error: "數值條件欄位不可空白" };
      if (!op) return { ok: false, error: "請選擇運算子" };
      if (op === "between") {
        const min = Number(row.querySelector("[data-min]")?.value ?? NaN);
        const max = Number(row.querySelector("[data-max]")?.value ?? NaN);
        if (Number.isNaN(min) || Number.isNaN(max)) return { ok: false, error: "區間條件需填寫最小值與最大值" };
        if (strict && min > max) return { ok: false, error: "區間最小值需小於等於最大值" };
        conditions.push({ type, numeric: { field, op, min, max } });
      } else {
        const value = Number(row.querySelector("[data-value]")?.value ?? NaN);
        if (Number.isNaN(value)) return { ok: false, error: "條件數值需為數字" };
        conditions.push({ type, numeric: { field, op, value } });
      }
    } else if (type === "category") {
      const field = row.querySelector("[data-category-field]")?.value || "";
      const values = parseList(row.querySelector("[data-category-values]")?.value || "");
      if (strict && values.length === 0) return { ok: false, error: "分類條件至少填一個值" };
      conditions.push({ type, category: { field, values } });
    } else if (type === "tags") {
      const includeAny = parseList(row.querySelector("[data-include-any]")?.value || "");
      const includeAll = parseList(row.querySelector("[data-include-all]")?.value || "");
      const excludeAny = parseList(row.querySelector("[data-exclude-any]")?.value || "");
      if (strict && includeAny.length === 0 && includeAll.length === 0 && excludeAny.length === 0) {
        return { ok: false, error: "標籤條件至少填一個包含或排除" };
      }
      conditions.push({ type, tags: { includeAny, includeAll, excludeAny } });
    } else if (type === "symbols") {
      const include = parseList(row.querySelector("[data-include-symbols]")?.value || "");
      const exclude = parseList(row.querySelector("[data-exclude-symbols]")?.value || "");
      if (strict && include.length === 0 && exclude.length === 0) {
        return { ok: false, error: "代碼條件請至少輸入一個代碼" };
      }
      conditions.push({ type, symbols: { include, exclude } });
    } else {
      return { ok: false, error: `不支援的條件類型：${type}` };
    }
  }

  return { ok: true, value: { logic, conditions } };
}

function collectRiskSettings(strict = true) {
  const order_size_mode = elements.orderSizeMode?.value || "fixed_usdt";
  const order_size_value = Number(elements.orderSizeValue?.value || 0);
  const price_mode = elements.priceMode?.value || "next_open";
  const fees_pct_raw = elements.feesPct?.value;
  const slippage_pct_raw = elements.slippagePct?.value;
  const stop_loss_pct_raw = elements.stopLossPct?.value;
  const take_profit_pct_raw = elements.takeProfitPct?.value;
  const cool_down_days = Number(elements.coolDownDays?.value || 0);
  const min_hold_days = Number(elements.minHoldDays?.value || 0);
  const max_positions = Number(elements.maxPositions?.value || 1);

  if (strict) {
    if (order_size_mode !== "fixed_usdt" && order_size_mode !== "percent_of_equity") {
      return { ok: false, error: "下單模式需為固定金額或資金比例" };
    }
    if (!order_size_value || Number.isNaN(order_size_value) || order_size_value <= 0) {
      return { ok: false, error: "下單金額/比例需為正數" };
    }
    if (order_size_mode === "percent_of_equity" && order_size_value > 1) {
      return { ok: false, error: "資金比例請填 0-1 之間" };
    }
    if (!price_mode) {
      return { ok: false, error: "請選擇成交價模式" };
    }
    if (Number.isNaN(cool_down_days) || cool_down_days < 0) {
      return { ok: false, error: "冷卻天數需為 0 或正整數" };
    }
    if (Number.isNaN(min_hold_days) || min_hold_days < 0) {
      return { ok: false, error: "最少持有天數需為 0 或正整數" };
    }
    if (Number.isNaN(max_positions) || max_positions < 1) {
      return { ok: false, error: "最多持倉需為大於 0 的整數" };
    }
  }

  const feesPctRes = toPercent(fees_pct_raw, "手續費", true, 0);
  if (!feesPctRes.ok) return { ok: false, error: feesPctRes.error };
  const slippagePctRes = toPercent(slippage_pct_raw, "滑價", true, 0);
  if (!slippagePctRes.ok) return { ok: false, error: slippagePctRes.error };
  const stopLossRes = toPercent(stop_loss_pct_raw, "停損", true, null);
  if (!stopLossRes.ok) return { ok: false, error: stopLossRes.error };
  const takeProfitRes = toPercent(take_profit_pct_raw, "停利", true, null);
  if (!takeProfitRes.ok) return { ok: false, error: takeProfitRes.error };

  return {
    ok: true,
    value: {
      order_size_mode,
      order_size_value,
      price_mode,
      fees_pct: feesPctRes.value ?? 0,
      slippage_pct: slippagePctRes.value ?? 0,
      stop_loss_pct: stopLossRes.value,
      take_profit_pct: takeProfitRes.value,
      cool_down_days,
      min_hold_days,
      max_positions,
    },
  };
}

function buildStrategyPayload(strict = true) {
  const name = (elements.createStrategyName?.value || "").trim();
  const base_symbol = (elements.createStrategySymbol?.value || "BTCUSDT").trim();
  const timeframe = (elements.createStrategyTimeframe?.value || "1d").trim();
  const env = elements.createStrategyEnv?.value || "both";

  if (strict && !name) {
    return { ok: false, error: "請填寫策略名稱" };
  }

  const buyRes = collectConditions(elements.buyConditions, elements.buyLogic?.value, strict);
  if (!buyRes.ok) {
    return { ok: false, error: `買入條件錯誤：${buyRes.error}` };
  }
  const sellRes = collectConditions(elements.sellConditions, elements.sellLogic?.value, strict);
  if (!sellRes.ok) {
    return { ok: false, error: `賣出條件錯誤：${sellRes.error}` };
  }
  const riskRes = collectRiskSettings(strict);
  if (!riskRes.ok) {
    return { ok: false, error: `風控設定錯誤：${riskRes.error}` };
  }

  return {
    ok: true,
    value: {
      name,
      base_symbol,
      timeframe,
      env,
      buy_conditions: buyRes.value,
      sell_conditions: sellRes.value,
      risk_settings: riskRes.value,
    },
  };
}

function updateStrategyPreview() {
  if (!elements.createStrategyPreview) return;
  const res = buildStrategyPayload(false);
  if (res.ok) {
    elements.createStrategyPreview.value = JSON.stringify(res.value, null, 2);
  } else {
    elements.createStrategyPreview.value = `目前預覽無法產生：${res.error}`;
  }
}

const loadStrategyTemplate = () => {
  if (elements.createStrategyName) elements.createStrategyName.value = "";
  if (elements.createStrategySymbol) elements.createStrategySymbol.value = "BTCUSDT";
  if (elements.createStrategyTimeframe) elements.createStrategyTimeframe.value = "1d";
  if (elements.createStrategyEnv) elements.createStrategyEnv.value = "both";
  if (elements.buyLogic) elements.buyLogic.value = "AND";
  if (elements.sellLogic) elements.sellLogic.value = "AND";

  resetConditionRows(elements.buyConditions, [{ field: "score", op: "gte", value: 60 }]);
  resetConditionRows(elements.sellConditions, [{ field: "score", op: "lte", value: 40 }]);

  if (elements.orderSizeMode) elements.orderSizeMode.value = "fixed_usdt";
  if (elements.orderSizeValue) elements.orderSizeValue.value = 1000;
  if (elements.priceMode) elements.priceMode.value = "next_open";
  if (elements.feesPct) elements.feesPct.value = 0.1;
  if (elements.slippagePct) elements.slippagePct.value = 0.1;
  if (elements.stopLossPct) elements.stopLossPct.value = 3;
  if (elements.takeProfitPct) elements.takeProfitPct.value = 6;
  if (elements.coolDownDays) elements.coolDownDays.value = 1;
  if (elements.minHoldDays) elements.minHoldDays.value = 1;
  if (elements.maxPositions) elements.maxPositions.value = 1;

  updateStrategyPreview();
  setMessage(elements.createStrategyMessage, "已載入範例設定", "info");
};

const createStrategy = async () => {
  const res = buildStrategyPayload(true);
  if (!res.ok) {
    setMessage(elements.createStrategyMessage, res.error, "error");
    return;
  }
  const payload = res.value;
  try {
    requireLogin();
    await api("/api/admin/strategies", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    setMessage(elements.createStrategyMessage, "策略建立成功", "good");
    logActivity("建立策略", `名稱 ${payload.name} · env ${payload.env}`);
    await loadStrategies();
  } catch (err) {
    setMessage(elements.createStrategyMessage, `建立失敗：${err.message}`, "error");
  }
};

const activateStrategy = async (id, env) => {
  await api(`/api/admin/strategies/${id}/activate`, {
    method: "POST",
    body: JSON.stringify({ env }),
  });
  logActivity("啟用策略", `ID ${id} · env ${env || "test"}`);
};

const deactivateStrategy = async (id) => {
  await api(`/api/admin/strategies/${id}/deactivate`, {
    method: "POST",
  });
  logActivity("停用策略", `ID ${id}`);
};

const runStrategyOnce = async (id, env) => {
  const qs = env ? `?env=${env}` : "";
  const res = await api(`/api/admin/strategies/${id}/run${qs}`, { method: "POST" });
  const trades = res.trades || [];
  logActivity("策略試跑", `ID ${id} · env ${env} · 筆數 ${fmtInt(trades.length)}`);
  setStatus(`試跑完成：產生 ${fmtInt(trades.length)} 筆交易`, trades.length ? "good" : "warn");
   state.lastRunTrades = trades;
   renderRunTrades(trades, env);
  return trades;
};

const bindStrategyActions = () => {
  if (!elements.strategyTable) return;
  elements.strategyTable.querySelectorAll("[data-action]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      try {
        requireLogin();
        const id = btn.dataset.id;
        const action = btn.dataset.action;
        const env = btn.dataset.env || "test";
        if (!id) return;
        setStatus("處理中...", "warn");
        if (action === "activate") {
          await activateStrategy(id, env);
        } else if (action === "deactivate") {
          await deactivateStrategy(id);
        } else if (action === "run") {
          await runStrategyOnce(id, env);
        }
        await loadStrategies();
        setStatus("操作完成", "good");
      } catch (err) {
        setStatus(`操作失敗：${err.message}`, "warn");
      }
    });
  });
};

const renderTradeTable = (items = []) => {
  if (!elements.tradeTable) return;
  if (!items.length) {
    renderEmptyState(elements.tradeTable, "尚未查詢");
    if (elements.tradeMeta) elements.tradeMeta.innerHTML = "";
    return;
  }
  const rows = items.map((t) => ({
    strategy_id: fmtText(t.strategy_id),
    env: fmtText(t.env),
    side: fmtText(t.side),
    entry_date: fmtText(t.entry_date),
    entry_price: fmtPrice(t.entry_price),
    exit_date: fmtText(t.exit_date),
    exit_price: fmtPrice(t.exit_price),
    pnl_usdt: fmtPrice(t.pnl_usdt),
    pnl_pct: t.pnl_pct != null ? fmtPercent(t.pnl_pct) : "—",
    reason: fmtText(t.reason),
  }));
  const cols = [
    { key: "strategy_id", label: "策略 ID", className: "mono" },
    { key: "env", label: "環境" },
    { key: "side", label: "方向" },
    { key: "entry_date", label: "進場日" },
    { key: "entry_price", label: "進場價", className: "mono" },
    { key: "exit_date", label: "出場日" },
    { key: "exit_price", label: "出場價", className: "mono" },
    { key: "pnl_usdt", label: "PNL (USDT)", className: "mono" },
    { key: "pnl_pct", label: "PNL%", className: "mono" },
    { key: "reason", label: "原因" },
  ];
  renderTable(elements.tradeTable, rows, cols);
  if (elements.tradeMeta) {
    elements.tradeMeta.innerHTML = `<div class="meta-item">共 ${fmtInt(items.length)} 筆交易</div>`;
  }
};

const renderPositionTable = (items = []) => {
  if (!elements.positionTable) return;
  if (!items.length) {
    renderEmptyState(elements.positionTable, "尚未查詢");
    if (elements.positionMeta) elements.positionMeta.innerHTML = "";
    return;
  }
  const rows = items.map((p) => ({
    strategy_id: fmtText(p.strategy_id),
    env: fmtText(p.env),
    entry_date: fmtText(p.entry_date),
    entry_price: fmtPrice(p.entry_price),
    size: fmtPrice(p.size),
    stop_loss: p.stop_loss != null ? fmtPrice(p.stop_loss) : "—",
    take_profit: p.take_profit != null ? fmtPrice(p.take_profit) : "—",
    status: fmtText(p.status),
  }));
  const cols = [
    { key: "strategy_id", label: "策略 ID", className: "mono" },
    { key: "env", label: "環境" },
    { key: "entry_date", label: "進場日" },
    { key: "entry_price", label: "進場價", className: "mono" },
    { key: "size", label: "部位金額", className: "mono" },
    { key: "stop_loss", label: "停損", className: "mono" },
    { key: "take_profit", label: "停利", className: "mono" },
    { key: "status", label: "狀態" },
  ];
  renderTable(elements.positionTable, rows, cols);
  if (elements.positionMeta) {
    elements.positionMeta.innerHTML = `<div class="meta-item">共 ${fmtInt(items.length)} 筆持倉</div>`;
  }
};

const loadPositions = async () => {
  if (!elements.positionForm) return;
  renderEmptyState(elements.positionTable, "載入持倉中...");
  const env = elements.positionEnv?.value || "";
  const qs = env ? `?env=${env}` : "";
  const res = await api(`/api/admin/positions${qs}`);
  renderPositionTable(res.positions || []);
  logActivity("查詢持倉", `筆數 ${fmtInt((res.positions || []).length)}`);
  setStatus("持倉列表已更新", "good");
};

const renderReportTable = (items = []) => {
  if (!elements.reportTable) return;
  state.reports = items || [];
  if (!items.length) {
    renderEmptyState(elements.reportTable, "尚未查詢");
    if (elements.reportMeta) elements.reportMeta.innerHTML = "";
    renderReportDetail(null);
    return;
  }
  const rows = items.map((r) => ({
    id: fmtText(r.id),
    env: fmtText(r.env),
    period: `${fmtText(r.period_start)} ~ ${fmtText(r.period_end)}`,
    summary: r.summary ? JSON.stringify(r.summary) : "—",
    created_at: r.created_at ? timeFormat.format(new Date(r.created_at)) : "—",
    raw: r,
  }));
  const cols = [
    { key: "id", label: "報告 ID", className: "mono" },
    { key: "env", label: "環境" },
    { key: "period", label: "期間" },
    { key: "summary", label: "摘要" },
    { key: "created_at", label: "建立時間" },
    {
      key: "actions",
      label: "操作",
      format: (_, row) => `<button class="ghost btn-sm" data-report-id="${row.id}">查看</button>`,
    },
  ];
  renderTable(elements.reportTable, rows, cols);
  if (elements.reportMeta) {
    elements.reportMeta.innerHTML = `<div class="meta-item">共 ${fmtInt(items.length)} 筆報告</div>`;
  }
  elements.reportTable.querySelectorAll("[data-report-id]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = btn.dataset.reportId;
      const report = state.reports.find((r) => r.id === id);
      renderReportDetail(report);
    });
  });
};

const loadReports = async () => {
  if (!elements.reportForm) return;
  const strategyId = (elements.reportStrategyId?.value || "").trim();
  if (!strategyId) {
    setStatus("請輸入策略 ID", "warn");
    renderEmptyState(elements.reportTable, "尚未查詢");
    renderReportDetail(null);
    return;
  }
  renderEmptyState(elements.reportTable, "載入報告中...");
  const res = await api(`/api/admin/strategies/${strategyId}/reports`);
  renderReportTable(res.reports || []);
  logActivity("查詢報告", `策略 ${strategyId} · 筆數 ${fmtInt((res.reports || []).length)}`);
  setStatus("報告列表已更新", "good");
};

const renderReportDetail = (report) => {
  state.selectedReport = report;
  if (!elements.reportDetail) return;
  if (!report) {
    renderEmptyState(elements.reportDetail, "請於上方列表點擊「查看」檢視報告內容");
    return;
  }
  const summaryPretty = report.summary ? JSON.stringify(report.summary, null, 2) : "—";
  const tradesCount = Array.isArray(report.trades_ref) ? report.trades_ref.length : 0;
  const tradesSample =
    tradesCount && report.trades_ref && report.trades_ref.slice
      ? JSON.stringify(report.trades_ref.slice(0, 3), null, 2)
      : "—";
  elements.reportDetail.innerHTML = `
    <div class="meta-row">
      <div class="meta-item">報告 ID：${fmtText(report.id)}</div>
      <div class="meta-item">環境：${fmtText(report.env)}</div>
      <div class="meta-item">區間：${fmtText(report.period_start)} ~ ${fmtText(report.period_end)}</div>
      <div class="meta-item">筆數：${fmtInt(tradesCount)}</div>
      <div class="meta-item">建立時間：${
        report.created_at ? timeFormat.format(new Date(report.created_at)) : "—"
      }</div>
    </div>
    <div class="code-block"><pre>${summaryPretty}</pre></div>
    <div class="code-block"><pre>交易樣本（前三筆）\n${tradesSample}</pre></div>
  `;
};

const renderLogTable = (items = []) => {
  if (!elements.logTable) return;
  if (!items.length) {
    renderEmptyState(elements.logTable, "尚未查詢");
    if (elements.logMeta) elements.logMeta.innerHTML = "";
    return;
  }
  const rows = items.map((l) => ({
    date: fmtText(l.date),
    env: fmtText(l.env),
    phase: fmtText(l.phase),
    message: fmtText(l.message),
    payload: l.payload ? JSON.stringify(l.payload) : "—",
    created_at: l.created_at ? timeFormat.format(new Date(l.created_at)) : "—",
  }));
  const cols = [
    { key: "date", label: "日期" },
    { key: "env", label: "環境" },
    { key: "phase", label: "階段" },
    { key: "message", label: "訊息" },
    { key: "payload", label: "內容" },
    { key: "created_at", label: "建立時間" },
  ];
  renderTable(elements.logTable, rows, cols);
  if (elements.logMeta) {
    elements.logMeta.innerHTML = `<div class="meta-item">共 ${fmtInt(items.length)} 筆日誌</div>`;
  }
};

const loadLogs = async () => {
  if (!elements.logForm) return;
  const strategyId = (elements.logStrategyId?.value || "").trim();
  if (!strategyId) {
    setStatus("請輸入策略 ID", "warn");
    renderEmptyState(elements.logTable, "尚未查詢");
    return;
  }
  renderEmptyState(elements.logTable, "載入日誌中...");
  const env = elements.logEnv?.value || "";
  const limit = elements.logLimit?.value || 50;
  const qs = new URLSearchParams();
  if (env) qs.append("env", env);
  if (limit) qs.append("limit", limit);
  const res = await api(`/api/admin/strategies/${strategyId}/logs${qs.toString() ? `?${qs.toString()}` : ""}`);
  renderLogTable(res.logs || []);
  logActivity("查詢策略日誌", `策略 ${strategyId} · 筆數 ${fmtInt((res.logs || []).length)}`);
  setStatus("日誌已更新", "good");
};

const loadTrades = async () => {
  if (!elements.tradeForm) return;
  renderEmptyState(elements.tradeTable, "載入交易中...");
  const qs = new URLSearchParams();
  const strategyId = (elements.tradeStrategyId?.value || "").trim();
  const env = elements.tradeEnv?.value || "";
  const start = elements.tradeStart?.value || "";
  const end = elements.tradeEnd?.value || "";
  if (strategyId) qs.append("strategy_id", strategyId);
  if (env) qs.append("env", env);
  if (start) qs.append("start_date", start);
  if (end) qs.append("end_date", end);
  const res = await api(`/api/admin/trades${qs.toString() ? `?${qs.toString()}` : ""}`);
  renderTradeTable(res.trades || []);
  logActivity("查詢交易紀錄", `筆數 ${fmtInt((res.trades || []).length)}`);
  setStatus("交易紀錄已更新", "good");
};

document.getElementById("loginForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  try {
    const email = document.getElementById("email").value;
    const password = document.getElementById("password").value;
    const res = await api("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
    state.token = res.access_token;
    setStatus(`已登入：${email}`, "good");
    setMessage(elements.loginMessage, "登入成功", "good");
    toggleProtectedSections(true);
    showSection("overview");
    logActivity("登入成功", `帳號 ${email}`);
    await loadChartHistory(true);
    fetchCombos();
    loadStrategies().catch(() => {});
    loadTrades().catch(() => {});
    loadPositions().catch(() => {});
  } catch (err) {
    setMessage(elements.loginMessage, `登入失敗：${err.message}`, "error");
    setStatus("未登入", "warn");
  }
});

if (elements.chartForm) {
  elements.chartForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    await loadChartHistory(false);
  });
}

if (elements.resetZoomBtn) {
  elements.resetZoomBtn.addEventListener("click", resetZoom);
}

async function loadChartHistory(auto = true) {
  try {
    requireLogin();
  } catch (err) {
    if (auto) return;
    renderChartPlaceholder("請先登入以載入走勢");
    return;
  }
  const start_date = elements.chartStart?.value;
  const end_date = elements.chartEnd?.value;
  if (!start_date || !end_date) {
    if (!auto) renderChartPlaceholder("請設定起始與結束日期");
    return;
  }
  try {
    renderChartLoading();
    const res = await api(
      `/api/analysis/history?symbol=BTCUSDT&start_date=${start_date}&end_date=${end_date}&only_success=true`
    );
    state.lastChart = res;
    renderHistoryChart(res, state.lastBacktest?.events || []);
    if (!auto) {
      logActivity("載入走勢圖", `區間 ${start_date} ~ ${end_date} · 筆數 ${fmtInt(res.total_count)}`);
    }
    scheduleAutoBacktest();
  } catch (err) {
    if (!auto) {
      renderChartPlaceholder(err.message);
    }
  }
}

if (elements.strategyForm) {
  elements.strategyForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      requireLogin();
      await loadStrategies();
    } catch (err) {
      setStatus(`載入策略失敗：${err.message}`, "warn");
    }
  });
}

if (elements.tradeForm) {
  elements.tradeForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      requireLogin();
      await loadTrades();
    } catch (err) {
      setStatus(`查詢交易失敗：${err.message}`, "warn");
    }
  });
}

if (elements.positionForm) {
  elements.positionForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      requireLogin();
      await loadPositions();
    } catch (err) {
      setStatus(`查詢持倉失敗：${err.message}`, "warn");
    }
  });
}

if (elements.reportForm) {
  elements.reportForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      requireLogin();
      await loadReports();
    } catch (err) {
      setStatus(`查詢報告失敗：${err.message}`, "warn");
    }
  });
}

if (elements.logForm) {
  elements.logForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      requireLogin();
      await loadLogs();
    } catch (err) {
      setStatus(`查詢日誌失敗：${err.message}`, "warn");
    }
  });
}

// 左側導航切換
navLinks.forEach((link) => {
  link.addEventListener("click", (e) => {
    e.preventDefault();
    const target = link.dataset.sectionTarget;
    if (!target) return;
    showSection(target);
  });
});

// 退場保險：委派監聽，避免事件漏掛
document.body.addEventListener("click", (e) => {
  const btn = e.target.closest("[data-section-target]");
  if (!btn) return;
  e.preventDefault();
  const target = btn.dataset.sectionTarget;
  if (target) showSection(target);
});

document.getElementById("summaryBtn").addEventListener("click", async () => {
  try {
    requireLogin();
    elements.summaryView.innerHTML = `<div class="empty-state">讀取中...</div>`;
    const res = await api("/api/analysis/summary");
    state.lastSummary = res;
    renderSummary(elements.summaryView, res);
    renderKpis();
    logActivity("取得走勢摘要", `交易日 ${fmtText(res.trade_date)}`);
  } catch (err) {
    elements.summaryView.innerHTML = `<div class="empty-state">${err.message}</div>`;
  }
});

if (elements.createStrategyForm) {
  elements.createStrategyForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    await createStrategy();
  });
}
if (elements.loadStrategyTemplate) {
  elements.loadStrategyTemplate.addEventListener("click", loadStrategyTemplate);
}

if (elements.strategyBacktestForm) {
  elements.strategyBacktestForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    await runStrategyBacktest();
  });
}

if (elements.strategyBacktestReload) {
  elements.strategyBacktestReload.addEventListener("click", async () => {
    try {
      requireLogin();
      await loadStrategies();
      setMessage(elements.strategyBacktestMessage, "策略列表已更新", "good");
    } catch (err) {
      setMessage(elements.strategyBacktestMessage, `載入策略失敗：${err.message}`, "error");
    }
  });
}

if (elements.strategyBacktestSelect) {
  elements.strategyBacktestSelect.addEventListener("change", async (e) => {
    const id = e.target.value;
    if (elements.strategyBacktestId) elements.strategyBacktestId.value = id;
    if (id) {
      await loadStrategyBacktests(id);
    } else {
      state.strategyBacktestsAll = [];
      renderStrategyBacktestHistory();
    }
  });
}

if (elements.strategyBacktestFilterForm) {
  elements.strategyBacktestFilterForm.addEventListener("submit", (e) => {
    e.preventDefault();
    state.strategyBacktestPage.page = 1;
    renderStrategyBacktestHistory();
  });
}

if (elements.strategyBtHistPrev) {
  elements.strategyBtHistPrev.addEventListener("click", () => {
    if (state.strategyBacktestPage.page > 1) {
      state.strategyBacktestPage.page -= 1;
      renderStrategyBacktestHistory();
    }
  });
}

if (elements.strategyBtHistNext) {
  elements.strategyBtHistNext.addEventListener("click", () => {
    const total = state.strategyBacktests ? state.strategyBacktests.length : 0;
    const totalPages = Math.max(1, Math.ceil(total / state.strategyBacktestPage.size));
    if (state.strategyBacktestPage.page < totalPages) {
      state.strategyBacktestPage.page += 1;
      renderStrategyBacktestHistory();
    }
  });
}

if (elements.exportBtTrades) {
  elements.exportBtTrades.addEventListener("click", exportBacktestTrades);
}
if (elements.exportBtEquity) {
  elements.exportBtEquity.addEventListener("click", exportBacktestEquity);
}
if (elements.strategyBtGoReports) {
  elements.strategyBtGoReports.addEventListener("click", async () => {
    try {
      requireLogin();
      const strategyId =
        state.lastStrategyBacktest?.strategy_id ||
        elements.strategyBacktestSelect?.value ||
        elements.strategyBacktestId?.value ||
        "";
      if (!strategyId) {
        setMessage(elements.strategyBacktestMessage, "請先選擇策略或執行回測", "warn");
        return;
      }
      if (elements.reportStrategyId) elements.reportStrategyId.value = strategyId;
      showSection("report");
      await loadReports();
      setStatus(`已切換至報告，策略 ${strategyId}`, "good");
    } catch (err) {
      setMessage(elements.strategyBacktestMessage, `切換報告失敗：${err.message}`, "error");
    }
  });
}

if (elements.addBuyCondition) {
  elements.addBuyCondition.addEventListener("click", () => {
    resetConditionRows(elements.buyConditions, [{ field: "score", op: "gte", value: 60 }]);
    updateStrategyPreview();
  });
}

if (elements.addSellCondition) {
  elements.addSellCondition.addEventListener("click", () => {
    resetConditionRows(elements.sellConditions, [{ field: "score", op: "lte", value: 40 }]);
    updateStrategyPreview();
  });
}

[
  "createStrategyName",
  "createStrategySymbol",
  "createStrategyTimeframe",
  "createStrategyEnv",
  "buyLogic",
  "sellLogic",
  "orderSizeMode",
  "orderSizeValue",
  "priceMode",
  "feesPct",
  "slippagePct",
  "stopLossPct",
  "takeProfitPct",
  "coolDownDays",
  "minHoldDays",
  "maxPositions",
].forEach((key) => {
  const el = elements[key];
  if (el) el.addEventListener("input", updateStrategyPreview);
});

if (elements.createStrategyForm && elements.loadStrategyTemplate) {
  loadStrategyTemplate();
}

window.addEventListener("resize", () => {
  if (state.lastChart && state.lastChart.items && state.lastChart.items.length) {
    renderHistoryChart(state.lastChart, state.lastBacktest?.events || []);
  }
});

function updateOptionalFields() {
  renderBacktestConditions();
  refreshConditionOptions();
}

let autoBacktestTimer = null;
function scheduleAutoBacktest() {
  clearTimeout(autoBacktestTimer);
  autoBacktestTimer = setTimeout(() => {
    runBacktest({ auto: true });
  }, 600);
}

const backtestInputs = [
  "btTotalMin",
  "btChangeMin",
  "btVolMin",
  "btReturnMin",
  "btMaGap",
  "btChangeWeight",
  "btVolWeight",
  "btReturnWeight",
  "btMaWeight",
];

backtestInputs.forEach((id) => {
  const el = document.getElementById(id);
  if (el) {
    el.addEventListener("input", scheduleAutoBacktest);
    el.addEventListener("change", scheduleAutoBacktest);
  }
});

if (elements.btSelectedConditions) {
  elements.btSelectedConditions.addEventListener("input", scheduleAutoBacktest);
}

async function runBacktest({ auto = false } = {}) {
  try {
    requireLogin();
  } catch (err) {
    if (auto) return;
    throw err;
  }
  const start_date = elements.chartStart?.value;
  const end_date = elements.chartEnd?.value;
  if (!start_date || !end_date) {
    if (!auto) renderChartPlaceholder("請設定回測日期區間");
    return;
  }
  if (backtestSelections.conditions.length === 0) {
    if (!auto) renderChartPlaceholder("請先選擇至少一個條件");
    return;
  }
  try {
    if (!auto) {
      renderChartLoading();
    }
    const payload = buildBacktestPayload();
    const res = await api("/api/analysis/backtest", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    const weighted = applyWeightedScoring(res);
    state.lastBacktest = weighted;
    renderBacktestSummary(weighted);
    renderBacktestEvents(weighted);
    if (state.lastChart && state.lastChart.items) {
      renderHistoryChart(state.lastChart, weighted.events || []);
    }
    if (!auto) {
      logActivity("條件回測", `命中 ${fmtInt(weighted.total_events)} 筆`);
    }
  } catch (err) {
    if (!auto) {
      renderChartPlaceholder(err.message);
    }
  }
}

function applyBacktestConfig(cfg) {
  if (!cfg) return;
  document.getElementById("btStart").value = cfg.start_date || document.getElementById("btStart").value;
  document.getElementById("btEnd").value = cfg.end_date || document.getElementById("btEnd").value;
  if (cfg.weights) {
    document.getElementById("btChangeWeight").value = cfg.weights.change_weight ?? 1;
    document.getElementById("btVolWeight").value = cfg.weights.volume_weight ?? 1;
    document.getElementById("btReturnWeight").value = cfg.weights.return_weight ?? 1;
    document.getElementById("btMaWeight").value = cfg.weights.ma_weight ?? 1;
  }
  if (cfg.thresholds) {
    document.getElementById("btTotalMin").value = cfg.thresholds.total_min ?? 1;
    document.getElementById("btChangeMin").value = (cfg.thresholds.change_min || 0) * 100;
    document.getElementById("btVolMin").value = cfg.thresholds.volume_ratio_min || 0;
    document.getElementById("btReturnMin").value = (cfg.thresholds.return5_min || 0) * 100;
    document.getElementById("btMaGap").value = (cfg.thresholds.ma_gap_min || 0) * 100;
  }
  let nextConds = [];
  if (cfg.conditions && cfg.conditions.length) {
    nextConds = [...cfg.conditions];
  } else if (cfg.flags) {
    if (cfg.flags.use_change) nextConds.push("change");
    if (cfg.flags.use_volume) nextConds.push("volume");
    if (cfg.flags.use_return) nextConds.push("return");
    if (cfg.flags.use_ma) nextConds.push("ma");
  }
  if (!nextConds.length) nextConds.push("change", "volume");
  backtestSelections.conditions = nextConds;
  renderBacktestConditions();
  refreshConditionOptions();
  scheduleAutoBacktest();
}

loadCombos();
