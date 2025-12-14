const state = {
  token: "",
};

const api = async (path, options = {}) => {
  const headers = options.headers || {};
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  headers["Content-Type"] = headers["Content-Type"] || "application/json";
  const res = await fetch(path, { ...options, headers });
  const data = await res.json().catch(() => ({}));
  if (!res.ok || data.success === false) {
    const msg = data.error || res.statusText;
    throw new Error(`${res.status} ${data.error_code || ""} ${msg}`);
  }
  return data;
};

const setStatus = (msg) => {
  document.getElementById("status").textContent = msg;
};

const pretty = (el, obj) => {
  el.textContent = obj ? JSON.stringify(obj, null, 2) : "";
};

async function initHealth() {
  try {
    const res = await fetch("/api/health");
    const data = await res.json();
    if (data.success) {
      setStatus(
        `健康檢查 OK ｜ DB: ${data.db} ｜ 合成資料: ${data.use_synthetic ? "ON" : "OFF"}`
      );
    }
  } catch (_) {
    // ignore
  }
}
initHealth();

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
    setStatus(`已登入：${email}`);
    alert("登入成功");
  } catch (err) {
    alert(`登入失敗：${err.message}`);
  }
});

const today = new Date().toISOString().slice(0, 10);
["ingestDate", "analysisDate", "queryDate", "screenerDate"].forEach((id) => {
  const el = document.getElementById(id);
  if (el) el.value = today;
});

document.getElementById("ingestForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  const out = document.getElementById("ingestResult");
  try {
    const trade_date = document.getElementById("ingestDate").value;
    const res = await api("/api/admin/ingestion/daily", {
      method: "POST",
      body: JSON.stringify({ trade_date }),
    });
    pretty(out, res);
  } catch (err) {
    out.textContent = err.message;
  }
});

document.getElementById("analysisForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  const out = document.getElementById("analysisResult");
  try {
    const trade_date = document.getElementById("analysisDate").value;
    const res = await api("/api/admin/analysis/daily", {
      method: "POST",
      body: JSON.stringify({ trade_date }),
    });
    pretty(out, res);
  } catch (err) {
    out.textContent = err.message;
  }
});

document.getElementById("queryForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  const table = document.getElementById("queryTable");
  try {
    const trade_date = document.getElementById("queryDate").value;
    const limit = document.getElementById("queryLimit").value || 20;
    const offset = document.getElementById("queryOffset").value || 0;
    const res = await api(
      `/api/analysis/daily?trade_date=${trade_date}&limit=${limit}&offset=${offset}`
    );
    renderTable(table, res.items || []);
  } catch (err) {
    table.innerHTML = `<div class="error">${err.message}</div>`;
  }
});

document.getElementById("screenerForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  const table = document.getElementById("screenerTable");
  try {
    const trade_date = document.getElementById("screenerDate").value;
    const score_min = document.getElementById("scoreMin").value || 70;
    const volume_ratio_min = document.getElementById("volMin").value || 1.5;
    const limit = document.getElementById("screenerLimit").value || 20;
    const res = await api(
      `/api/screener/strong-stocks?trade_date=${trade_date}&score_min=${score_min}&volume_ratio_min=${volume_ratio_min}&limit=${limit}`
    );
    renderTable(table, res.items || []);
  } catch (err) {
    table.innerHTML = `<div class="error">${err.message}</div>`;
  }
});

document.getElementById("summaryBtn").addEventListener("click", async () => {
  const view = document.getElementById("summaryView");
  view.innerHTML = "讀取中...";
  try {
    const res = await api("/api/analysis/summary");
    renderSummary(view, res);
  } catch (err) {
    view.innerHTML = `<div class="error">${err.message}</div>`;
  }
});

function renderTable(container, rows) {
  if (!rows.length) {
    container.innerHTML = `<div class="pill">無資料</div>`;
    return;
  }
  const headers = [
    "trading_pair",
    "market_type",
    "close_price",
    "change_percent",
    "return_5d",
    "volume",
    "volume_ratio",
    "score",
  ];
  const thead = headers.map((h) => `<th>${h}</th>`).join("");
  const tbody = rows
    .map((r) => {
      return `<tr>${headers
        .map((h) => `<td>${fmt(r[h])}</td>`)
        .join("")}</tr>`;
    })
    .join("");
  container.innerHTML = `<table><thead><tr>${thead}</tr></thead><tbody>${tbody}</tbody></table>`;
}

function fmt(v) {
  if (v === null || v === undefined) return "";
  if (typeof v === "number") {
    return Math.abs(v) >= 1000 ? v.toLocaleString() : v.toFixed(3).replace(/\.?0+$/, "");
  }
  return v;
}

function renderSummary(el, res) {
  const trendLabel =
    res.trend === "bullish"
      ? "偏多"
      : res.trend === "bearish"
      ? "偏空"
      : "中性";
  const ret5 = res.metrics.return_5d != null ? fmt(res.metrics.return_5d) : "N/A";
  const volr = res.metrics.volume_ratio != null ? fmt(res.metrics.volume_ratio) : "N/A";
  el.innerHTML = `
    <div class="summary-pill">
      <div>日期：${res.trade_date}</div>
      <div>交易對：${res.trading_pair}</div>
      <div>趨勢：${trendLabel}</div>
    </div>
    <div class="summary-grid">
      <div><strong>收盤</strong><span>${fmt(res.metrics.close_price)}</span></div>
      <div><strong>日漲跌幅</strong><span>${fmt(res.metrics.change_percent)}</span></div>
      <div><strong>近5日報酬</strong><span>${ret5}</span></div>
      <div><strong>量能倍率</strong><span>${volr}</span></div>
      <div><strong>Score</strong><span>${fmt(res.metrics.score)}</span></div>
    </div>
    <div class="advice">${res.advice}</div>
  `;
}
