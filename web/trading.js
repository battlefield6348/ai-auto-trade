import { updateExchangeLink, initSidebar, initBinanceConfigModal, initGlobalEnvSelector, handleUnauthorized, apiFetch, showMessage } from "./common.js";

const state = {
    positions: [],
    trades: [],
    prices: {},
    env: "test"
};

const el = (id) => document.getElementById(id);

async function fetchPrices(symbols) {
    for (const s of symbols) {
        try {
            const data = await apiFetch(`/admin/binance/price?symbol=${s}`);
            state.prices[s] = data.price;
        } catch (err) {
            console.error(`Failed to fetch price for ${s}:`, err);
        }
    }
}

async function refreshTradingData() {
    try {
        const [posData, tradeData] = await Promise.all([
            apiFetch(`/admin/positions?env=${state.env}`),
            apiFetch(`/admin/trades?env=${state.env}`)
        ]);

        state.positions = posData.positions || [];
        state.trades = (tradeData.trades || []).sort((a, b) => new Date(b.created_at) - new Date(a.created_at));

        const symbols = [...new Set(state.positions.map(p => p.symbol))];
        if (symbols.length > 0) await fetchPrices(symbols);

        renderUI();
        el('lastUpdated').textContent = new Date().toLocaleTimeString('zh-TW', { hour12: false });
    } catch (err) {
        console.error("Failed to refresh trading data:", err);
    }
}

function renderUI() {
    renderStats();
    renderPositions();
    renderLogs();
}

function renderStats() {
    let totalUnrealized = 0;
    state.positions.forEach(p => {
        const current = state.prices[p.symbol] || p.entry_price;
        totalUnrealized += (current - p.entry_price) * p.size;
    });

    const unrealPnlEl = el('unrealizedPnl');
    unrealPnlEl.textContent = `${totalUnrealized >= 0 ? '+' : ''}${totalUnrealized.toFixed(2)}`;
    unrealPnlEl.className = `text-2xl font-black font-mono ${totalUnrealized >= 0 ? 'text-success' : 'text-danger'}`;
}

function renderPositions() {
    const tbody = el('positionsTableBody');
    if (!tbody) return;
    tbody.innerHTML = "";

    if (state.positions.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="px-8 py-20 text-center text-slate-600 italic">目前無持有倉位</td></tr>';
        return;
    }

    state.positions.forEach(p => {
        const current = state.prices[p.symbol] || p.entry_price;
        const pnl = (current - p.entry_price) * p.size;
        const pnlPct = ((current / p.entry_price) - 1) * 100;
        const isProfit = pnl >= 0;

        const tr = document.createElement('tr');
        tr.className = "hover:bg-white/5 transition-colors group";
        tr.innerHTML = `
            <td class="px-8 py-6">
                <div class="flex items-center gap-4">
                    <div class="size-10 rounded-full bg-slate-800 flex items-center justify-center font-bold text-[10px] text-white">${p.symbol.substring(0, 3)}</div>
                    <div>
                        <div class="text-sm font-black text-white">${p.symbol}</div>
                        <div class="text-[9px] text-slate-500 font-bold uppercase tracking-widest">${p.env}</div>
                    </div>
                </div>
            </td>
            <td class="px-6 py-6 text-center">
                <div class="text-[10px] text-white font-mono font-bold">${p.size.toFixed(4)} ${p.symbol.replace('USDT', '')}</div>
                <div class="text-[9px] text-success font-black mt-1">LONG 10x</div>
            </td>
            <td class="px-6 py-6 text-center font-mono text-[11px] text-slate-400">${p.entry_price.toLocaleString()}</td>
            <td class="px-6 py-6 text-center font-mono text-[11px] text-white font-bold">${current.toLocaleString()}</td>
            <td class="px-8 py-6 text-right">
                <div class="text-sm font-black ${isProfit ? 'text-success' : 'text-danger'}">${isProfit ? '+' : ''}${pnl.toFixed(2)}</div>
                <div class="text-[10px] font-bold ${isProfit ? 'text-success/70' : 'text-danger/70'}">(${isProfit ? '+' : ''}${pnlPct.toFixed(2)}%)</div>
            </td>
        `;
        tbody.appendChild(tr);
    });
}

function renderLogs() {
    const container = el('logsContainer');
    if (!container) return;
    container.innerHTML = "";

    if (state.trades.length === 0) {
        container.innerHTML = '<p class="text-[10px] text-slate-600 italic text-center py-10">尚無交易紀錄</p>';
        return;
    }

    state.trades.slice(0, 10).forEach(t => {
        const isBuy = t.side === 'buy';
        const item = document.createElement('div');
        item.className = "flex items-start gap-4 animate-in fade-in slide-in-from-right-2";
        item.innerHTML = `
            <div class="size-10 rounded-full ${isBuy ? 'bg-success/10 text-success border-success/20' : 'bg-danger/10 text-danger border-danger/20'} flex items-center justify-center border shrink-0">
                <span class="material-symbols-outlined text-lg">${isBuy ? 'shopping_cart' : 'sell'}</span>
            </div>
            <div class="flex-1 border-b border-surface-border/10 pb-4">
                <div class="flex justify-between mb-1">
                    <span class="text-[10px] text-white font-bold">${isBuy ? '買入' : '賣出'} ${t.symbol}</span>
                    <span class="text-[8px] text-slate-600 font-mono">${new Date(t.created_at).toLocaleTimeString('zh-TW', { hour12: false })}</span>
                </div>
                <p class="text-[10px] text-slate-500 leading-relaxed">成交價格: ${t.entry_price.toLocaleString()} ● 數量: ${t.amount}</p>
            </div>
        `;
        container.appendChild(item);
    });
}

async function panicSell() {
    if (!confirm("確定要執行緊急平倉嗎？這將會市價平掉所有當前持倉！")) return;

    try {
        // Since we might not have a dedicated panic endpoint, we loop through and close
        const promises = state.positions.map(p => apiFetch(`/admin/positions/${p.id}/close`, { method: "POST" }));
        await Promise.all(promises);
        showMessage("全數平倉指令已送出", "success");
        refreshTradingData();
    } catch (err) {
        showMessage(err.message, "danger");
    }
}

function bootstrap() {
    initSidebar();
    updateExchangeLink();
    initBinanceConfigModal();

    initGlobalEnvSelector((env) => {
        state.env = env;
        refreshTradingData();
    });

    el('panicSellBtn').addEventListener('click', panicSell);
    el('switchEnvBtn').addEventListener('click', () => {
        // Fallback to modal if no global selector visible
        const modal = el('binanceConfigModal');
        if (modal) modal.classList.remove('hidden');
    });

    refreshTradingData();
    setInterval(refreshTradingData, 10000);
}

bootstrap();
