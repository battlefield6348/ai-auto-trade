import { updateExchangeLink, initSidebar, initBinanceConfigModal, initAuthModal, initGlobalEnvSelector, handleUnauthorized, apiFetch, showMessage } from "./common.js";

const state = {
  token: localStorage.getItem("aat_token") || "",
  role: "",
  scoreChart: null,
};

const el = (id) => document.getElementById(id);

function setupSliders() {
  const sliders = [
    { id: 'maWeightSlider', valId: 'maWeightVal' },
    { id: 'volWeightSlider', valId: 'volWeightVal' },
    { id: 'rsiWeightSlider', valId: 'rsiWeightVal' }
  ];

  sliders.forEach(s => {
    const slider = el(s.id);
    const valDisp = el(s.valId);
    if (slider && valDisp) {
      slider.addEventListener('input', () => {
        valDisp.textContent = `${slider.value}%`;
      });
    }
  });
}

function readForm() {
  const horizons = [3, 5, 10]; // Fixed for now based on UI

  // Sliders
  const maWeight = parseInt(el('maWeightSlider').value);
  const volWeight = parseInt(el('volWeightSlider').value);
  const rsiWeight = parseInt(el('rsiWeightSlider').value);

  // Inputs
  const tp = parseFloat(el('tpInput').value) / 100 || 0.125;
  const sl = parseFloat(el('slInput').value) / 100 || 0.03;

  // Build standard request payload compatible with backend
  return {
    symbol: "BTCUSDT",
    start_date: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0], // Last 30 days
    end_date: new Date().toISOString().split('T')[0],
    entry: {
      weights: {
        score: rsiWeight, // Map RSI to base score weight for now
        ma_bonus: maWeight,
        volume_bonus: volWeight,
      },
      thresholds: {
        ma_gap_min: 0.01,
        volume_ratio_min: 1.5,
      },
      flags: {
        use_ma: true,
        use_volume: true,
      },
      total_min: 60
    },
    exit: {
      // Using TP/SL on the UI side translates to specific rules for the backend
      weights: { score: 100 },
      thresholds: { total_min: 40 },
      flags: {},
      total_min: 40
    },
    horizons: horizons,
    timeframe: "1h" // Switched to 1h based on UI screenshot
  };
}

async function runBacktest() {
  const payload = readForm();
  const btn = el('runBacktestBtn');
  btn.disabled = true;
  btn.classList.add('opacity-50');

  try {
    const res = await apiFetch("/analysis/backtest", {
      method: "POST",
      body: JSON.stringify(payload)
    });

    if (res.success) {
      renderResult(res.result || res.data);
      showMessage("回測執行完成", "success");
    }
  } catch (err) {
    showMessage(err.message, "danger");
  } finally {
    btn.disabled = false;
    btn.classList.remove('opacity-50');
  }
}

function renderResult(res) {
  if (!res) return;

  // Update Summary
  el('summaryPnl').textContent = `${(res.summary?.total_return || 0).toFixed(2)}%`;
  el('summaryMdd').textContent = `${(res.summary?.mdd || 0).toFixed(2)}%`;
  el('summaryWinRate').textContent = `${(res.summary?.win_rate || 0).toFixed(1)}%`;

  // Render Table
  const tbody = el('tradeLogsBody');
  tbody.innerHTML = "";
  const trades = res.trades || [];

  if (trades.length === 0) {
    tbody.innerHTML = '<tr><td colspan="7" class="px-8 py-20 text-center text-slate-600 italic">無交易紀錄</td></tr>';
  } else {
    trades.forEach(t => {
      const tr = document.createElement('tr');
      tr.className = "border-b border-surface-border/20 hover:bg-white/5 transition-all group";
      tr.innerHTML = `
                <td class="px-8 py-5 text-slate-400 font-mono">${t.entry_date}</td>
                <td class="px-6 py-5 text-center"><span class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded">LONG</span></td>
                <td class="px-6 py-5 text-center font-mono text-white">${t.entry_price.toLocaleString()}</td>
                <td class="px-6 py-5 text-center">
                    <div class="w-full bg-slate-800 h-1 rounded-full overflow-hidden mb-1">
                        <div class="bg-secondary h-full" style="width: 85%"></div>
                    </div>
                </td>
                <td class="px-6 py-5 text-center"><span class="px-2 py-0.5 bg-primary/10 text-primary text-[10px] font-bold rounded uppercase">${t.reason || 'Take Profit'}</span></td>
                <td class="px-6 py-5 text-center font-mono text-white">${t.exit_price.toLocaleString()}</td>
                <td class="px-8 py-5 text-right font-black ${t.pnl_pct >= 0 ? 'text-success' : 'text-danger'}">${(t.pnl_pct * 100).toFixed(2)}%</td>
            `;
      tbody.appendChild(tr);
    });
  }

  renderChart(res.events || []);
  renderHorizonBars(res.stats || {});
}

function renderHorizonBars(stats) {
  // Expected horizons from UI: 3, 5, 10
  const horizons = [3, 5, 10];
  horizons.forEach(h => {
    const key = `d${h}`;
    const data = stats[key];
    const barContainer = document.querySelector(`.flex-1.flex.flex-col.items-center.gap-2:nth-child(${horizons.indexOf(h) + 1})`);

    if (barContainer && data) {
      const innerBar = barContainer.querySelector('.rounded-t-lg div');
      const valLabel = barContainer.querySelector('span:last-child');

      if (innerBar) {
        // Scale height based on return (clamped for visual)
        const height = Math.min(100, Math.max(10, (data.AvgReturn * 1000) + 30));
        innerBar.style.height = `${height}%`;
        innerBar.className = `absolute bottom-0 w-full rounded-t-lg ${data.AvgReturn >= 0 ? 'bg-primary' : 'bg-danger'}`;
      }
      if (valLabel) {
        valLabel.textContent = `${data.AvgReturn >= 0 ? '+' : ''}${(data.AvgReturn * 100).toFixed(1)}%`;
        valLabel.className = `text-[8px] font-bold ${data.AvgReturn >= 0 ? 'text-primary' : 'text-danger'}`;
      }
    }
  });
}

function renderChart(events) {
  const ctx = el('btScoreChart').getContext('2d');
  if (state.scoreChart) state.scoreChart.destroy();

  const sorted = [...events].sort((a, b) => a.trade_date.localeCompare(b.trade_date));
  const labels = sorted.map(e => e.trade_date.split(' ')[0]);
  const prices = sorted.map(e => e.close_price);
  const scores = sorted.map(e => e.total_score);

  state.scoreChart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: labels,
      datasets: [
        {
          label: 'BTC 價格',
          data: prices,
          borderColor: '#0ddff2',
          borderWidth: 2,
          pointRadius: 0,
          tension: 0.4,
          yAxisID: 'y',
        },
        {
          label: 'AI 置信度',
          data: scores,
          borderColor: '#7c3aed',
          borderWidth: 1,
          borderDash: [5, 5],
          pointRadius: 0,
          tension: 0.4,
          yAxisID: 'y1',
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        y: {
          position: 'left',
          grid: { color: 'rgba(255,255,255,0.05)' },
          ticks: { color: '#64748b', font: { size: 10 } }
        },
        y1: {
          position: 'right',
          min: 0,
          max: 100,
          grid: { display: false },
          ticks: { color: '#7c3aed', font: { size: 10 } }
        },
        x: {
          grid: { display: false },
          ticks: { color: '#64748b', font: { size: 9 }, maxRotation: 0 }
        }
      },
      plugins: {
        legend: { display: false }
      }
    }
  });
}

function bootstrap() {
  initSidebar();
  updateExchangeLink();
  setupSliders();

  el('runBacktestBtn').addEventListener('click', runBacktest);

  // Initial Backtest
  setTimeout(runBacktest, 500);
}

bootstrap();
