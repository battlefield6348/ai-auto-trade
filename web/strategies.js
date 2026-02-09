import { updateExchangeLink, initSidebar, initBinanceConfigModal, initGlobalEnvSelector } from "./common.js";

const state = {
    token: localStorage.getItem("aat_token") || "",
    strategies: [],
    env: "test" // Default
};

const el = (id) => document.getElementById(id);

function setAlert(msg, type = "info") {
    const box = el("alert");
    if (!box) return;
    if (!msg) {
        box.classList.add("hidden");
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
    setTimeout(() => box.classList.add("hidden"), 5000);
}

async function api(path, { method = "GET", body } = {}) {
    const headers = {};
    if (state.token) headers["Authorization"] = `Bearer ${state.token}`;

    let payload = body;
    if (body && typeof body === "object") {
        headers["Content-Type"] = "application/json";
        payload = JSON.stringify(body);
    }

    const res = await fetch(path, { method, headers, body: payload });
    if (res.status === 401) {
        window.location.href = "/";
        return;
    }

    const data = await res.json().catch(() => ({}));
    if (!res.ok || data.success === false) {
        throw new Error(data.message || data.error || res.statusText);
    }
    return data;
}

async function fetchStrategies() {
    try {
        const data = await api("/api/analysis/strategies");
        state.strategies = data.strategies || [];
        renderTable();
    } catch (err) {
        setAlert(err.message, "error");
    }
}

function renderTable() {
    const tbody = el("strategyTableBody");
    const empty = el("emptyState");
    if (!tbody) return;
    tbody.innerHTML = "";

    if (state.strategies.length === 0) {
        if (empty) empty.classList.remove("hidden");
        return;
    }
    if (empty) empty.classList.add("hidden");

    state.strategies.forEach((s) => {
        const date = new Date(s.updated_at).toLocaleString();
        const tr = document.createElement("tr");
        tr.className = "hover:bg-white/5 transition-colors group";
        const isActive = s.status === "active";
        // Map backend 'prod' to 'real' for display
        const displayEnv = s.env === 'prod' ? 'REAL' : s.env.toUpperCase();

        tr.innerHTML = `
      <td class="px-4 py-4 font-medium text-white">
        ${s.name}
        <span class="ml-2 text-[10px] px-1.5 py-0.5 rounded bg-slate-700 text-slate-300 border border-slate-600 uppercase">${displayEnv}</span>
      </td>
      <td class="px-4 py-4 font-mono text-xs text-slate-400">${s.slug}</td>
      <td class="px-4 py-4 text-center font-mono text-primary">${s.threshold}</td>
      <td class="px-4 py-4 text-center">
        <button class="status-toggle-btn group/toggle flex items-center justify-center mx-auto" data-id="${s.id}" data-status="${s.active ? 'active' : 'inactive'}">
          <div class="w-10 h-5 rounded-full p-1 transition-colors duration-200 ${s.active ? 'bg-primary' : 'bg-slate-700'}">
            <div class="w-3 h-3 bg-white rounded-full transition-transform duration-200 transform ${s.active ? 'translate-x-5' : 'translate-x-0'}"></div>
          </div>
        </button>
      </td>
      <td class="px-4 py-4 text-xs text-slate-500">${date}</td>
      <td class="px-4 py-4 text-right space-x-2">
        <button class="edit-btn text-xs px-2 py-1 rounded bg-surface-border text-slate-300 hover:text-white" data-slug="${s.slug}">
          修改
        </button>
        <button class="delete-btn text-xs px-2 py-1 rounded bg-danger/10 text-danger/80 hover:bg-danger/20 hover:text-danger" data-id="${s.id}" data-name="${s.name}">
          刪除
        </button>
      </td>
    `;
        tbody.appendChild(tr);
    });

    // Attach events
    document.querySelectorAll(".delete-btn").forEach((btn) => {
        btn.addEventListener("click", () => deleteStrategy(btn.dataset.id, btn.dataset.name));
    });
    document.querySelectorAll(".edit-btn").forEach((btn) => {
        btn.addEventListener("click", () => {
            // Redirect to backtest with this slug selected
            window.location.href = `/backtest.html?slug=${btn.dataset.slug}`;
        });
    });
    document.querySelectorAll(".status-toggle-btn").forEach((btn) => {
        btn.addEventListener("click", () => toggleStatus(btn.dataset.id, btn.dataset.status));
    });
}

async function toggleStatus(id, currentStatus) {
    const nextStatus = currentStatus === "active" ? "draft" : "active";
    try {
        const path = nextStatus === "active" ? "activate" : "deactivate";
        // Use current state.env
        // Backend expects 'prod' not 'real'
        const envToSend = state.env === 'real' ? 'prod' : state.env;

        await api(`/api/admin/strategies/${id}/${path}`, {
            method: "POST",
            body: { env: envToSend }
        });
        setAlert(`策略已${nextStatus === "active" ? "啟用" : "停用"} (${state.env})`, "success");
        fetchStrategies();
    } catch (err) {
        setAlert(err.message, "error");
    }
}

async function deleteStrategy(id, name) {
    if (!confirm(`確定要刪除策略 [${name}] 嗎？此動作不可復原。`)) return;

    try {
        await api(`/api/admin/strategies/${id}`, { method: "DELETE" });
        setAlert(`策略 ${name} 已刪除`, "success");
        fetchStrategies();
    } catch (err) {
        setAlert(err.message, "error");
    }
}

function bootstrap() {
    updateExchangeLink();
    initSidebar();
    initBinanceConfigModal();

    // Initialize Global Environment Selectors
    initGlobalEnvSelector((env) => {
        state.env = env;
        renderTable(); // Re-render table with filtering
    });
    if (!state.token) {
        window.location.href = "/";
        return;
    }

    el("refreshBtn").addEventListener("click", fetchStrategies);
    el("logoutBtn").classList.remove("hidden");
    el("logoutBtn").addEventListener("click", () => {
        localStorage.removeItem("aat_token");
        window.location.href = "/";
    });

    fetchStrategies();
}

bootstrap();
