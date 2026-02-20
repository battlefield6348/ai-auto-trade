import { updateExchangeLink, initSidebar, initBinanceConfigModal, initAuthModal, initGlobalEnvSelector, handleUnauthorized } from "./common.js";

const state = {
  token: localStorage.getItem("aat_token") || "",
  lastEmail: localStorage.getItem("aat_email") || "",
  role: "",
  btScoreChart: null,
  btReturnChart: null,
  debounceTimer: null,
};

const el = (id) => document.getElementById(id);

function debounce(fn, delay = 500) {
  return (...args) => {
    clearTimeout(state.debounceTimer);
    state.debounceTimer = setTimeout(() => fn(...args), delay);
  };
}

function setAlert(msg, type = "info") {
  const box = el("alert");
  if (!msg) {
    box.classList.add("hidden");
    box.textContent = "";
    return;
  }
  const palette = {
    info: "border-primary/30 bg-primary/10 text-primary",
    error: "border-danger/30 bg-danger/10 text-danger",
    success: "border-success/30 bg-success/10 text-success",
  };
  box.className = `rounded border px-4 py-3 text-sm ${palette[type] || palette.info}`;
  box.textContent = msg;
  box.classList.remove("hidden");
  // Scroll to top to ensure user sees the alert
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function authHeaders(requireAuth = true) {
  const headers = {};
  if (requireAuth && state.token) headers["Authorization"] = `Bearer ${state.token}`;
  return headers;
}

async function api(path, { method = "GET", body, requireAuth = true } = {}) {
  const headers = authHeaders(requireAuth);
  let payload = body;
  if (body && typeof body === "object" && !(body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  console.log(`[API] ${method} ${path}`, body || "");
  const res = await fetch(path, { method, headers, body: payload });
  if (res.status === 401) {
    handleUnauthorized();
    throw new Error("需要登入或權限不足");
  }
  const data = await res.json().catch(() => ({}));
  console.log(`[API] Response ${path}:`, data);
  if (!res.ok || data.success === false) {
    const msg = data?.message || data?.error || data?.reason || res.statusText;
    throw new Error(msg);
  }
  if (res.headers.get("x-user-role")) {
    state.role = res.headers.get("x-user-role");
    el("loginStatus").textContent = `已登入：${state.lastEmail || state.role}`;
  }
  return data;
}

function readForm() {
  const parse = (id, def = 0) => {
    const v = parseFloat(el(id).value);
    return isNaN(v) ? def : v;
  };
  const horizons = el("btHorizons").value
    .split(",")
    .map((s) => parseInt(s.trim(), 10))
    .filter((n) => !isNaN(n));

  const buildSide = (prefix) => ({
    weights: {
      score: parse(prefix + "ScoreWeight", 1.0),
      change_bonus: parse(prefix + "ChangeBonus", 10),
      volume_bonus: parse(prefix + "VolumeBonus", 10),
      ma_bonus: parse(prefix + "MaBonus", 5),
      range_bonus: parse(prefix + "RangeBonus", 5),
      return_bonus: 0,
      amp_bonus: 0
    },
    thresholds: {
      change_min: parse(prefix + "ChangeMin", 0.5) / 100,
      volume_ratio_min: parse(prefix + "VolMin", 1.2),
      ma_gap_min: parse(prefix + "MaGapMin", 1) / 100,
      range_min: parse(prefix + "RangeMin", 80),
      return5_min: 0,
      amp_min: 0
    },
    flags: {
      use_change: el(prefix + "UseChange").checked,
      use_volume: el(prefix + "UseVolume").checked,
      use_ma: el(prefix + "UseMa").checked,
      use_range: el(prefix + "UseRange").checked,
      use_return: false,
      use_amp: false
    },
    total_min: parse(prefix + "TotalMin", 60)
  });

  return {
    symbol: (el("btSymbol").value || "BTCUSDT").trim().toUpperCase(),
    start_date: el("btStart").value,
    end_date: el("btEnd").value,
    entry: buildSide("btEntry"),
    exit: buildSide("btExit"),
    horizons: horizons.length ? horizons : [3, 5, 10],
    timeframe: "1d"
  };
}

function updateVisibility() {
  // Simple check based on prefix
  ["btEntry", "btExit"].forEach(prefix => {
    ["UseChange", "UseVolume", "UseMa", "UseRange"].forEach(name => {
      const checkbox = el(prefix + name);
      if (checkbox) {
        // Toggle opacity or related inputs if needed, though they are inline now
        // For now just refresh max score
      }
    });
  });
  updateMaxScore();
}

function updateMaxScore() {
  const parse = (id) => parseFloat(el(id)?.value) || 0;

  let totalEntry = parse("btEntryScoreWeight");
  if (el("btEntryUseChange").checked) totalEntry += parse("btEntryChangeBonus");
  if (el("btEntryUseVolume").checked) totalEntry += parse("btEntryVolumeBonus");
  if (el("btEntryUseMa").checked) totalEntry += parse("btEntryMaBonus");
  if (el("btEntryUseRange")?.checked) totalEntry += parse("btEntryRangeBonus");

  const display = el("maxPossibleScore");
  if (display) display.textContent = totalEntry.toFixed(1);
}

function fillForm(cfg) {
  if (!cfg) return;
  if (cfg.symbol) el("btSymbol").value = cfg.symbol;
  if (cfg.start_date) el("btStart").value = cfg.start_date;
  if (cfg.end_date) el("btEnd").value = cfg.end_date;

  const mapSide = (prefix, sideCfg) => {
    if (!sideCfg) return;
    if (sideCfg.weights) {
      if (sideCfg.weights.score !== undefined) el(prefix + "ScoreWeight").value = sideCfg.weights.score;
      if (sideCfg.weights.change_bonus !== undefined) el(prefix + "ChangeBonus").value = sideCfg.weights.change_bonus;
      if (sideCfg.weights.volume_bonus !== undefined) el(prefix + "VolumeBonus").value = sideCfg.weights.volume_bonus;
      if (sideCfg.weights.ma_bonus !== undefined) el(prefix + "MaBonus").value = sideCfg.weights.ma_bonus;
      if (sideCfg.weights.range_bonus !== undefined) el(prefix + "RangeBonus").value = sideCfg.weights.range_bonus;
    }
    if (sideCfg.thresholds) {
      if (sideCfg.thresholds.change_min !== undefined) el(prefix + "ChangeMin").value = sideCfg.thresholds.change_min * 100;
      if (sideCfg.thresholds.volume_ratio_min !== undefined) el(prefix + "VolMin").value = sideCfg.thresholds.volume_ratio_min;
      if (sideCfg.thresholds.ma_gap_min !== undefined) el(prefix + "MaGapMin").value = sideCfg.thresholds.ma_gap_min * 100;
      if (sideCfg.thresholds.range_min !== undefined) el(prefix + "RangeMin").value = sideCfg.thresholds.range_min;
    }
    if (sideCfg.flags) {
      if (sideCfg.flags.use_change !== undefined) el(prefix + "UseChange").checked = sideCfg.flags.use_change;
      if (sideCfg.flags.use_volume !== undefined) el(prefix + "UseVolume").checked = sideCfg.flags.use_volume;
      if (sideCfg.flags.use_ma !== undefined) el(prefix + "UseMa").checked = sideCfg.flags.use_ma;
      if (sideCfg.flags.use_range !== undefined) el(prefix + "UseRange").checked = sideCfg.flags.use_range;
    }
    if (sideCfg.total_min !== undefined) el(prefix + "TotalMin").value = sideCfg.total_min;
  };

  if (cfg.entry) mapSide("btEntry", cfg.entry);
  if (cfg.exit) mapSide("btExit", cfg.exit);

  // Transition support for legacy presets
  if (cfg.weights && !cfg.entry) {
    // Basic mapping for score and common flags
    el("btEntryScoreWeight").value = cfg.weights.score || 50;
    el("btEntryTotalMin").value = cfg.thresholds?.total_min || 60;
  }

  if (cfg.horizons) el("btHorizons").value = cfg.horizons.join(",");
  updateVisibility();
}


function renderResult(res) {
  const box = el("btResult");
  const list = el("btEvents");
  const resetCharts = () => {
    if (state.btScoreChart) state.btScoreChart.destroy();
    if (state.btReturnChart) state.btReturnChart.destroy();
    state.btScoreChart = null;
    state.btReturnChart = null;
  };
  if (list) list.innerHTML = "";
  if (!box) return;
  if (!res || res.error) {
    box.textContent = res?.error || "尚未回測";
    resetCharts();
    return;
  }
  const summary = res.summary || {};
  const stats = res.stats || {};
  const rows = [];
  rows.push(`<div class="text-xs text-slate-400">交易對：${res.symbol || ""}</div>`);
  rows.push(`<div>期間：${res.start_date || "--"} ~ ${res.end_date || "--"}</div>`);

  if (summary.total_trades !== undefined) {
    rows.push(`
      <div class="grid grid-cols-2 gap-4 mt-4 p-4 bg-primary/5 rounded-xl border border-primary/20">
        <div>
          <div class="text-[10px] text-slate-500 uppercase font-bold">總交易次數</div>
          <div class="text-xl font-black text-white">${summary.total_trades} 次</div>
        </div>
        <div>
          <div class="text-[10px] text-slate-500 uppercase font-bold">累積收益率</div>
          <div class="text-xl font-black ${summary.total_return >= 0 ? 'text-success' : 'text-danger'}">${summary.total_return.toFixed(2)}%</div>
        </div>
        <div>
          <div class="text-[10px] text-slate-500 uppercase font-bold">勝率</div>
          <div class="text-xl font-black text-white">${summary.win_rate?.toFixed(1) || 0}%</div>
        </div>
      </div>
    `);
  }

  if (stats.returns) {
    rows.push(`<div class="mt-4 text-[10px] text-slate-500 uppercase font-bold">統計預估：</div>`);
    const items = Object.entries(stats.returns).map(
      ([k, v]) => `<span class="text-xs text-slate-300">${k.replace('d', '')}日後: ${(v.avg_return * 100).toFixed(2)}% (勝率 ${(v.win_rate * 100).toFixed(1)}%)</span>`
    );
    rows.push(`<div class="flex flex-wrap gap-x-4 gap-y-1">${items.join("")}</div>`);
  }
  box.innerHTML = rows.join("");

  if (list) {
    list.innerHTML = "";
    const trades = res.trades || [];
    if (trades.length > 0) {
      // Show Simulation Trades
      const h3 = document.createElement("h3");
      h3.className = "text-[10px] text-slate-500 font-bold uppercase tracking-widest mb-4 mt-6";
      h3.textContent = "交易明細 (Trades)";
      list.appendChild(h3);

      trades.forEach((t) => {
        const div = document.createElement("div");
        div.className = "border border-surface-border rounded-lg p-3 bg-background-dark/50 hover:border-primary/50 transition-all";
        div.innerHTML = `
          <div class="flex justify-between items-start mb-2">
            <div class="flex flex-col">
              <span class="text-[10px] text-slate-500 font-bold uppercase">進場 (Entry)</span>
              <span class="text-xs text-white font-mono">${t.entry_date}</span>
            </div>
            <div class="flex flex-col items-end">
              <span class="text-[10px] text-slate-500 font-bold uppercase">出場 (Exit)</span>
              <span class="text-xs text-white font-mono">${t.exit_date}</span>
            </div>
          </div>
          <div class="flex justify-between items-center bg-black/20 p-2 rounded">
             <div class="text-xs font-mono text-slate-400">@ ${t.entry_price.toFixed(1)}</div>
             <div class="material-symbols-outlined text-xs text-slate-600">arrow_forward</div>
             <div class="text-xs font-mono text-slate-400">@ ${t.exit_price.toFixed(1)}</div>
          </div>
          <div class="flex justify-between items-center mt-2">
            <span class="text-[10px] text-slate-400">${t.reason || ""}</span>
            <span class="text-sm font-black ${t.pnl_pct >= 0 ? 'text-success' : 'text-error'}">
              ${t.pnl_pct >= 0 ? '+' : ''}${(t.pnl_pct * 100).toFixed(2)}%
            </span>
          </div>
        `;
        list.appendChild(div);
      });
    } else {
      // Fallback to Events if no trades
      const hits = (res.events || []).filter(e => e.is_triggered);
      hits.slice(0, 20).forEach((ev) => {
        const div = document.createElement("div");
        div.className = "border border-surface-border rounded-lg p-3 bg-background-dark/50";
        div.innerHTML = `
          <div class="flex justify-between text-xs text-slate-400">
            <span>${ev.trade_date || "--"}</span>
            <span>命中</span>
          </div>
          <div class="text-sm text-white mt-1 font-mono">收盤 ${ev.close_price}</div>
          <div class="text-xs text-primary">總分：${ev.total_score != null ? ev.total_score.toFixed(1) : "--"}</div>
        `;
        list.appendChild(div);
      });
    }
  }

  renderCharts(res.events || []);
}

function renderCharts(events) {
  const sorted = [...events].sort((a, b) => (a.trade_date || "").localeCompare(b.trade_date || ""));
  const labels = sorted.map((e) => e.trade_date);
  const scoreData = sorted.map((e) => e.total_score ?? null);
  const closeData = sorted.map((e) => e.close_price ?? null);
  const signalData = sorted.map((e, idx) => (e.is_triggered ? closeData[idx] : null));
  const fwd5 = sorted.map((e) => {
    const v = e.forward_returns?.d5;
    return v == null ? null : v * 100;
  });

  if (state.btScoreChart) state.btScoreChart.destroy();
  if (state.btReturnChart) state.btReturnChart.destroy();

  const scoreCtx = el("btScoreChart");
  const retCtx = el("btReturnChart");
  if (scoreCtx) {
    state.btScoreChart = new Chart(scoreCtx, {
      type: "line",
      data: {
        labels,
        datasets: [
          {
            label: "買入訊號",
            data: signalData,
            type: "scatter",
            backgroundColor: "#facc15",
            borderColor: "#fff",
            borderWidth: 1.5,
            pointRadius: 6,
            pointHoverRadius: 9,
            yAxisID: "y",
            zIndex: 10
          },
          {
            label: "收盤價格",
            data: closeData,
            borderColor: "#0ddff2",
            backgroundColor: "rgba(13,223,242,0.05)",
            borderWidth: 2,
            fill: true,
            yAxisID: "y",
            pointRadius: 0,
            tension: 0.2,
            zIndex: 5
          },
          {
            label: "量化總分",
            data: scoreData,
            borderColor: "rgba(124,58,237,0.4)",
            backgroundColor: "transparent",
            borderWidth: 1,
            fill: false,
            yAxisID: "y1",
            pointRadius: 0,
            tension: 0.1,
            zIndex: 1
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: "index", intersect: false },
        scales: {
          y: {
            position: "left",
            title: { display: true, text: "價格 (USDT)", color: "#64748b", font: { size: 10, weight: "bold" } },
            ticks: { color: "#cbd5e1", font: { family: "JetBrains Mono, monospace", size: 10 } },
            grid: { color: "rgba(255,255,255,0.03)" }
          },
          y1: {
            position: "right",
            min: 0,
            max: 100,
            title: { display: true, text: "評分", color: "#64748b", font: { size: 10, weight: "bold" } },
            ticks: { color: "rgba(124,58,237,0.6)", font: { size: 10 } },
            grid: { drawOnChartArea: false }
          },
          x: {
            ticks: { color: "#64748b", font: { size: 9 }, maxRotation: 45, minRotation: 45 },
            grid: { display: false }
          },
        },
        plugins: {
          legend: {
            position: "top",
            align: "end",
            labels: { color: "#94a3b8", font: { size: 10, weight: "bold" }, usePointStyle: true, boxWidth: 6 }
          },
          tooltip: {
            backgroundColor: "rgba(15, 23, 42, 0.9)",
            titleFont: { size: 12 },
            bodyFont: { size: 11 },
            padding: 12,
            borderColor: "rgba(255,255,255,0.1)",
            borderWidth: 1
          }
        },
      },
    });
  }

  if (retCtx) {
    state.btReturnChart = new Chart(retCtx, {
      type: "bar",
      data: { labels, datasets: [{ label: "Forward Return (d5)", data: fwd5, backgroundColor: "#10b981" }] },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: { ticks: { color: "#cbd5e1", callback: (v) => `${v}%` }, grid: { color: "rgba(255,255,255,0.05)" } },
          x: { ticks: { color: "#94a3b8" }, grid: { color: "rgba(255,255,255,0.05)" } },
        },
        plugins: { legend: { labels: { color: "#e2e8f0" } } },
      },
    });
  }
}

function setBusy(id, busy, labelWhenIdle) {
  const btn = el(id);
  if (!btn) return;
  if (busy) {
    btn.disabled = true;
    btn.dataset.originalLabel = btn.textContent;
    btn.textContent = "執行中...";
    btn.classList.add("opacity-50", "cursor-wait");
  } else {
    btn.disabled = false;
    btn.textContent = labelWhenIdle || btn.dataset.originalLabel || "";
    btn.classList.remove("opacity-50", "cursor-wait");
  }
}

async function runBacktest(isAuto = false) {
  const payload = readForm();
  if (!payload.start_date || !payload.end_date) return;
  console.log("[Backtest] Running...", isAuto ? "(auto)" : "");
  if (!isAuto) setBusy("runBacktestBtn", true, "執行中...");
  try {
    const res = await api("/api/analysis/backtest", { method: "POST", body: payload });
    renderResult(res);
    if (!isAuto) setAlert("回測完成", "success");
    else setAlert(""); // Clear any old error alerts on success
  } catch (err) {
    if (!isAuto) setAlert(err.message, "error");
    renderResult({ error: err.message });
  } finally {
    if (!isAuto) setBusy("runBacktestBtn", false, "執行回測");
  }
}

const debouncedRunBacktest = debounce(() => runBacktest(true), 400);

// runDbStrategy removed, now unified into runBacktest


async function confirmSaveScoringStrategy() {
  const name = el("newStrategyName")?.value.trim();
  const slug = el("newStrategySlug")?.value.trim();
  if (!name || !slug) {
    console.warn("[SaveStrategy] Name or Slug missing");
    return setAlert("請輸入名稱與代碼", "error");
  }

  setBusy("confirmSaveStrategyBtn", true, "儲存中");
  try {
    const rules = [];

    // 1. Entry Rules
    rules.push({
      condition_name: "進場 AI 核心評分",
      type: "BASE_SCORE",
      params: {},
      weight: parseFloat(el("btEntryScoreWeight").value) || 0,
      rule_type: "entry"
    });

    if (el("btEntryUseChange").checked) {
      rules.push({
        condition_name: "進場漲幅要求",
        type: "PRICE_RETURN",
        params: { days: 1, min: (parseFloat(el("btEntryChangeMin").value) || 0) / 100 },
        weight: parseFloat(el("btEntryChangeBonus").value) || 0,
        rule_type: "entry"
      });
    }
    if (el("btEntryUseVolume").checked) {
      rules.push({
        condition_name: "進場量能激增",
        type: "VOLUME_SURGE",
        params: { min: parseFloat(el("btEntryVolMin").value) || 0 },
        weight: parseFloat(el("btEntryVolumeBonus").value) || 0,
        rule_type: "entry"
      });
    }
    if (el("btEntryUseMa").checked) {
      rules.push({
        condition_name: "進場均線偏離",
        type: "MA_DEVIATION",
        params: { ma: 20, min: (parseFloat(el("btEntryMaGapMin").value) || 0) / 100 },
        weight: parseFloat(el("btEntryMaBonus").value) || 0,
        rule_type: "entry"
      });
    }
    if (el("btEntryUseRange")?.checked) {
      rules.push({
        condition_name: "進場位階限制",
        type: "RANGE_POS",
        params: { days: 20, min: (parseFloat(el("btEntryRangeMin").value) || 0) / 100 },
        weight: parseFloat(el("btEntryRangeBonus").value) || 0,
        rule_type: "entry"
      });
    }

    // 2. Exit Rules
    rules.push({
      condition_name: "出場 AI 核心評分",
      type: "BASE_SCORE",
      params: {},
      weight: parseFloat(el("btExitScoreWeight").value) || 0,
      rule_type: "exit"
    });

    if (el("btExitUseChange").checked) {
      rules.push({
        condition_name: "出場跌幅止損",
        type: "PRICE_RETURN",
        params: { days: 1, min: (parseFloat(el("btExitChangeMin").value) || 0) / 100 },
        weight: parseFloat(el("btExitChangeBonus").value) || 0,
        rule_type: "exit"
      });
    }
    if (el("btExitUseMa").checked) {
      rules.push({
        condition_name: "出場均線支撐",
        type: "MA_DEVIATION",
        params: { ma: 20, min: (parseFloat(el("btExitMaGapMin").value) || 0) / 100 },
        weight: parseFloat(el("btExitMaBonus").value) || 0,
        rule_type: "exit"
      });
    }

    const payload = {
      name: name,
      slug: slug,
      threshold: parseFloat(el("btEntryTotalMin").value) || 0,
      exit_threshold: parseFloat(el("btExitTotalMin").value) || 0,
      rules: rules
    };

    console.log("[SaveStrategy] Sending payload:", payload);
    await api("/api/analysis/strategies/save-scoring", { method: "POST", body: payload });

    const isUpdate = el("newStrategySlug").disabled;
    setAlert(`策略 [${name}] 已${isUpdate ? '更新' : '成功存入資料庫'}`, "success");
    el("saveAsStrategyForm").classList.add("hidden");

    // Refresh strategy list
    const select = el("btStrategySlug");
    if (select) {
      while (select.options.length > 1) select.remove(1);
    }
    await fetchStrategies();
  } catch (err) {
    console.error("[SaveStrategy] Failed:", err);
    setAlert(err.message, "error");
  } finally {
    setBusy("confirmSaveStrategyBtn", false, "確認儲存");
  }
}

async function fetchStrategies() {
  try {
    const res = await api("/api/analysis/strategies");
    const select = el("btStrategySlug");
    if (res.strategies) {
      res.strategies.forEach(s => {
        const opt = document.createElement("option");
        opt.value = s.slug;
        opt.textContent = `${s.name} (${s.slug})`;
        select.appendChild(opt);
      });
    }
  } catch (err) {
    console.error("Failed to fetch strategies:", err);
  }
}

async function loadStrategyDetails(slug) {
  if (!slug) return;
  console.log(`[Strategy] Loading details for ${slug}...`);
  try {
    const res = await api(`/api/analysis/strategies/get?slug=${slug}`);
    if (res.strategy) {
      const s = res.strategy;
      const cfg = {
        symbol: s.base_symbol,
        entry: { weights: { score: 0 }, thresholds: {}, flags: {}, total_min: s.threshold },
        exit: { weights: { score: 0 }, thresholds: {}, flags: {}, total_min: s.exit_threshold || 45 }
      };

      (s.rules || []).forEach(r => {
        const type = r.condition?.type;
        const params = r.condition?.params || {};
        const side = r.rule_type === "exit" ? "exit" : "entry";
        const target = cfg[side];

        if (type === "PRICE_RETURN" && params.days === 1) {
          target.weights.change_bonus = r.weight;
          target.thresholds.change_min = params.min * 100;
          target.flags.use_change = true;
        } else if (type === "VOLUME_SURGE") {
          target.weights.volume_bonus = r.weight;
          target.thresholds.volume_ratio_min = params.min;
          target.flags.use_volume = true;
        } else if (type === "MA_DEVIATION") {
          target.weights.ma_bonus = r.weight;
          target.thresholds.ma_gap_min = params.min * 100;
          target.flags.use_ma = true;
        } else if (type === "BASE_SCORE") {
          target.weights.score = r.weight;
        } else if (type === "RANGE_POS") {
          target.weights.range_bonus = r.weight;
          target.thresholds.range_min = params.min * 100;
          target.flags.use_range = true;
        } else {
          // Fallback for unknown types (like rsi_oversold) - add to score to keep Max Score accurate
          console.log("Adding unknown rule weight to score for visualization:", type, r.weight);
          target.weights.score += r.weight;
        }
      });
      fillForm(cfg);

      // Pre-fill Save Form for Editing
      if (el("newStrategyName")) el("newStrategyName").value = s.name;
      if (el("newStrategySlug")) {
        el("newStrategySlug").value = s.slug;
        el("newStrategySlug").disabled = true; // Slug is the unique key, don't change while editing
        el("newStrategySlug").classList.add("opacity-50", "cursor-not-allowed");
      }
      if (el("saveFormTitle")) el("saveFormTitle").textContent = "更新現有策略 (Update Strategy)";
      if (el("confirmSaveStrategyBtn")) el("confirmSaveStrategyBtn").textContent = "確認並更新 (Update)";

      fillForm(cfg);
      setAlert(`已載入策略: ${s.name}，上方的回測條件已更新`, "success");
      // Trigger a run with new parameters
      setTimeout(() => runBacktest(true), 200);
    }
  } catch (err) {
    console.error("[Strategy] Load failed:", err);
    setAlert(err.message, "error");
  }
}

async function savePreset() {
  const payload = readForm();
  setBusy("savePresetBtn", true, "儲存中");
  console.log("[Preset] Saving...", payload);
  try {
    await api("/api/analysis/backtest/preset", { method: "POST", body: payload });
    setAlert("已儲存為預設", "success");
  } catch (err) {
    setAlert(err.message, "error");
  } finally {
    setBusy("savePresetBtn", false, "儲存為預設");
  }
}

async function loadPreset() {
  setBusy("loadPresetBtn", true, "載入中");
  try {
    console.log("[Preset] Loading...");
    const res = await api("/api/analysis/backtest/preset");
    if (res.preset) {
      fillForm(res.preset);
      setAlert("已載入預設", "success");
    } else {
      setAlert(res.message || "尚無預設", "info");
    }
  } catch (err) {
    setAlert(err.message, "error");
  } finally {
    setBusy("loadPresetBtn", false, "載入預設");
  }
}

async function login(email, password) {
  console.log(`[Auth] Logging in as ${email}...`);
  const data = await api("/api/auth/login", { method: "POST", body: { email, password }, requireAuth: false });
  state.token = data.access_token;
  state.lastEmail = email;
  localStorage.setItem("aat_token", state.token);
  localStorage.setItem("aat_email", email);
  el("loginStatus").textContent = `已登入：${email}`;
  el("loginBtn").classList.add("hidden");
  el("logoutBtn").classList.remove("hidden");
  setAlert("登入成功", "success");
}

function logout(silent = false) {
  console.log("[Auth] Logging out...");
  state.token = "";
  state.role = "";
  localStorage.removeItem("aat_token");
  el("loginStatus").textContent = "未登入";
  el("loginBtn").classList.remove("hidden");
  el("logoutBtn").classList.add("hidden");
  if (!silent) setAlert("已登出", "info");
}

function setupAuth() {
  initAuthModal((data) => {
    state.token = data.access_token;
    state.lastEmail = localStorage.getItem("aat_email");

    // Refresh page to reload strategies and presets for the user
    window.location.reload();
  });

  el("logoutBtn").addEventListener("click", () => logout());
}




function bootstrap() {
  updateExchangeLink();
  initSidebar();
  initBinanceConfigModal();
  initGlobalEnvSelector((env) => {
    console.log("[Backtest] Env switched to:", env);
    // You can add logic here if backtest needs to react to env change
  });

  window.onBinanceConfigUpdate = () => {
    // Refresh page or update data
    window.location.reload();
  };

  setupAuth();

  el("runBacktestBtn").addEventListener("click", () => runBacktest());
  el("saveAsStrategyBtn").addEventListener("click", () => {
    const form = el("saveAsStrategyForm");
    form.classList.remove("hidden");

    // If we're not specifically in "editing mode" (slug in URL), ensure form is clean
    const params = new URLSearchParams(window.location.search);
    if (!params.get("slug")) {
      el("newStrategySlug").disabled = false;
      el("newStrategySlug").classList.remove("opacity-50", "cursor-not-allowed");
      el("saveFormTitle").textContent = "儲存為全新策略 (Save New)";
      el("confirmSaveStrategyBtn").textContent = "確認儲存 (Save)";
    }
  });
  el("cancelSaveStrategyBtn").addEventListener("click", () => el("saveAsStrategyForm").classList.add("hidden"));
  el("confirmSaveStrategyBtn").addEventListener("click", confirmSaveScoringStrategy);
  el("savePresetBtn").addEventListener("click", savePreset);
  el("loadPresetBtn").addEventListener("click", loadPreset);

  fetchStrategies().then(() => {
    const params = new URLSearchParams(window.location.search);
    const slug = params.get("slug");
    if (slug) {
      el("btStrategySlug").value = slug;
      loadStrategyDetails(slug);
    }
  });

  el("btStrategySlug").addEventListener("change", (e) => {
    if (e.target.value) {
      loadStrategyDetails(e.target.value);
    }
  });

  // Checkboxes
  const checkboxes = [
    "btEntryUseChange", "btEntryUseVolume", "btEntryUseMa", "btEntryUseRange",
    "btExitUseChange", "btExitUseMa", "btExitUseVolume"
  ];
  checkboxes.forEach((id) => {
    const input = el(id);
    if (input) {
      input.addEventListener("change", () => {
        updateVisibility();
        debouncedRunBacktest();
      });
    }
  });

  // Numeric and Text Inputs
  const inputs = [
    "btSymbol", "btStart", "btEnd", "btHorizons",
    "btEntryTotalMin", "btEntryScoreWeight", "btEntryChangeMin", "btEntryChangeBonus",
    "btEntryVolMin", "btEntryVolumeBonus", "btEntryMaGapMin", "btEntryMaBonus",
    "btEntryRangeMin", "btEntryRangeBonus",
    "btExitTotalMin", "btExitScoreWeight", "btExitChangeMin", "btExitChangeBonus",
    "btExitMaGapMin", "btExitMaBonus", "btExitVolMin", "btExitVolumeBonus"
  ];
  inputs.forEach(id => {
    const input = el(id);
    if (!input) return;
    const eventType = input.type === "date" ? "change" : "input";
    input.addEventListener(eventType, debouncedRunBacktest);
  });

  updateVisibility();

  // Set default dates: 2024/1/1 to Today
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  if (!el("btStart").value) el("btStart").value = "2024-01-01";
  if (!el("btEnd").value) el("btEnd").value = `${y}-${m}-${d}`;

  if (state.token) {
    el("loginStatus").textContent = state.lastEmail ? `已登入：${state.lastEmail}` : "已登入";
  }

  // Initial run
  setTimeout(() => runBacktest(true), 800);

  // Binance Live
  el("refreshBinanceBtn")?.addEventListener("click", fetchBinanceInfo);
  el("btStrategySlug").addEventListener("change", updateMonitoringUI);
  el("checkExecuteBtn").addEventListener("click", toggleMonitoring);
  fetchBinanceInfo();
  setInterval(updateMonitoringUI, 10000);
  setInterval(fetchBinanceInfo, 30000);
}



async function fetchBinanceInfo() {
  const balanceEl = el("binanceBalance");
  const assetsEl = el("binanceAssets");
  const tag = el("binanceTag");
  const title = el("binanceHeaderTitle");

  if (!balanceEl) return;

  try {
    const health = await api("/api/health");
    const activeEnv = health.active_env || (health.use_testnet ? "test" : "prod");

    if (activeEnv === "prod") {
      if (title) title.textContent = "正式站連線觀測 (Binance Mainnet)";
      if (tag) {
        tag.textContent = "Live Mode";
        tag.className = "px-2 py-0.5 rounded bg-amber-500/10 text-amber-500 border border-amber-500/20 text-[10px] font-bold uppercase";
      }
    } else if (activeEnv === "paper") {
      if (title) title.textContent = "虛擬實盤監測 (Paper Trading)";
      if (tag) {
        tag.textContent = "Paper Mode";
        tag.className = "px-2 py-0.5 rounded bg-primary/10 text-primary border border-primary/20 text-[10px] font-bold uppercase";
      }
    } else {
      if (title) title.textContent = "測試網連線觀測 (Binance Testnet)";
      if (tag) {
        tag.textContent = "Testnet";
        tag.className = "px-2 py-0.5 rounded bg-secondary/10 text-secondary border border-secondary/20 text-[10px] font-bold uppercase";
      }
    }

    assetsEl.textContent = "Fetching...";
    const data = await api("/api/admin/binance/account");
    if (data.account && data.account.balances) {
      const usdt = data.account.balances.find(b => b.asset === "USDT");
      const btc = data.account.balances.find(b => b.asset === "BTC");

      if (usdt) {
        balanceEl.innerHTML = `${parseFloat(usdt.free).toFixed(2)} <span class="text-xs font-normal text-slate-500">USDT</span>`;
      }

      let assetStr = "";
      if (btc) assetStr += `BTC: ${parseFloat(btc.free).toFixed(6)}`;
      assetsEl.textContent = assetStr || "No other assets found";
      assetsEl.classList.remove("text-danger", "text-primary/70");
    }
  } catch (err) {
    console.error("Failed to fetch Binance account:", err);
    // Check if we are in a mock/paper mode
    const isMock = (tag && tag.textContent.toUpperCase().includes("PAPER"));

    if (isMock) {
      balanceEl.innerHTML = `VIRTUAL <span class="text-xs font-normal text-slate-500">USDT</span>`;
      assetsEl.textContent = "未偵測到實盤金鑰 (API-401 已攔截)";
      assetsEl.classList.remove("text-danger");
      assetsEl.classList.add("text-primary/70");
    } else {
      balanceEl.textContent = "連線失敗";
      assetsEl.textContent = "請檢查 API Key 或連線環境";
      assetsEl.classList.add("text-danger");
    }
  }
}


async function updateMonitoringUI() {
  const slug = el("btStrategySlug").value;
  const statusEl = el("monitoringStatus");
  const btn = el("checkExecuteBtn");
  const logEl = el("executionLog");

  if (!slug) {
    statusEl.innerHTML = `<div class="size-1.5 rounded-full bg-slate-500"></div> STANDBY`;
    statusEl.className = "flex items-center gap-1.5 text-[10px] uppercase font-bold text-slate-500";
    btn.innerHTML = `<span class="material-symbols-outlined text-sm">play_arrow</span> 啟動監聽監測 (Start Monitor)`;
    btn.className = "flex-1 px-4 py-2 rounded bg-primary/20 text-primary border border-primary/40 hover:bg-primary/30 text-xs font-bold transition-all flex items-center justify-center gap-2";
    return;
  }

  try {
    const data = await api(`/api/analysis/strategies/get?slug=${slug}`);
    const strat = data.strategy;
    if (strat.is_active) {
      statusEl.innerHTML = `<div class="size-1.5 rounded-full bg-success animate-pulse"></div> MONITORING`;
      statusEl.className = "flex items-center gap-1.5 text-[10px] uppercase font-bold text-success";
      btn.innerHTML = `<span class="material-symbols-outlined text-sm">stop</span> 停止監聽 (Stop Monitor)`;
      btn.className = "flex-1 px-4 py-2 rounded bg-danger/20 text-danger border border-danger/40 hover:bg-danger/30 text-xs font-bold transition-all flex items-center justify-center gap-2";
      logEl.textContent = `Active monitoring for ${slug}...`;
    } else {
      statusEl.innerHTML = `<div class="size-1.5 rounded-full bg-slate-500"></div> STANDBY`;
      statusEl.className = "flex items-center gap-1.5 text-[10px] uppercase font-bold text-slate-500";
      btn.innerHTML = `<span class="material-symbols-outlined text-sm">play_arrow</span> 啟動監聽監測 (Start Monitor)`;
      btn.className = "flex-1 px-4 py-2 rounded bg-primary/20 text-primary border border-primary/40 hover:bg-primary/30 text-xs font-bold transition-all flex items-center justify-center gap-2";
      logEl.textContent = `Strategy ${slug} is currently idle.`;
    }
  } catch (err) {
    console.error("Failed to fetch strat status:", err);
  }
}

async function toggleMonitoring() {
  const slug = el("btStrategySlug").value;
  if (!slug) {
    alert("請先選取一個資料庫策略。");
    return;
  }

  const btn = el("checkExecuteBtn");
  const isRunning = btn.textContent.includes("停止");
  const logEl = el("executionLog");

  try {
    const data = await api(`/api/analysis/strategies/get?slug=${slug}`);
    const id = data.strategy.id;
    const minBalance = parseFloat(el("autoStopLimit").value) || 0;

    if (!isRunning) {
      logEl.textContent = "Starting monitoring...";
      // Activation will now follow system default mode (Live/Paper/Testnet)
      await api(`/api/admin/strategies/${id}/activate`, {
        method: "POST",
        body: {
          env: "", // Backend will use default system mode
          auto_stop_min_balance: minBalance
        }
      });
      logEl.textContent = "Monitoring started. Program is now listening to Binance...";
    } else {
      logEl.textContent = "Stopping monitoring...";
      await api(`/api/admin/strategies/${id}/deactivate`, { method: "POST" });
      logEl.textContent = "Monitoring stopped.";
    }
    updateMonitoringUI();
    setTimeout(fetchBinanceInfo, 1000);
  } catch (err) {
    logEl.textContent = `ERROR: ${err.message}`;
    logEl.classList.add("text-danger");
  }
}

bootstrap();
