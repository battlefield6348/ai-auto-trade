const state = {
  token: localStorage.getItem("aat_token") || "",
  lastEmail: localStorage.getItem("aat_email") || "",
  symbol: "BTCUSDT",
  chart: null,
  role: "",
  pollId: null,
  polling: false,
};

const el = (id) => document.getElementById(id);

function setAlert(message, type = "info") {
  const box = el("alert");
  if (!message) {
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
  box.textContent = message;
}

function authHeaders(requireAuth = true) {
  const headers = {};
  if (requireAuth && state.token) {
    headers["Authorization"] = `Bearer ${state.token}`;
  }
  return headers;
}

async function api(path, { method = "GET", body, requireAuth = true } = {}) {
  const headers = authHeaders(requireAuth);
  let payload = body;
  if (body && typeof body === "object" && !(body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(path, { method, headers, body: payload });
  if (res.status === 401) {
    logout(true);
    throw new Error("需要登入或權限不足");
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok || data.success === false) {
    const msg = data?.message || data?.error || data?.reason || res.statusText;
    throw new Error(msg);
  }
  if (res.headers.get("x-user-role")) {
    state.role = res.headers.get("x-user-role");
    el("roleLabel").textContent = state.role;
  }
  return data;
}

function formatTime(value) {
  if (!value) return "--";
  const date = new Date(value);
  if (isNaN(date)) return value;
  return date.toLocaleString("zh-TW", { hour12: false, timeZone: "Asia/Taipei" });
}

function formatDuration(seconds) {
  if (!seconds && seconds !== 0) return "--";
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s`;
}

function applyStatus(data) {
  el("nextRun").textContent = data.next_run ? formatTime(data.next_run).split(" ")[1] || formatTime(data.next_run) : "--";
  el("retryStrategy").textContent = Array.isArray(data.retry_strategy) ? data.retry_strategy.join(", ") : "--";
  el("nextRunHuman").textContent = data.next_run ? formatTime(data.next_run) : "";
  el("syntheticBadge").textContent = data.use_synthetic ? "ACTIVE" : "INACTIVE";
  el("autoInterval").textContent = data.auto_interval_seconds ? `${Math.round(data.auto_interval_seconds / 60)}m` : "--";
  el("dataSource").textContent = data.data_source || "--";
  if (data.last_run) applyLastRun(data.last_run);
  applyPermissions();
}

function applyLastRun(run) {
  const badge = el("latestBadge");
  const ingOK = run.ingestion?.success ?? true;
  const anOK = run.analysis?.success ?? true;
  const allOK = ingOK && (!run.analysis?.enabled || anOK);
  badge.textContent = allOK ? "Success" : "Partial/Error";
  badge.className = `px-2 py-0.5 rounded text-xs border flex items-center gap-1 ${allOK ? "bg-success/10 text-success border-success/30" : "bg-danger/10 text-danger border-danger/30"}`;

  el("lastStart").textContent = run.start ? formatTime(run.start) : "--";
  el("lastEnd").textContent = run.end ? formatTime(run.end) : "--";
  el("lastDuration").textContent = formatDuration(run.duration_seconds || 0);
  el("lastSucc").textContent = run.analysis?.success_count ?? "--";
  el("lastFail").textContent = run.analysis?.failure_count ?? (run.failures?.length || 0);
  const err = run.analysis?.error || run.ingestion?.error || "--";
  el("lastError").textContent = err || "--";
  if (run.trading_pair) {
    el("dataSource").textContent = run.data_source || el("dataSource").textContent;
  }
}

function applyJobHistory(list) {
  const container = el("historyList");
  container.innerHTML = "";
  if (!list || list.length === 0) {
    container.innerHTML = '<p class="text-slate-500 text-sm">暫無歷史紀錄</p>';
    return;
  }
  list.forEach((item) => {
    const div = document.createElement("div");
    const badge = item.analysis?.success ? "text-success" : "text-danger";
    div.className = "border border-surface-border rounded-lg p-3 bg-background-dark/50";
    div.innerHTML = `
      <div class="flex justify-between text-xs text-slate-400">
        <span>${item.kind.toUpperCase()} · 觸發者：${item.triggered_by || "--"}</span>
        <span>${formatTime(item.start)}</span>
      </div>
      <div class="text-sm text-white mt-1">耗時 ${formatDuration(item.duration_seconds || 0)}</div>
      <div class="text-xs ${badge}">分析成功：${item.analysis?.success ? "是" : "否"} · Ingestion：${item.ingestion?.success ? "是" : "否"}</div>
      <div class="text-xs text-slate-400">來源：${item.data_source || "--"}</div>
    `;
    container.appendChild(div);
  });
}

function renderOpResult(tag, payload) {
  const box = el("opResult");
  if (!box) return;
  if (!payload) {
    box.textContent = "尚未執行";
    return;
  }
  const rows = [];
  rows.push(`<div class="text-xs text-slate-400">操作：${tag}</div>`);
  if (payload.trade_date) rows.push(`<div>日期：${payload.trade_date}</div>`);
  if (payload.start_date && payload.end_date) rows.push(`<div>區間：${payload.start_date} ~ ${payload.end_date}</div>`);
  if (payload.duration_seconds != null) rows.push(`<div>耗時：${payload.duration_seconds}s</div>`);
  if (payload.ingestion) {
    rows.push(`<div>擷取：${payload.ingestion.success ? "成功" : "失敗"} ${payload.ingestion.error || ""}</div>`);
  }
  if (payload.analysis) {
    rows.push(`<div>分析：${payload.analysis.enabled ? (payload.analysis.success ? "成功" : "失敗") : "未啟用"} (${payload.analysis.success_count ?? 0}/${payload.analysis.total ?? 0}) ${payload.analysis.error || ""}</div>`);
  }
  if (Array.isArray(payload.failures) && payload.failures.length > 0) {
    rows.push(`<div class="mt-1 text-danger">失敗明細：</div>`);
    rows.push(`<ul class="list-disc list-inside text-danger text-xs">${payload.failures.slice(0, 5).map((f) => `<li>${f.trade_date || ""} ${f.stage || ""} ${f.reason || ""}</li>`).join("")}</ul>`);
  }
  if (payload.error) rows.push(`<div class="text-danger">錯誤：${payload.error}</div>`);
  box.innerHTML = rows.join("");
}

async function loadStatusAndHistory() {
  try {
    const status = await api("/api/admin/jobs/status");
    applyStatus(status);
    const hist = await api("/api/admin/jobs/history?limit=10");
    applyJobHistory(hist.items || []);
    setAlert("");
  } catch (err) {
    setAlert(err.message, "error");
  }
}

async function loadSummary() {
  try {
    const res = await api("/api/analysis/summary");
    el("summaryDate").textContent = res.trade_date || "--";
    el("summaryPair").textContent = res.trading_pair || "--";
    el("summaryTrend").textContent = res.trend || "--";
    el("summaryAdvice").textContent = res.advice || "";
    const m = res.metrics || {};
    el("summaryPrice").textContent = m.close_price ?? "--";
    el("summaryChange").textContent = m.change_percent != null ? `${(m.change_percent * 100).toFixed(2)}%` : "--";
    el("summaryReturn5").textContent = m.return_5d != null ? `${(m.return_5d * 100).toFixed(2)}%` : "--";
    el("summaryVolume").textContent = m.volume_ratio != null ? `${m.volume_ratio.toFixed(2)}x` : "--";
    el("summaryScore").textContent = m.score != null ? m.score.toFixed(1) : "--";
  } catch (err) {
    setAlert(`摘要：${err.message}`, "error");
  }
}

async function loadAnalysisHistory() {
  try {
    const hist = await api(`/api/analysis/history?symbol=${encodeURIComponent(state.symbol)}&limit=30`);
    const items = hist.items || [];
    const box = el("analysisList");
    box.innerHTML = "";
    if (items.length === 0) {
      box.innerHTML = '<p class="text-slate-500 text-sm">尚無分析紀錄</p>';
      return;
    }
    renderChart(items);
    items.slice(0, 8).reverse().forEach((item) => {
      const div = document.createElement("div");
      div.className = "border border-surface-border rounded-lg p-3 bg-background-dark/50";
      div.innerHTML = `
        <div class="flex justify-between text-xs text-slate-400">
          <span>${item.trade_date || "--"}</span>
          <span>${item.trading_pair || "BTCUSDT"}</span>
        </div>
        <div class="text-sm text-white mt-1 font-mono">收盤 ${item.close_price}</div>
        <div class="text-xs text-slate-300">日漲跌：${item.change_percent != null ? (item.change_percent * 100).toFixed(2) + "%" : "--"}</div>
        <div class="text-xs text-slate-300">近5日：${item.return_5d != null ? (item.return_5d * 100).toFixed(2) + "%" : "--"}</div>
        <div class="text-xs text-slate-300">量能倍數：${item.volume_ratio != null ? item.volume_ratio.toFixed(2) + "x" : "--"}</div>
        <div class="text-xs text-primary">Score：${item.score != null ? item.score.toFixed(1) : "--"}</div>
      `;
      box.appendChild(div);
    });
  } catch (err) {
    setAlert(`分析歷史：${err.message}`, "error");
  }
}

function renderChart(items) {
  if (!Array.isArray(items) || items.length === 0 || typeof Chart === "undefined") return;
  const ctx = el("analysisChart");
  const labels = items.map((i) => i.trade_date).reverse();
  const closeData = items.map((i) => i.close_price).reverse();
  const scoreData = items.map((i) => i.score).reverse();
  if (state.chart) state.chart.destroy();
  state.chart = new Chart(ctx, {
    type: "line",
    data: {
      labels,
      datasets: [
        { label: "收盤", data: closeData, borderColor: "#0ddff2", backgroundColor: "rgba(13,223,242,0.2)", yAxisID: "y" },
        { label: "Score", data: scoreData, borderColor: "#7c3aed", backgroundColor: "rgba(124,58,237,0.2)", yAxisID: "y1" },
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

async function login(email, password) {
  const data = await api("/api/auth/login", { method: "POST", body: { email, password }, requireAuth: false });
  state.token = data.access_token;
  state.lastEmail = email;
  localStorage.setItem("aat_token", state.token);
  localStorage.setItem("aat_email", email);
  el("loginStatus").textContent = `已登入：${email}`;
  el("loginBtn").classList.add("hidden");
  el("logoutBtn").classList.remove("hidden");
  setAlert("登入成功", "success");
  await loadStatusAndHistory();
  await Promise.all([loadSummary(), loadAnalysisHistory()]);
}

function logout() {
  state.token = "";
  localStorage.removeItem("aat_token");
  el("loginStatus").textContent = "未登入";
  el("loginBtn").classList.remove("hidden");
  el("logoutBtn").classList.add("hidden");
  state.role = "";
  el("roleLabel").textContent = "--";
  if (state.pollId) {
    clearInterval(state.pollId);
    state.pollId = null;
  }
  el("opResult").textContent = "尚未執行";
  setAlert("已登出", "info");
}

function setupClock() {
  const clock = el("clock");
  const tick = () => {
    const now = new Date();
    clock.textContent = now.toLocaleTimeString("zh-TW", { hour12: false, timeZone: "Asia/Taipei" });
  };
  tick();
  setInterval(tick, 1000);
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
  el("logoutBtn").addEventListener("click", logout);
}

function setupActions() {
  el("rerunBtn").addEventListener("click", async () => {
    const date = el("rerunDate").value;
    if (!date) return setAlert("請選擇日期", "error");
    setBusy("rerunBtn", true, "擷取並分析");
    try {
      const res = await api("/api/admin/ingestion/daily", { method: "POST", body: { trade_date: date, run_analysis: true } });
      setAlert(`已執行 ${date} 擷取+分析`, "success");
      renderOpResult("單日擷取+分析", res);
      await loadStatusAndHistory();
      await Promise.all([loadSummary(), loadAnalysisHistory()]);
    } catch (err) {
      setAlert(err.message, "error");
      renderOpResult("單日擷取+分析", { error: err.message });
    } finally {
      setBusy("rerunBtn", false, "擷取並分析");
    }
  });

  el("analysisOnlyBtn").addEventListener("click", async () => {
    const date = el("rerunDate").value;
    if (!date) return setAlert("請選擇日期", "error");
    setBusy("analysisOnlyBtn", true, "僅分析");
    try {
      const res = await api("/api/admin/analysis/daily", { method: "POST", body: { trade_date: date } });
      setAlert(`已執行 ${date} 單日分析`, "success");
      renderOpResult("單日分析", res);
      await loadStatusAndHistory();
      await Promise.all([loadSummary(), loadAnalysisHistory()]);
    } catch (err) {
      setAlert(err.message, "error");
      renderOpResult("單日分析", { error: err.message });
    } finally {
      setBusy("analysisOnlyBtn", false, "僅分析");
    }
  });

  el("rangeBtn").addEventListener("click", async () => {
    const start = el("rangeStart").value;
    const end = el("rangeEnd").value;
    if (!start || !end) return setAlert("請選擇起訖日期", "error");
    setBusy("rangeBtn", true, "回補並分析");
    try {
      const res = await api("/api/admin/ingestion/backfill", { method: "POST", body: { start_date: start, end_date: end, run_analysis: true } });
      setAlert(`已回補 ${start} ~ ${end}`, "success");
      renderOpResult("區間回補+分析", res);
      await loadStatusAndHistory();
      await Promise.all([loadSummary(), loadAnalysisHistory()]);
    } catch (err) {
      setAlert(err.message, "error");
      renderOpResult("區間回補+分析", { error: err.message });
    } finally {
      setBusy("rangeBtn", false, "回補並分析");
    }
  });

  el("refreshSummary").addEventListener("click", () => {
    loadSummary();
    loadAnalysisHistory();
  });

  el("refreshHistory").addEventListener("click", () => {
    loadAnalysisHistory();
  });
}

function startPolling() {
  if (state.pollId) clearInterval(state.pollId);
  const tick = async () => {
    if (state.polling) return;
    state.polling = true;
    try {
      await loadStatusAndHistory();
      await loadSummary();
      await loadAnalysisHistory();
    } catch (err) {
      // 已由內層 setAlert
    } finally {
      state.polling = false;
    }
  };
  tick();
  state.pollId = setInterval(tick, 60000);
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

function bootstrap() {
  setupClock();
  setupLoginModal();
  setupActions();
  el("symbolInput").addEventListener("change", () => {
    state.symbol = el("symbolInput").value.trim().toUpperCase() || "BTCUSDT";
    loadAnalysisHistory();
  });
  if (state.token) {
    el("loginStatus").textContent = state.lastEmail ? `已登入：${state.lastEmail}` : "已登入";
    el("loginBtn").classList.add("hidden");
    el("logoutBtn").classList.remove("hidden");
    startPolling();
  }
}

bootstrap();
function applyPermissions() {
  const restricted = state.role && !["admin", "analyst"].includes(state.role.toLowerCase());
  ["rerunBtn", "analysisOnlyBtn", "rangeBtn"].forEach((id) => {
    const btn = el(id);
    if (!btn) return;
    btn.disabled = restricted;
    btn.classList.toggle("opacity-50", restricted);
    btn.classList.toggle("cursor-not-allowed", restricted);
  });
  const msg = restricted ? "目前帳號無操作權限，僅可查看狀態。" : "";
  if (msg) setAlert(msg, "info");
}
