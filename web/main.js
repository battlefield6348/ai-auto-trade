const state = {
  token: "",
  health: null,
  lastIngestion: null,
  lastAnalysis: null,
  lastSummary: null,
  lastQuery: null,
  lastScreener: null,
  lastBackfill: null,
  lastChart: null,
  lastBacktestCriteria: null,
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
};

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
  must: {
    change: false,
    volume: false,
    return: false,
    ma: false,
  },
};

const readMustFlags = () => {
  const next = { ...backtestSelections.must };
  const idMap = {
    change: "btChangeMust",
    volume: "btVolMust",
    return: "btReturnMust",
    ma: "btMaMust",
  };
  Object.entries(idMap).forEach(([key, id]) => {
    const el = document.getElementById(id);
    next[key] = !!el?.checked;
  });
  backtestSelections.must = next;
  return next;
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
  const chips = Object.entries(returns)
    .map(([key, val]) => {
      const avg = fmtPercent(val.avg_return);
      const win = fmtPercent(val.win_rate);
      return `<span class="meta-item">${key} 平均 ${avg} ｜ 勝率 ${win}</span>`;
    })
    .join("");
  elements.backtestSummary.innerHTML = `
    <div class="result-header">
      <div>
        <div class="result-title">回測結果</div>
        <div class="result-sub">${res.start_date} ~ ${res.end_date}</div>
      </div>
      <span class="badge ${res.total_events ? "good" : "warn"}">命中 ${fmtInt(res.total_events)}</span>
    </div>
    <div class="meta-row">${chips || '<div class="meta-item">尚無統計</div>'}</div>
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

const applyBacktestMustFilter = (res) => {
  if (!res || !res.events || !res.events.length) return res;
  const cfg = state.lastBacktestCriteria;
  if (!cfg || !cfg.must) return res;
  const { must, thresholds = {} } = cfg;
  const activeMust = Object.entries(must).filter(([, v]) => v);
  if (!activeMust.length) return res;
  const hit = (row, cond) => {
    switch (cond) {
      case "change":
        return row.change_percent >= (thresholds.change_min || 0);
      case "volume":
        return row.volume_ratio >= (thresholds.volume_ratio_min || 0);
      case "return":
        return (row.return_5d || 0) >= (thresholds.return5_min || 0);
      case "ma":
        return (row.ma_gap || 0) >= (thresholds.ma_gap_min || 0);
      default:
        return true;
    }
  };
  const filteredEvents = res.events.filter((row) => activeMust.every(([cond]) => hit(row, cond)));
  return { ...res, events: filteredEvents, total_events: filteredEvents.length };
};

const buildBacktestPayload = () => {
  const mustFlags = readMustFlags();
  const start_date = document.getElementById("btStart").value;
  const end_date = document.getElementById("btEnd").value;
  const config = {
    symbol: "BTCUSDT",
    start_date,
    end_date,
    weights: {
      score: Number(document.getElementById("btScoreWeight").value || 1),
      change_bonus: backtestSelections.conditions.includes("change")
        ? Number(document.getElementById("btChangeBonus").value || 0)
        : 0,
      volume_bonus: backtestSelections.conditions.includes("volume")
        ? Number(document.getElementById("btVolBonus").value || 0)
        : 0,
      return_bonus: backtestSelections.conditions.includes("return")
        ? Number(document.getElementById("btReturnBonus").value || 0)
        : 0,
      ma_bonus: backtestSelections.conditions.includes("ma")
        ? Number(document.getElementById("btMaBonus").value || 0)
        : 0,
    },
    thresholds: {
      total_min: Number(document.getElementById("btTotalMin").value || 0),
      change_min: backtestSelections.conditions.includes("change")
        ? Number(document.getElementById("btChangeMin").value || 0) / 100
        : 0,
      volume_ratio_min: backtestSelections.conditions.includes("volume")
        ? Number(document.getElementById("btVolMin").value || 0)
        : 0,
      return5_min: backtestSelections.conditions.includes("return")
        ? Number(document.getElementById("btReturnMin").value || 0) / 100
        : 0,
      ma_gap_min: backtestSelections.conditions.includes("ma")
        ? Number(document.getElementById("btMaGap").value || 0) / 100
        : 0,
    },
    flags: {
      use_change: backtestSelections.conditions.includes("change"),
      use_volume: backtestSelections.conditions.includes("volume"),
      use_return: backtestSelections.conditions.includes("return"),
      use_ma: backtestSelections.conditions.includes("ma"),
    },
    horizons: [3, 5, 10],
    must: mustFlags,
  };
  state.lastBacktestCriteria = config;
  return {
    symbol: config.symbol,
    start_date: config.start_date,
    end_date: config.end_date,
    weights: config.weights,
    thresholds: config.thresholds,
    flags: config.flags,
    horizons: config.horizons,
  };
};

const renderBacktestConditions = () => {
  if (!elements.btSelectedConditions) return;
  if (!backtestSelections.conditions.length) {
    renderEmptyState(elements.btSelectedConditions, "尚未選擇條件");
    return;
  }
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
              <label>日漲跌加分 <input type="number" step="1" id="btChangeBonus" value="10"></label>
              <label>漲幅門檻(%) <input type="number" step="0.1" id="btChangeMin" value="0.5"></label>
              <label class="inline-check"><input type="checkbox" id="btChangeMust" ${
                backtestSelections.must.change ? "checked" : ""
              }> 必須命中</label>
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
              <label>量能加分 <input type="number" step="1" id="btVolBonus" value="10"></label>
              <label>量能門檻(倍率) <input type="number" step="0.1" id="btVolMin" value="1.2"></label>
              <label class="inline-check"><input type="checkbox" id="btVolMust" ${
                backtestSelections.must.volume ? "checked" : ""
              }> 必須命中</label>
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
              <label>報酬加分 <input type="number" step="1" id="btReturnBonus" value="8"></label>
              <label>報酬門檻(%) <input type="number" step="0.1" id="btReturnMin" value="1.0"></label>
              <label class="inline-check"><input type="checkbox" id="btReturnMust" ${
                backtestSelections.must.return ? "checked" : ""
              }> 必須命中</label>
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
              <label>均線加分 <input type="number" step="1" id="btMaBonus" value="5"></label>
              <label>乖離門檻(%) <input type="number" step="0.1" id="btMaGap" value="1.0"></label>
              <label class="inline-check"><input type="checkbox" id="btMaMust" ${
                backtestSelections.must.ma ? "checked" : ""
              }> 必須命中</label>
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
      delete backtestSelections.must[cond];
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
renderActivity();
renderKpis();
updateOverviewMode();
renderBacktestConditions();
refreshConditionOptions();
toggleProtectedSections(false);
refreshAccessToken().then((ok) => {
  if (ok) {
    setMessage(elements.loginMessage, "已自動登入，Token 已更新", "good");
    logActivity("自動登入", "沿用前一次的登入狀態");
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
  }
  const nextConds = [];
  if (c.flags?.use_change) nextConds.push("change");
  if (c.flags?.use_volume) nextConds.push("volume");
  if (c.flags?.use_return) nextConds.push("return");
  if (c.flags?.use_ma) nextConds.push("ma");
  if (!nextConds.length) nextConds.push("change", "volume");
  backtestSelections.conditions = nextConds;
  backtestSelections.must = {
    change: false,
    volume: false,
    return: false,
    ma: false,
  };
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
    logActivity("登入成功", `帳號 ${email}`);
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
      const filtered = applyBacktestMustFilter(res);
      state.lastBacktest = filtered;
      renderBacktestSummary(filtered);
      renderBacktestEvents(filtered);
      if (state.lastChart && state.lastChart.items) {
        renderHistoryChart(state.lastChart, filtered.events || []);
      }
      logActivity("條件回測", `命中 ${fmtInt(filtered.total_events)} 筆`);
    } catch (err) {
      renderChartPlaceholder(err.message);
    }
  });
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

window.addEventListener("resize", () => {
  if (state.lastChart && state.lastChart.items && state.lastChart.items.length) {
    renderHistoryChart(state.lastChart, state.lastBacktest?.events || []);
  }
});

function updateOptionalFields() {
  renderBacktestConditions();
  refreshConditionOptions();
}
