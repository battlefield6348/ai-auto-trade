/**
 * Common utilities for AI Auto Trade Web Console
 */

export async function updateExchangeLink() {
    const link = document.getElementById('exchangeLink');
    if (!link) return;

    // Show it immediately to avoid "missing" button issues
    link.classList.remove('hidden');

    try {
        const res = await fetch('/api/health');
        const data = await res.json();

        if (data.use_testnet) {
            link.href = 'https://demo.binance.com/zh-TW/spot';
            link.innerHTML = '<span class="material-symbols-outlined text-sm">science</span> 模擬交易 (Demo)';
            link.classList.remove('bg-warning/20', 'text-warning', 'border-warning/40', 'hover:bg-warning/30');
            link.classList.add('bg-secondary/20', 'text-secondary', 'border-secondary/40', 'hover:bg-secondary/30');
        } else {
            link.href = 'https://www.binance.com/zh-TW/trade/BTC_USDT?type=spot';
            link.innerHTML = '<span class="material-symbols-outlined text-sm">currency_exchange</span> 正式交易所';
            link.classList.remove('bg-secondary/20', 'text-secondary', 'border-secondary/40', 'hover:bg-secondary/30');
            link.classList.add('bg-warning/20', 'text-warning', 'border-warning/40', 'hover:bg-warning/30');
        }
    } catch (err) {
        console.error('Failed to update exchange link:', err);
        // Fallback or keep current state
    }
}
