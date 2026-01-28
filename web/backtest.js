const state = {
  token: localStorage.getItem("aat_token") || "",
  lastEmail: localStorage.getItem("aat_email") || "",
  role: "",
  btScoreChart: null,
  btReturnChart: null,
};

const el = (id) => document.getElementById(id);

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
  if (res.status === 401) throw new Error("需要登入或權限不足");
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
  return {
    symbol: (el("btSymbol").value || "BTCUSDT").trim().toUpperCase(),
    start_date: el("btStart").value,
    end_date: el("btEnd").value,
    weights: {
      score: 1,
      change_bonus: parse("btChangeBonus", 10),
      volume_bonus: parse("btVolumeBonus", 10),
      return_bonus: parse("btReturnBonus", 8),
      ma_bonus: parse("btMaBonus", 5),
    },
    thresholds: {
      total_min: parse("btTotalMin", 60),
      change_min: parse("btChangeMin", 0.5) / 100,
      volume_ratio_min: parse("btVolMin", 1.2),
      return5_min: parse("btRet5Min", 1) / 100,
      ma_gap_min: parse("btMaGapMin", 1) / 100,
    },
    flags: {
      use_change: el("btUseChange").checked,
      use_volume: el("btUseVolume").checked,
      use_return: el("btUseReturn").checked,
      use_ma: el("btUseMa").checked,
    },
    horizons: horizons.length ? horizons : [3, 5, 10],
  };
}

function updateVisibility() {
  const map = {
    btUseChange: ["wrapper-btChangeBonus", "wrapper-btChangeMin"],
    btUseVolume: ["wrapper-btVolumeBonus", "wrapper-btVolMin"],
    btUseReturn: ["wrapper-btReturnBonus", "wrapper-btRet5Min"],
    btUseMa: ["wrapper-btMaBonus", "wrapper-btMaGapMin"],
  };

  Object.entries(map).forEach(([checkId, wrapperIds]) => {
    const checked = el(checkId).checked;
    wrapperIds.forEach((id) => {
      const w = el(id);
      if (w) {
        if (checked) w.classList.remove("invisible");
        else w.classList.add("invisible");
      }
    });
  });
}

function fillForm(cfg) {
  if (!cfg) return;
  el("btSymbol").value = cfg.symbol || "BTCUSDT";
  el("btStart").value = cfg.start_date || "";
  el("btEnd").value = cfg.end_date || "";

  el("btChangeBonus").value = cfg.weights?.change_bonus ?? 10;
  el("btVolumeBonus").value = cfg.weights?.volume_bonus ?? 10;
  el("btReturnBonus").value = cfg.weights?.return_bonus ?? 8;
  el("btMaBonus").value = cfg.weights?.ma_bonus ?? 5;
  el("btTotalMin").value = cfg.thresholds?.total_min ?? 60;

  // Convert decimals back to percentages for UI
  el("btChangeMin").value = (cfg.thresholds?.change_min ?? 0.005) * 100;
  el("btVolMin").value = cfg.thresholds?.volume_ratio_min ?? 1.2;
  el("btRet5Min").value = (cfg.thresholds?.return5_min ?? 0.01) * 100;
  el("btMaGapMin").value = (cfg.thresholds?.ma_gap_min ?? 0.01) * 100;

  el("btUseChange").checked = cfg.flags?.use_change ?? true;
  el("btUseVolume").checked = cfg.flags?.use_volume ?? true;
  el("btUseReturn").checked = cfg.flags?.use_return ?? false;
  el("btUseMa").checked = cfg.flags?.use_ma ?? false;
  el("btHorizons").value = (cfg.horizons || [3, 5, 10]).join(",");
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
  const stats = res.stats || {};
  const rows = [];
  rows.push(`<div class="text-xs text-slate-400">交易對：${res.symbol || ""}</div>`);
  rows.push(`<div>期間：${res.start_date || "--"} ~ ${res.end_date || "--"} | 命中 ${res.total_events ?? 0} 筆</div>`);
  if (stats.returns) {
    rows.push(`<div class="mt-1 text-slate-400">報酬統計：</div>`);
    const items = Object.entries(stats.returns).map(
      ([k, v]) => `${k}: 報酬 ${(v.avg_return * 100).toFixed(2)}% / 勝率 ${(v.win_rate * 100).toFixed(1)}%`
    );
    rows.push(`<div>${items.join(" ｜ ")}</div>`);
  }
  box.innerHTML = rows.join("");

  if (list) {
    (res.events || []).slice(0, 20).forEach((ev) => {
      const div = document.createElement("div");
      div.className = "border border-surface-border rounded-lg p-3 bg-background-dark/50";
      div.innerHTML = `
        <div class="flex justify-between text-xs text-slate-400">
          <span>${ev.trade_date || "--"}</span>
          <span>${ev.trading_pair || ""}</span>
        </div>
        <div class="text-sm text-white mt-1 font-mono">收盤 ${ev.close_price}</div>
        <div class="text-xs text-slate-300">日漲跌：${ev.change_percent != null ? (ev.change_percent * 100).toFixed(2) + "%" : "--"}</div>
        <div class="text-xs text-slate-300">量能倍數：${ev.volume_ratio != null ? ev.volume_ratio.toFixed(2) + "x" : "--"}</div>
        <div class="text-xs text-primary">總分：${ev.total_score != null ? ev.total_score.toFixed(1) : "--"}</div>
      `;
      list.appendChild(div);
    });
  }

  renderCharts(res.events || []);
}

function renderCharts(events) {
  const sorted = [...events].sort((a, b) => (a.trade_date || "").localeCompare(b.trade_date || ""));
  const labels = sorted.map((e) => e.trade_date);
  const scoreData = sorted.map((e) => e.total_score ?? null);
  const closeData = sorted.map((e) => e.close_price ?? null);
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
          { label: "總分", data: scoreData, borderColor: "#7c3aed", backgroundColor: "rgba(124,58,237,0.2)", yAxisID: "y1" },
          { label: "收盤", data: closeData, borderColor: "#0ddff2", backgroundColor: "rgba(13,223,242,0.15)", yAxisID: "y" },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: { position: "left", ticks: { color: "#cbd5e1" }, grid: { color: "rgba(255,255,255,0.05)" } },
          y1: { position: "right", ticks: { color: "#cbd5e1" }, grid: { drawOnChartArea: false } },
          x: { ticks: { color: "#94a3b8" }, grid: { color: "rgba(255,255,255,0.05)" } },
        },
        plugins: { legend: { labels: { color: "#e2e8f0" } } },
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

async function runBacktest() {
  const payload = readForm();
  if (!payload.start_date || !payload.end_date) return setAlert("請選擇回測起訖日", "error");
  console.log("[Backtest] Running with payload:", payload);
  setBusy("runBacktestBtn", true, "執行回測");
  try {
    const res = await api("/api/analysis/backtest", { method: "POST", body: payload });
    renderResult(res);
    setAlert("回測完成", "success");
  } catch (err) {
    setAlert(err.message, "error");
    renderResult({ error: err.message });
  } finally {
    setBusy("runBacktestBtn", false, "執行回測");
  }
}

async function runDbStrategy() {
  const slug = el("btStrategySlug").value;
  if (!slug) return setAlert("請選擇一個策略", "error");
  const payload = {
    slug: slug,
    symbol: (el("btSymbol").value || "BTCUSDT").trim().toUpperCase(),
    start_date: el("btStart").value,
    end_date: el("btEnd").value,
  };
  if (!payload.start_date || !payload.end_date) return setAlert("請選擇回測起訖日", "error");

  setBusy("runDbStrategyBtn", true, "執行中");
  try {
    const res = await api("/api/analysis/backtest/slug", { method: "POST", body: payload });
    renderResult(res.result);
    setAlert(`策略 [${slug}] 回測完成`, "success");
  } catch (err) {
    setAlert(err.message, "error");
    renderResult({ error: err.message });
  } finally {
    setBusy("runDbStrategyBtn", false, "執行資料庫策略回測");
  }
}

async function confirmSaveScoringStrategy() {
  const name = el("newStrategyName").value.trim();
  const slug = el("newStrategySlug").value.trim();
  if (!name || !slug) return setAlert("請輸入名稱與代碼", "error");

  const form = readForm();
  const rules = [];
  if (el("btUseChange").checked) {
    rules.push({
      condition_name: "日漲跌",
      type: "PRICE_RETURN",
      params: {
        days: 1,
        min: (parseFloat(el("btChangeMin").value) || 0) / 100
      },
      weight: parseFloat(el("btChangeBonus").value) || 0
    });
  }
  if (el("btUseVolume").checked) {
    rules.push({
      condition_name: "量能激增",
      type: "VOLUME_SURGE",
      params: {
        min: parseFloat(el("btVolMin").value) || 0
      },
      weight: parseFloat(el("btVolumeBonus").value) || 0
    });
  }
  if (el("btUseReturn").checked) {
    rules.push({
      condition_name: "近5日報酬",
      type: "PRICE_RETURN",
      params: {
        days: 5,
        min: (parseFloat(el("btRet5Min").value) || 0) / 100
      },
      weight: parseFloat(el("btReturnBonus").value) || 0
    });
  }
  if (el("btUseMa").checked) {
    rules.push({
      condition_name: "MA 乖離",
      type: "MA_DEVIATION",
      params: {
        ma: 20,
        min: (parseFloat(el("btMaGapMin").value) || 0) / 100
      },
      weight: parseFloat(el("btMaBonus").value) || 0
    });
  }

  const payload = {
    name: name,
    slug: slug,
    threshold: parseFloat(el("btTotalMin").value) || 0,
    rules: rules
  };

  setBusy("confirmSaveStrategyBtn", true, "儲存中");
  try {
    await api("/api/analysis/strategies/save-scoring", { method: "POST", body: payload });
    setAlert(`新策略 [${name}] 已成功存入資料庫`, "success");
    el("saveAsStrategyForm").classList.add("hidden");
    // Refresh strategy list
    const select = el("btStrategySlug");
    while (select.options.length > 1) select.remove(1);
    await fetchStrategies();
  } catch (err) {
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

function setupLoginModal() {
  const dialog = el("loginModal");
  el("loginBtn").addEventListener("click", () => {
    dialog.showModal();
    if (state.lastEmail) el("loginEmail").value = state.lastEmail;
  });
  el("closeLogin").addEventListener("click", () => dialog.close());
  el("loginForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      await login(el("loginEmail").value, el("loginPassword").value);
      dialog.close();
    } catch (err) {
      setAlert(err.message, "error");
    }
  });
  el("logoutBtn").addEventListener("click", () => logout());
}



function bootstrap() {
  setupLoginModal();
  el("runBacktestBtn").addEventListener("click", runBacktest);
  el("runDbStrategyBtn").addEventListener("click", runDbStrategy);
  el("saveAsStrategyBtn").addEventListener("click", () => el("saveAsStrategyForm").classList.remove("hidden"));
  el("cancelSaveStrategyBtn").addEventListener("click", () => el("saveAsStrategyForm").classList.add("hidden"));
  el("confirmSaveStrategyBtn").addEventListener("click", confirmSaveScoringStrategy);
  el("savePresetBtn").addEventListener("click", savePreset);
  el("loadPresetBtn").addEventListener("click", loadPreset);

  fetchStrategies();

  ["btUseChange", "btUseVolume", "btUseReturn", "btUseMa"].forEach((id) => {
    el(id).addEventListener("change", updateVisibility);
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
}



bootstrap();
