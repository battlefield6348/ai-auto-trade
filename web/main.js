const state = {
  token: "",
  currentSection: "overview",
  health: null,
  lastIngestion: null,
  lastAnalysis: null,
  lastSummary: null,
  lastQuery: null,
  lastScreener: null,
  lastBackfill: null,
  lastChart: null,
  strategies: [],
  activity: [],
  updatedAt: null,
};

const elements = {
  status: document.getElementById("status"),
  healthStatus: document.getElementById("healthStatus"),
  overviewTime: document.getElementById("overviewTime"),
  overviewMode: document.getElementById("overviewMode"),
  loginMessage: document.getElementById("loginMessage"),
  comboName: document.getElementById("comboName"),
  comboSelect: document.getElementById("comboSelect"),
  comboSaveBtn: document.getElementById("comboSaveBtn"),
  comboApplyBtn: document.getElementById("comboApplyBtn"),
  comboDeleteBtn: document.getElementById("comboDeleteBtn"),
  resetZoomBtn: document.getElementById("resetZoomBtn"),
  chartForm: document.getElementById("chartForm"),
  chartStart: document.getElementById("chartStart"),
  chartEnd: document.getElementById("chartEnd"),
  chartMeta: document.getElementById("chartMeta"),
  chartCanvas: document.getElementById("chartCanvas"),
  chartTooltip: document.getElementById("chartTooltip"),
  chartHighScores: document.getElementById("chartHighScores"),
  backtestForm: document.getElementById("backtestForm"),
  backtestSummary: document.getElementById("backtestSummary"),
  backtestEvents: document.getElementById("backtestEvents"),
  btConditionSelect: document.getElementById("btConditionSelect"),
  btSelectedConditions: document.getElementById("btSelectedConditions"),
  summaryView: document.getElementById("summaryView"),
  queryMeta: document.getElementById("queryMeta"),
  queryHighlights: document.getElementById("queryHighlights"),
  queryTable: document.getElementById("queryTable"),
  screenerMeta: document.getElementById("screenerMeta"),
  screenerHighlights: document.getElementById("screenerHighlights"),
  screenerTable: document.getElementById("screenerTable"),
  activityList: document.getElementById("activityList"),
  strategyForm: document.getElementById("strategyForm"),
  strategyStatus: document.getElementById("strategyStatus"),
  strategyEnv: document.getElementById("strategyEnv"),
  strategyName: document.getElementById("strategyName"),
  strategyTable: document.getElementById("strategyTable"),
  strategyMeta: document.getElementById("strategyMeta"),
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
  if (elements.chartHighScores) renderEmptyState(elements.chartHighScores, "尚無高分點（Score ≥ 50）");
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
  if (elements.chartHighScores) {
    const highRows = rows.filter((row) => typeof row.score === "number" && row.score >= 50);
    renderHighScoreList(elements.chartHighScores, highRows);
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
    const eventDates = new Set(backtestEvents.map((e) => e.trade_date));
    const markers = points
      .filter((pt) => eventDates.has(pt.row.trade_date))
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

const renderHighScoreList = (container, rows) => {
  if (!container) return;
  if (!rows.length) {
    renderEmptyState(container, "尚無高分點（Score ≥ 50）");
    return;
  }
  const list = rows
    .sort((a, b) => b.score - a.score)
    .slice(0, 8)
    .map(
      (row) => `
        <article class="highlight-card">
          <h4>${row.trade_date}</h4>
          <div class="highlight-item"><span>收盤</span><span>${fmtPrice(row.close_price)}</span></div>
          <div class="highlight-item"><span>日漲跌</span><span class="delta ${deltaClass(
            row.change_percent
          )}">${fmtPercent(row.change_percent)}</span></div>
          <div class="highlight-item"><span>近 5 日</span><span class="delta ${deltaClass(
            row.return_5d
          )}">${fmtPercent(row.return_5d)}</span></div>
          <div class="highlight-item"><span>量能倍率</span><span>${fmtRatio(
            row.volume_ratio
          )}</span></div>
          <div class="highlight-item"><span>Score</span><span>${fmtScore(row.score)}</span></div>
        </article>
      `
    )
    .join("");
  container.innerHTML = list;
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
  score: numInput("btScoreWeight", 1),
  change_bonus: backtestSelections.conditions.includes("change") ? numInput("btChangeBonus", 0) : 0,
  change_weight: backtestSelections.conditions.includes("change") ? numInput("btChangeWeight", 1) : 0,
  volume_bonus: backtestSelections.conditions.includes("volume") ? numInput("btVolBonus", 0) : 0,
  volume_weight: backtestSelections.conditions.includes("volume") ? numInput("btVolWeight", 1) : 0,
  return_bonus: backtestSelections.conditions.includes("return") ? numInput("btReturnBonus", 0) : 0,
  return_weight: backtestSelections.conditions.includes("return") ? numInput("btReturnWeight", 1) : 0,
  ma_bonus: backtestSelections.conditions.includes("ma") ? numInput("btMaBonus", 0) : 0,
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
    const baseScore = (row.score || 0) * (weights.score || 0);
    comp.base = baseScore;
    total += baseScore;
    if (flags.change && row.change_percent >= thresholds.change_min) {
      const v = weights.change_bonus * (weights.change_weight || 1);
      comp.change = v;
      total += v;
    }
    if (flags.volume && row.volume_ratio >= thresholds.volume_ratio_min) {
      const v = weights.volume_bonus * (weights.volume_weight || 1);
      comp.volume = v;
      total += v;
    }
    if (flags.return && (row.return_5d || 0) >= thresholds.return5_min) {
      const v = weights.return_bonus * (weights.return_weight || 1);
      comp.return = v;
      total += v;
    }
    if (flags.ma && (row.ma_gap || 0) >= thresholds.ma_gap_min) {
      const v = weights.ma_bonus * (weights.ma_weight || 1);
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
  const start_date = document.getElementById("btStart").value;
  const end_date = document.getElementById("btEnd").value;
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
              <label>日漲跌加分 <input type="number" step="1" id="btChangeBonus" value="${current(
                "btChangeBonus",
                10
              )}"></label>
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
              <label>量能加分 <input type="number" step="1" id="btVolBonus" value="${current(
                "btVolBonus",
                10
              )}"></label>
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
              <label>報酬加分 <input type="number" step="1" id="btReturnBonus" value="${current(
                "btReturnBonus",
                8
              )}"></label>
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
              <label>均線加分 <input type="number" step="1" id="btMaBonus" value="${current(
                "btMaBonus",
                5
              )}"></label>
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

const columns = [
  { key: "trading_pair", label: "交易對", className: "mono" },
  { key: "close_price", label: "收盤", format: fmtPrice },
  { key: "change_percent", label: "日漲跌", format: fmtPercent, delta: true },
  { key: "return_5d", label: "近 5 日", format: fmtPercent, delta: true },
  { key: "volume", label: "成交量", format: fmtInt },
  { key: "volume_ratio", label: "量能倍率", format: fmtRatio },
  { key: "score", label: "Score", format: fmtScore },
  { key: "market_type", label: "市場", format: fmtText },
];

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
renderEmptyState(elements.queryTable, "尚未查詢");
renderEmptyState(elements.screenerTable, "尚未查詢");
renderEmptyState(elements.queryHighlights, "尚無亮點資料");
renderEmptyState(elements.screenerHighlights, "尚無亮點資料");
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
["queryDate", "screenerDate", "chartEnd"].forEach((id) => {
  const el = document.getElementById(id);
  if (el) el.value = today;
});
if (elements.chartStart) elements.chartStart.value = startOfYear;
const btStart = document.getElementById("btStart");
const btEnd = document.getElementById("btEnd");
if (btStart) btStart.value = startOfYear;
if (btEnd) btEnd.value = today;
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
  document.getElementById("btStart").value = c.start_date || document.getElementById("btStart").value;
  document.getElementById("btEnd").value = c.end_date || document.getElementById("btEnd").value;
  document.getElementById("btScoreWeight").value = c.weights?.score ?? 1;
  document.getElementById("btTotalMin").value = c.thresholds?.total_min ?? 60;
  if (c.thresholds) {
    document.getElementById("btChangeMin").value = (c.thresholds.change_min || 0) * 100;
    document.getElementById("btVolMin").value = c.thresholds.volume_ratio_min || 0;
    document.getElementById("btReturnMin").value = (c.thresholds.return5_min || 0) * 100;
    document.getElementById("btMaGap").value = (c.thresholds.ma_gap_min || 0) * 100;
  }
  if (c.weights) {
    document.getElementById("btChangeBonus").value = c.weights.change_bonus || 0;
    document.getElementById("btVolBonus").value = c.weights.volume_bonus || 0;
    document.getElementById("btReturnBonus").value = c.weights.return_bonus || 0;
    document.getElementById("btMaBonus").value = c.weights.ma_bonus || 0;
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

const fmtDate = (v) => {
  if (!v) return "—";
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return fmtText(v);
  return d.toISOString().slice(0, 10);
};

const setStrategyBacktestDefaults = () => {
  const today = new Date();
  const start = new Date();
  start.setDate(today.getDate() - 180);
  if (elements.strategyBtStart) elements.strategyBtStart.value = start.toISOString().slice(0, 10);
  if (elements.strategyBtEnd) elements.strategyBtEnd.value = today.toISOString().slice(0, 10);
};

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

const renderStrategyBacktestSummary = (rec) => {
  if (!elements.strategyBacktestSummary) return;
  if (!rec) {
    renderEmptyState(elements.strategyBacktestSummary, "尚未執行回測");
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
    logActivity("策略回測", `策略 ${res.strategyID} · ${res.payload.start_date}~${res.payload.end_date}`);
  } catch (err) {
    setMessage(elements.strategyBacktestMessage, `回測失敗：${err.message}`, "error");
    renderStrategyBacktestSummary(null);
  }
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
  if (!items.length) {
    renderEmptyState(elements.reportTable, "尚未查詢");
    if (elements.reportMeta) elements.reportMeta.innerHTML = "";
    return;
  }
  const rows = items.map((r) => ({
    id: fmtText(r.id),
    env: fmtText(r.env),
    period: `${fmtText(r.period_start)} ~ ${fmtText(r.period_end)}`,
    summary: r.summary ? JSON.stringify(r.summary) : "—",
    created_at: r.created_at ? timeFormat.format(new Date(r.created_at)) : "—",
  }));
  const cols = [
    { key: "id", label: "報告 ID", className: "mono" },
    { key: "env", label: "環境" },
    { key: "period", label: "期間" },
    { key: "summary", label: "摘要" },
    { key: "created_at", label: "建立時間" },
  ];
  renderTable(elements.reportTable, rows, cols);
  if (elements.reportMeta) {
    elements.reportMeta.innerHTML = `<div class="meta-item">共 ${fmtInt(items.length)} 筆報告</div>`;
  }
};

const loadReports = async () => {
  if (!elements.reportForm) return;
  const strategyId = (elements.reportStrategyId?.value || "").trim();
  if (!strategyId) {
    setStatus("請輸入策略 ID", "warn");
    renderEmptyState(elements.reportTable, "尚未查詢");
    return;
  }
  renderEmptyState(elements.reportTable, "載入報告中...");
  const res = await api(`/api/admin/strategies/${strategyId}/reports`);
  renderReportTable(res.reports || []);
  logActivity("查詢報告", `策略 ${strategyId} · 筆數 ${fmtInt((res.reports || []).length)}`);
  setStatus("報告列表已更新", "good");
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
    try {
      requireLogin();
      const start_date = elements.chartStart.value;
      const end_date = elements.chartEnd.value;
      if (!start_date || !end_date) {
        renderChartPlaceholder("請設定起始與結束日期");
        return;
      }
      renderChartLoading();
      const res = await api(
        `/api/analysis/history?symbol=BTCUSDT&start_date=${start_date}&end_date=${end_date}&only_success=true`
      );
      state.lastChart = res;
      renderHistoryChart(res, state.lastBacktest?.events || []);
      logActivity("載入走勢圖", `區間 ${start_date} ~ ${end_date} · 筆數 ${fmtInt(res.total_count)}`);
    } catch (err) {
      renderChartPlaceholder(err.message);
    }
  });
}

if (elements.resetZoomBtn) {
  elements.resetZoomBtn.addEventListener("click", resetZoom);
}

const comboStore = {
  list: [],
};

const renderComboList = () => {
  if (!elements.comboSelect) return;
  const options =
    comboStore.list.length === 0
      ? `<option value="">尚未儲存組合</option>`
      : `<option value="">選擇組合套用…</option>` +
        comboStore.list
          .map((c) => `<option value="${c.id}">${c.name || "未命名組合"}</option>`)
          .join("");
  elements.comboSelect.innerHTML = options;
};

const fetchCombos = async () => {
  if (!state.token) return;
  try {
    const res = await api("/api/analysis/backtest/presets");
    if (res.success) {
      comboStore.list = res.items || [];
      renderComboList();
    }
  } catch (err) {
    console.warn("fetch combos failed", err);
  }
};

const saveCombo = async () => {
  try {
    requireLogin();
    const name = (elements.comboName?.value || "").trim() || "未命名組合";
    const payload = buildBacktestPayload();
    await api("/api/analysis/backtest/presets", {
      method: "POST",
      body: JSON.stringify({ name, config: payload }),
    });
    await fetchCombos();
    setStatus(`已儲存組合：${name}`, "good");
  } catch (err) {
    setStatus(`儲存組合失敗：${err.message}`, "warn");
  }
};

const applyCombo = () => {
  const id = elements.comboSelect?.value;
  if (!id) return;
  const combo = comboStore.list.find((c) => c.id === id);
  if (!combo) return;
  applyBacktestConfig(combo.config || combo.payload);
  setStatus(`已套用組合：${combo.name}`, "good");
};

const deleteCombo = async () => {
  try {
    requireLogin();
    const id = elements.comboSelect?.value;
    if (!id) return;
    await api(`/api/analysis/backtest/presets/${id}`, { method: "DELETE" });
    await fetchCombos();
    setStatus("組合已刪除", "warn");
  } catch (err) {
    setStatus(`刪除組合失敗：${err.message}`, "warn");
  }
};

if (elements.backtestForm) {
  elements.backtestForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      requireLogin();
      const start_date = document.getElementById("btStart").value;
      const end_date = document.getElementById("btEnd").value;
      if (!start_date || !end_date) {
        renderChartPlaceholder("請設定回測日期區間");
        return;
      }
  if (backtestSelections.conditions.length === 0) {
    renderChartPlaceholder("請先選擇至少一個條件");
    return;
  }
  const payload = buildBacktestPayload();
  renderChartLoading();
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
      logActivity("條件回測", `命中 ${fmtInt(weighted.total_events)} 筆`);
    } catch (err) {
      renderChartPlaceholder(err.message);
    }
  });
}

if (elements.comboSaveBtn) {
  elements.comboSaveBtn.addEventListener("click", saveCombo);
}

if (elements.comboApplyBtn) {
  elements.comboApplyBtn.addEventListener("click", applyCombo);
}

if (elements.comboDeleteBtn) {
  elements.comboDeleteBtn.addEventListener("click", deleteCombo);
}

document.getElementById("queryForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  try {
    requireLogin();
    const trade_date = document.getElementById("queryDate").value;
    const limit = document.getElementById("queryLimit").value || 20;
    const offset = document.getElementById("queryOffset").value || 0;
    const res = await api(
      `/api/analysis/daily?trade_date=${trade_date}&limit=${limit}&offset=${offset}`
    );
    state.lastQuery = res;
    renderMeta(elements.queryMeta, [
      `交易日：${fmtText(res.trade_date)}`,
      `總筆數：${fmtInt(res.total_count)}`,
      `顯示：${fmtInt(res.items?.length || 0)}`,
    ]);
    renderHighlights(elements.queryHighlights, res.items || []);
    renderTable(elements.queryTable, res.items || [], columns);
    logActivity("查詢分析結果", `交易日 ${trade_date} · 筆數 ${fmtInt(res.items?.length || 0)}`);
  } catch (err) {
    renderMeta(elements.queryMeta, ["查詢失敗"]);
    renderEmptyState(elements.queryHighlights, err.message);
    renderEmptyState(elements.queryTable, err.message);
  }
});

document.getElementById("screenerForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  try {
    requireLogin();
    const trade_date = document.getElementById("screenerDate").value;
    const score_min = document.getElementById("scoreMin").value || 70;
    const volume_ratio_min = document.getElementById("volMin").value || 1.5;
    const limit = document.getElementById("screenerLimit").value || 20;
    const res = await api(
      `/api/screener/strong-stocks?trade_date=${trade_date}&score_min=${score_min}&volume_ratio_min=${volume_ratio_min}&limit=${limit}`
    );
    state.lastScreener = res;
    renderMeta(elements.screenerMeta, [
      `交易日：${fmtText(res.trade_date)}`,
      `條件：Score ≥ ${fmtNumber(res.params?.score_min)} · 量能 ≥ ${fmtNumber(
        res.params?.volume_ratio_min
      )}`,
      `筆數：${fmtInt(res.total_count)}`,
    ]);
    renderHighlights(elements.screenerHighlights, res.items || []);
    renderTable(elements.screenerTable, res.items || [], columns);
    renderKpis();
    logActivity("查詢強勢交易對", `交易日 ${trade_date} · 筆數 ${fmtInt(res.items?.length || 0)}`);
  } catch (err) {
    renderMeta(elements.screenerMeta, ["查詢失敗"]);
    renderEmptyState(elements.screenerHighlights, err.message);
    renderEmptyState(elements.screenerTable, err.message);
  }
});

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

if (elements.addBuyCondition) {
  elements.addBuyCondition.addEventListener("click", () => {
    addConditionRow(elements.buyConditions, { field: "score", op: "gte", value: 60 });
    updateStrategyPreview();
  });
}

if (elements.addSellCondition) {
  elements.addSellCondition.addEventListener("click", () => {
    addConditionRow(elements.sellConditions, { field: "score", op: "lte", value: 40 });
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

function applyBacktestConfig(cfg) {
  if (!cfg) return;
  document.getElementById("btStart").value = cfg.start_date || document.getElementById("btStart").value;
  document.getElementById("btEnd").value = cfg.end_date || document.getElementById("btEnd").value;
  if (cfg.weights) {
    document.getElementById("btScoreWeight").value = cfg.weights.score ?? 1;
    document.getElementById("btChangeBonus").value = cfg.weights.change_bonus ?? 0;
    document.getElementById("btVolBonus").value = cfg.weights.volume_bonus ?? 0;
    document.getElementById("btReturnBonus").value = cfg.weights.return_bonus ?? 0;
    document.getElementById("btMaBonus").value = cfg.weights.ma_bonus ?? 0;
    document.getElementById("btChangeWeight").value = cfg.weights.change_weight ?? 1;
    document.getElementById("btVolWeight").value = cfg.weights.volume_weight ?? 1;
    document.getElementById("btReturnWeight").value = cfg.weights.return_weight ?? 1;
    document.getElementById("btMaWeight").value = cfg.weights.ma_weight ?? 1;
  }
  if (cfg.thresholds) {
    document.getElementById("btTotalMin").value = cfg.thresholds.total_min ?? 60;
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
}

loadCombos();
