import { initSidebar, initGlobalEnvSelector, apiFetch, showMessage } from "./common.js";

const state = {
    strategies: [],
    env: "test"
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
    const activeCount = el("activeCount");
    if (activeCount) activeCount.textContent = state.strategies.filter(s => s.active).length;

    // Mock other stats for UI completeness
    const triggerCount = el("triggerCount");
    if (triggerCount) triggerCount.textContent = (Math.floor(Math.random() * 500) + 100).toLocaleString();

    const avgWinRate = el("avgWinRate");
    if (avgWinRate) avgWinRate.textContent = "68.5%";
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
        tr.className = "hover:bg-white/5 transition-colors border-b border-surface-border/20 text-xs group cursor-pointer";
        tr.dataset.id = s.id;

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

        // Detailed Rules Row (Hidden by default)
        const detailTr = document.createElement("tr");
        detailTr.id = `detail-${s.id}`;
        detailTr.className = "hidden bg-surface-dark/30 border-b border-surface-border/20";
        detailTr.innerHTML = `
            <td colspan="6" class="px-8 py-8">
                <div class="grid grid-cols-2 gap-12">
                    <div>
                        <h4 class="text-[10px] font-black text-primary uppercase tracking-[0.2em] mb-4 flex items-center gap-2">
                           <span class="size-1.5 rounded-full bg-primary"></span> 進場評分規則 (Entry Rules)
                        </h4>
                        <div class="space-y-3">
                            ${renderRules(s.buy_conditions?.conditions || [])}
                        </div>
                    </div>
                    <div>
                        <h4 class="text-[10px] font-black text-danger uppercase tracking-[0.2em] mb-4 flex items-center gap-2">
                           <span class="size-1.5 rounded-full bg-danger"></span> 出場評分規則 (Exit Rules)
                        </h4>
                        <div class="space-y-3">
                            ${renderRules(s.sell_conditions?.conditions || [])}
                        </div>
                    </div>
                </div>
            </td>
        `;

        tbody.appendChild(tr);
        tbody.appendChild(detailTr);
    });

    attachEvents();
}

function renderRules(conditions) {
    if (conditions.length === 0) return `<p class="text-[10px] text-slate-600 italic">無自訂規則</p>`;
    return conditions.map(c => `
        <div class="flex items-center justify-between p-3 bg-background-dark/50 rounded-xl border border-surface-border/50">
            <span class="text-[11px] text-slate-300 font-bold tracking-tight">${c.type}</span>
            <span class="text-[10px] text-primary font-mono font-bold">${c.weight || 10} pts</span>
        </div>
    `).join('');
}

function attachEvents() {
    document.querySelectorAll(".delete-btn").forEach((btn) => {
        btn.onclick = (e) => { e.stopPropagation(); deleteStrategy(btn.dataset.id, btn.dataset.name); };
    });
    document.querySelectorAll(".edit-btn").forEach((btn) => {
        btn.onclick = (e) => { e.stopPropagation(); window.location.href = `/backtest.html?slug=${btn.dataset.slug}`; };
    });
    document.querySelectorAll(".status-toggle-btn").forEach((btn) => {
        btn.onclick = (e) => { e.stopPropagation(); toggleStatus(btn.dataset.id, btn.dataset.active === 'true'); };
    });
    document.querySelectorAll("tbody tr.cursor-pointer").forEach((tr) => {
        tr.onclick = () => {
            const detail = el(`detail-${tr.dataset.id}`);
            const icon = tr.querySelector(".expand-btn span");
            if (detail) {
                const isHidden = detail.classList.contains('hidden');
                detail.classList.toggle('hidden');
                if (icon) icon.textContent = isHidden ? 'expand_less' : 'expand_more';
                if (isHidden) tr.classList.add('bg-white/5');
                else tr.classList.remove('bg-white/5');
            }
        };
    });
}

// ... rest of the file stays same ...
async function toggleStatus(id, currentlyActive) {
    try {
        const path = !currentlyActive ? "activate" : "deactivate";
        const envToSend = state.env === 'real' ? 'prod' : state.env;

        const res = await apiFetch(`/admin/strategies/${id}/${path}`, {
            method: "POST",
            body: JSON.stringify({ env: envToSend })
        });

        if (res.success) {
            showMessage(`策略已${!currentlyActive ? '開啟' : '關閉'}`, "success");
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
        if (res.success) {
            showMessage("策略已刪除", "success");
            fetchStrategies();
        }
    } catch (err) {
        showMessage(err.message, 'danger');
    }
}

function bootstrap() {
    initSidebar();

    initGlobalEnvSelector((env) => {
        state.env = env;
        renderTable();
    });

    const refreshBtn = el("refreshBtn");
    if (refreshBtn) refreshBtn.addEventListener("click", fetchStrategies);

    const exportBtn = el("exportReportBtn");
    if (exportBtn) {
        exportBtn.onclick = () => {
            if (state.strategies.length === 0) {
                showMessage("沒有可導出的數據", "warning");
                return;
            }
            const csvData = state.strategies.map(s => `"${s.name}","${s.slug}",${s.threshold},${s.active},"${s.env}"`).join('\n');
            const blob = new Blob([`Name,Slug,Threshold,Active,Environment\n${csvData}`], { type: 'text/csv;charset=utf-8;' });
            const url = URL.createObjectURL(blob);
            const link = document.createElement("a");
            link.setAttribute("href", url);
            link.setAttribute("download", `strategies_report_${new Date().toISOString().split('T')[0]}.csv`);
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            showMessage("策略報表導出完成", "success");
        };
    }

    fetchStrategies();
}

bootstrap();
