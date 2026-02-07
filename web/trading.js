import { updateExchangeLink, initBinanceConfigModal, initSidebar, initGlobalEnvSelector } from "./common.js";

const state = {
    token: localStorage.getItem("aat_token") || "",
    positions: [],
    trades: [],
    prices: {}, // symbol -> price
    env: "test"
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
        warning: "border-warning/30 bg-warning/10 text-warning",
    };
    box.className = `rounded border px-4 py-3 text-sm mb-4 ${palette[type] || palette.info}`;
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

async function fetchPrices(symbols) {
    for (const s of symbols) {
        try {
            const data = await api(`/api/admin/binance/price?symbol=${s}`);
            state.prices[s] = data.price;
        } catch (err) {
            console.error(`Failed to fetch price for ${s}:`, err);
        }
    }
}

async function fetchPositions() {
    try {
        const data = await api(`/api/admin/positions?env=${state.env}`);
        state.positions = data.positions || [];

        // Get unique symbols
        const symbols = [...new Set(state.positions.map(p => p.symbol || "BTCUSDT"))];
        await fetchPrices(symbols);
        renderPositions();
    } catch (err) {
        setAlert(err.message, "error");
    }
}

async function fetchTrades() {
    try {
        const data = await api(`/api/admin/trades?env=${state.env}`);
        state.trades = (data.trades || []).sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
        renderTrades();
    } catch (err) {
        setAlert(err.message, "error");
    }
}

function renderPositions() {
    const container = el("positionsList");
    if (!container) return;
    container.innerHTML = "";

    if (state.positions.length === 0) {
        container.innerHTML = `
      <div class="col-span-full py-20 text-center text-slate-500 bg-surface-dark/30 border border-dashed border-surface-border rounded-xl">
         <span class="material-symbols-outlined text-5xl mb-3 block opacity-20">hourglass_empty</span>
         目前無持有倉位 (${state.env})
      </div>
    `;
        return;
    }

    state.positions.forEach((p) => {
        const sym = p.symbol || "BTCUSDT";
        const currentPrice = state.prices[sym] || p.entry_price;
        const pnl = (currentPrice - p.entry_price) * p.size;
        const pnlPct = ((currentPrice / p.entry_price) - 1) * 100;
        const isProfit = pnl >= 0;

        const card = document.createElement("div");
        card.className = "bg-surface-dark border border-surface-border rounded-xl p-6 hover:border-primary/30 transition-all group";
        card.innerHTML = `
      <div class="flex items-start justify-between mb-4">
        <div>
           <h3 class="font-bold text-lg flex items-center gap-2">
             ${p.symbol || "BTCUSDT"}
             <span class="text-[10px] px-1.5 py-0.5 rounded bg-primary/10 text-primary border border-primary/20 uppercase">${p.env}</span>
           </h3>
           <p class="text-xs text-slate-500 font-mono">${p.id.split('-')[0]}</p>
        </div>
        <div class="text-right">
           <div class="text-sm font-bold ${isProfit ? 'text-success' : 'text-danger'}">
             ${isProfit ? '+' : ''}${pnl.toFixed(2)} USDT
           </div>
           <div class="text-[10px] font-mono ${isProfit ? 'text-success/70' : 'text-danger/70'}">
             ${isProfit ? '+' : ''}${pnlPct.toFixed(2)}%
           </div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4 mb-6">
         <div class="bg-background-dark/50 rounded-lg p-3 border border-surface-border/50">
            <p class="text-[10px] text-slate-500 uppercase mb-1">進場價格</p>
            <p class="text-sm font-mono text-slate-200">${p.entry_price.toLocaleString()}</p>
         </div>
         <div class="bg-background-dark/50 rounded-lg p-3 border border-surface-border/50">
            <p class="text-[10px] text-slate-500 uppercase mb-1">當前價格</p>
            <p class="text-sm font-mono text-primary">${currentPrice.toLocaleString()}</p>
         </div>
         <div class="bg-background-dark/50 rounded-lg p-3 border border-surface-border/50">
            <p class="text-[10px] text-slate-500 uppercase mb-1">持有數量</p>
            <p class="text-sm font-mono text-slate-200">${p.size.toFixed(5)}</p>
         </div>
         <div class="bg-background-dark/50 rounded-lg p-3 border border-surface-border/50">
            <p class="text-[10px] text-slate-500 uppercase mb-1">進場時間</p>
            <p class="text-[10px] text-slate-400">${new Date(p.entry_date).toLocaleDateString()}</p>
         </div>
      </div>

      <button class="close-pos-btn w-full py-2.5 rounded-lg bg-danger/10 text-danger border border-danger/20 hover:bg-danger hover:text-white transition-all text-sm font-bold flex items-center justify-center gap-2" data-id="${p.id}">
         <span class="material-symbols-outlined text-sm">cancel</span>
         手動平倉 (Market Sell)
      </button>
    `;
        container.appendChild(card);
    });

    document.querySelectorAll(".close-pos-btn").forEach(btn => {
        btn.addEventListener("click", () => closePosition(btn.dataset.id));
    });
}

async function closePosition(id) {
    if (!confirm("確定要手動平倉嗎？這將會以市價賣出所有持倉。")) return;
    try {
        await api(`/api/admin/positions/${id}/close`, { method: "POST" });
        setAlert("平倉成功", "success");
        refresh();
    } catch (err) {
        setAlert(err.message, "error");
    }
}

function renderTrades() {
    const tbody = el("tradesTableBody");
    const empty = el("emptyTrades");
    if (!tbody) return;
    tbody.innerHTML = "";

    if (state.trades.length === 0) {
        if (empty) empty.classList.remove("hidden");
        return;
    }
    if (empty) empty.classList.add("hidden");

    state.trades.forEach((t) => {
        const isBuy = t.side === "buy";
        const pnl = t.pnl_usdt || 0;
        const pnlPct = t.pnl_pct ? (t.pnl_pct * 100) : 0;

        const tr = document.createElement("tr");
        tr.className = "hover:bg-white/5 transition-colors";
        tr.innerHTML = `
      <td class="px-6 py-4">
        <div class="flex items-center gap-3">
          <span class="material-symbols-outlined text-sm ${isBuy ? 'text-success' : 'text-danger'}">
            ${isBuy ? 'south_east' : 'north_east'}
          </span>
          <div>
            <div class="font-bold text-slate-200 tracking-tight">${isBuy ? 'BUY' : 'SELL'}</div>
            <div class="text-[10px] text-slate-500 font-mono">${new Date(t.created_at).toLocaleString()}</div>
          </div>
        </div>
      </td>
      <td class="px-6 py-4 font-mono text-xs text-slate-400">${t.strategy_id.split('-')[0]}</td>
      <td class="px-6 py-4 font-mono">
        <div class="text-slate-300">${t.entry_price.toFixed(2)}</div>
        <div class="text-[10px] text-slate-500">${t.exit_price ? t.exit_price.toFixed(2) : '-'}</div>
      </td>
      <td class="px-6 py-4">
        ${!isBuy && t.exit_price ? `
          <div class="font-bold ${pnl >= 0 ? 'text-success' : 'text-danger'}">${pnl >= 0 ? '+' : ''}${pnl.toFixed(2)}</div>
          <div class="text-[10px] ${pnl >= 0 ? 'text-success/70' : 'text-danger/70'}">${pnl >= 0 ? '+' : ''}${pnlPct.toFixed(2)}%</div>
        ` : '-'}
      </td>
      <td class="px-6 py-4 text-xs text-slate-400 max-w-[200px] truncate" title="${t.reason || ''}">${t.reason || '-'}</td>
    `;
        tbody.appendChild(tr);
    });
}

function refresh() {
    fetchPositions();
    fetchTrades();
}

function bootstrap() {
    updateExchangeLink();
    initSidebar();
    initBinanceConfigModal();

    window.onBinanceConfigUpdate = () => {
        refresh();
    };

    if (!state.token) {
        window.location.href = "/";
        return;
    }

    // Initialize Global Environment Selectors
    initGlobalEnvSelector((env) => {
        state.env = env;
        refresh();
    });

    const refreshPositionsBtn = el("refreshPositionsBtn");
    if (refreshPositionsBtn) refreshPositionsBtn.addEventListener("click", fetchPositions);

    const refreshTradesBtn = el("refreshTradesBtn");
    if (refreshTradesBtn) refreshTradesBtn.addEventListener("click", fetchTrades);

    const manualBuyBtn = el("manualBuyBtn");
    if (manualBuyBtn) {
        manualBuyBtn.addEventListener("click", async () => {
            const symbol = el("manualSymbol").value.trim().toUpperCase();
            const amount = parseFloat(el("manualAmount").value);
            if (!symbol || isNaN(amount) || amount <= 0) {
                setAlert("請輸入正確的交易對與金額", "error");
                return;
            }

            if (!confirm(`確定要手動買入 ${amount} USDT 的 ${symbol} (${state.env}) 嗎？`)) return;

            manualBuyBtn.disabled = true;
            manualBuyBtn.textContent = "執行中...";
            try {
                await api("/api/admin/trades/manual-buy", {
                    method: "POST",
                    body: { symbol, amount, env: state.env }
                });
                setAlert("手動買入成功", "success");
                refresh();
            } catch (err) {
                setAlert(err.message, "error");
            } finally {
                manualBuyBtn.disabled = false;
                manualBuyBtn.textContent = "立即買入 (Market Buy)";
            }
        });
    }

    const logoutBtn = el("logoutBtn");
    if (logoutBtn) {
        logoutBtn.classList.remove("hidden");
        logoutBtn.addEventListener("click", () => {
            localStorage.removeItem("aat_token");
            window.location.href = "/";
        });
    }

    refresh();
    setInterval(refresh, 30000); // Auto refresh every 30s
}

bootstrap();
