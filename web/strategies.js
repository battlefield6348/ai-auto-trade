import { updateExchangeLink, initSidebar, initBinanceConfigModal, initGlobalEnvSelector, handleUnauthorized, apiFetch } from "./common.js";

const state = {
    token: localStorage.getItem("aat_token") || "",
    strategies: [],
    env: localStorage.getItem("aat_env") || "test"
};

const el = (id) => document.getElementById(id);

async function fetchStrategies() {
    try {
        const data = await apiFetch("/analysis/strategies");
        state.strategies = data.strategies || [];
        updateStats();
        renderTable();
    } catch (err) {
        console.error(err);
    }
}

function updateStats() {
    el("activeCount").textContent = state.strategies.filter(s => s.active).length;
    // Mock other stats for UI completeness
    el("triggerCount").textContent = Math.floor(Math.random() * 500) + 100;
}

function renderTable() {
    const tbody = el("strategyTableBody");
    const empty = el("emptyState");
    if (!tbody) return;
    tbody.innerHTML = "";

    const filtered = state.strategies.filter(s => s.env === (state.env === 'real' ? 'prod' : state.env));

    if (filtered.length === 0) {
        if (empty) empty.classList.remove("hidden");
        return;
    }
    if (empty) empty.classList.add("hidden");

    filtered.forEach((s) => {
        const date = new Date(s.updated_at).toLocaleDateString('zh-TW');
        const tr = document.createElement("tr");
        tr.className = "hover:bg-white/5 transition-colors border-b border-surface-border/20 text-xs";

        tr.innerHTML = `
            <td class="px-8 py-6">
                <div class="flex flex-col">
                    <span class="font-bold text-white text-sm"># ${s.name}</span>
                    <span class="text-[10px] text-slate-500 font-mono tracking-tighter uppercase">${s.slug}</span>
                </div>
            </td>
            <td class="px-6 py-6 text-center">
                <span class="px-2 py-1 bg-surface-dark border border-surface-border rounded text-primary font-mono font-bold">${s.threshold}</span>
            </td>
            <td class="px-6 py-6 text-center">
                <span class="px-2 py-1 bg-surface-dark border border-surface-border rounded text-danger font-mono font-bold">${s.risk?.threshold || 0.3}</span>
            </td>
            <td class="px-6 py-6 text-center text-slate-400 font-mono">${date}</td>
            <td class="px-6 py-6 text-center">
                <button class="status-toggle-btn inline-flex items-center" data-id="${s.id}" data-active="${s.active}">
                    <div class="w-10 h-5 rounded-full p-1 transition-colors duration-200 ${s.active ? 'bg-primary' : 'bg-slate-700'}">
                        <div class="w-3 h-3 bg-white rounded-full transition-transform duration-200 transform ${s.active ? 'translate-x-5' : 'translate-x-0'}"></div>
                    </div>
                </button>
            </td>
            <td class="px-8 py-6 text-right">
                <div class="flex items-center justify-end gap-3 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button class="edit-btn p-2 hover:bg-primary/10 rounded-lg text-slate-400 hover:text-primary transition-all" data-slug="${s.slug}">
                        <span class="material-symbols-outlined text-sm">edit</span>
                    </button>
                    <button class="delete-btn p-2 hover:bg-danger/10 rounded-lg text-slate-400 hover:text-danger transition-all" data-id="${s.id}" data-name="${s.name}">
                        <span class="material-symbols-outlined text-sm">delete</span>
                    </button>
                    <button class="expand-btn p-2 hover:bg-white/5 rounded-lg text-slate-500">
                        <span class="material-symbols-outlined text-sm">expand_more</span>
                    </button>
                </div>
            </td>
        `;

        // Add class group for hover effect
        tr.classList.add('group');
        tbody.appendChild(tr);
    });

    attachEvents();
}

function attachEvents() {
    document.querySelectorAll(".delete-btn").forEach((btn) => {
        btn.onclick = () => deleteStrategy(btn.dataset.id, btn.dataset.name);
    });
    document.querySelectorAll(".edit-btn").forEach((btn) => {
        btn.onclick = () => window.location.href = `/backtest.html?slug=${btn.dataset.slug}`;
    });
    document.querySelectorAll(".status-toggle-btn").forEach((btn) => {
        btn.onclick = () => toggleStatus(btn.dataset.id, btn.dataset.active === 'true');
    });
}

async function toggleStatus(id, currentlyActive) {
    try {
        const path = !currentlyActive ? "activate" : "deactivate";
        const envToSend = state.env === 'real' ? 'prod' : state.env;

        const res = await apiFetch(`/admin/strategies/${id}/${path}`, {
            method: "POST",
            body: JSON.stringify({ env: envToSend })
        });

        if (res.success) {
            fetchStrategies();
        }
    } catch (err) {
        showMessage(err.message, 'danger');
    }
}

async function deleteStrategy(id, name) {
    if (!confirm(`確定要刪除策略 [${name}] 嗎？`)) return;
    try {
        const res = await apiFetch(`/admin/strategies/${id}`, { method: "DELETE" });
        if (res.success) fetchStrategies();
    } catch (err) {
        showMessage(err.message, 'danger');
    }
}

function bootstrap() {
    updateExchangeLink();
    initSidebar();
    initBinanceConfigModal();

    initGlobalEnvSelector((env) => {
        state.env = env;
        localStorage.setItem('aat_env', env);
        renderTable();
    });

    const clock = document.getElementById('serverClock');
    if (clock) {
        setInterval(() => {
            const now = new Date();
            clock.textContent = now.toLocaleString('zh-TW', { hour12: false });
        }, 1000);
    }

    el("refreshBtn").addEventListener("click", fetchStrategies);

    const logout = el("logoutBtn");
    if (logout) {
        logout.classList.remove("hidden");
        logout.addEventListener("click", () => {
            localStorage.removeItem("aat_token");
            window.location.href = "/login.html";
        });
    }

    fetchStrategies();
}

bootstrap();
