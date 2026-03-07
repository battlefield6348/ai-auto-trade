import { initSidebar, initGlobalEnvSelector, apiFetch, showMessage, formatTime } from './common.js';

document.addEventListener('DOMContentLoaded', async () => {
  initSidebar();
  setupClock();

  initGlobalEnvSelector((env) => {
    updateDashboard();
  });

  await updateDashboard();
  // Update price and stats more frequently
  setInterval(updatePrice, 10000);
  setInterval(updateDashboard, 30000);

  // Start countdown timer
  startNextRunCountdown();
  setupEventListeners();
});

function setupClock() {
  const clock = document.getElementById('serverClock');
  if (!clock) return;
  setInterval(() => {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    const seconds = String(now.getSeconds()).padStart(2, '0');
    clock.textContent = `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
  }, 1000);
}

function setupEventListeners() {
  const startBtn = document.getElementById('startEngineBtn');
  if (startBtn) {
    startBtn.onclick = async () => {
      startBtn.disabled = true;
      const originalText = startBtn.innerHTML;
      startBtn.innerHTML = `<span class="material-symbols-outlined text-sm animate-spin">sync</span> 執行中...`;

      try {
        const res = await apiFetch('/admin/ingestion/daily', {
          method: 'POST',
          body: JSON.stringify({ run_analysis: true })
        });

        if (res.success) {
          showMessage("手動分析任務啟動成功", "success");
          updateDashboard();
        } else {
          showMessage(res.error || "任務啟動失敗", "error");
        }
      } catch (err) {
        showMessage("連線錯誤", "error");
      } finally {
        startBtn.disabled = false;
        startBtn.innerHTML = originalText;
      }
    };
  }
}

async function updatePrice() {
  try {
    const data = await apiFetch('/admin/binance/price?symbol=BTCUSDT');
    if (data.success) {
      const priceEl = document.getElementById('btcPrice');
      if (priceEl) {
        const priceValue = parseFloat(data.price);
        if (!isNaN(priceValue)) {
          const formatted = priceValue.toLocaleString('en-US', { minimumFractionDigits: 2 });
          priceEl.textContent = `$${formatted}`;
        }
      }
    }
  } catch (err) {
    console.error('Failed to fetch BTC price:', err);
  }
}

async function updateDashboard() {
  try {
    // 1. Update User Status
    const email = localStorage.getItem('aat_email');
    if (email) {
      const loginStatus = document.getElementById('loginStatus');
      const roleLabel = document.getElementById('roleLabel');
      if (loginStatus) loginStatus.textContent = email;
      if (roleLabel) roleLabel.textContent = "ADMIN";
    }

    // 2. Fetch AI Score & Analysis Summary
    try {
      const analysisData = await apiFetch('/analysis/summary');
      if (analysisData.success && analysisData.top_picks && analysisData.top_picks.length > 0) {
        const best = analysisData.top_picks[0];
        updateGauge(best.BaseScore || 82);

        const volRatioEl = document.getElementById('volRatio');
        if (volRatioEl) volRatioEl.innerHTML = `${(best.VolumeRatio || 1.0).toFixed(2)}x <span class="text-xs text-primary ml-1">↑</span>`;

        const volatilityEl = document.getElementById('volatility');
        if (volatilityEl) volatilityEl.innerHTML = `${(best.Amplitude * 100 || 0.42).toFixed(2)}% <span class="text-xs text-slate-500 ml-1">Stable</span>`;
      }
    } catch (e) { console.warn("Analysis summary not available yet"); }

    // 3. Fetch Job Stats & History
    try {
      const statusData = await apiFetch('/admin/jobs/status');
      const historyData = await apiFetch('/admin/jobs/history');

      if (statusData.success && historyData.success) {
        const history = historyData.data || [];

        let totalSucc = 0;
        let totalFail = 0;
        let totalLatency = 0;
        let latencyCount = 0;

        history.forEach(job => {
          totalSucc += (job.analysis_succ || 0);
          totalFail += (job.analysis_fail || 0);
          if (job.end && job.start) {
            const lat = new Date(job.end) - new Date(job.start);
            totalLatency += lat;
            latencyCount++;
          }
        });

        const succEl = document.getElementById('totalSuccess');
        const failEl = document.getElementById('totalFailure');
        const latEl = document.getElementById('lastLatency');

        if (succEl) succEl.textContent = totalSucc.toLocaleString();
        if (failEl) failEl.textContent = totalFail.toLocaleString();
        if (latEl && latencyCount > 0) {
          latEl.innerHTML = `${Math.round(totalLatency / latencyCount)} <span class="text-sm">ms</span>`;
        }

        renderJobHistory(history.slice(0, 5));

        if (statusData.last_auto_end) {
          window.lastAutoEnd = new Date(statusData.last_auto_end);
        }
      }
    } catch (e) { console.warn("Job status not available yet"); }

    updatePrice();
  } catch (err) {
    console.error('Dashboard update failed:', err);
  }
}

function renderJobHistory(jobs) {
  const container = document.getElementById('jobHistory');
  if (!container) return;

  container.innerHTML = jobs.map(job => {
    const time = job.start ? new Date(job.start).toLocaleTimeString('zh-TW', { hour12: false }) : '--:--';
    const isSuccess = job.analysis_ok && job.ingestion_ok;
    const statusClass = isSuccess ? 'bg-success/10 text-success border-success/20' : 'bg-danger/10 text-danger border-danger/20';
    const statusText = isSuccess ? 'Success' : 'Failed';

    return `
          <div class="flex items-center justify-between p-4 bg-background-dark/40 rounded-2xl border border-surface-border transition-all hover:border-primary/30 group">
            <div class="flex items-center gap-4">
              <span class="text-xs font-mono text-slate-500">${time}</span>
              <div>
                <h5 class="text-xs font-bold text-white group-hover:text-primary transition-colors">${job.kind || '遺漏任務'}</h5>
                <p class="text-[9px] text-slate-600 font-mono">${job.data_source || 'Unknown'}</p>
              </div>
            </div>
            <span class="px-2 py-0.5 rounded ${statusClass} text-[9px] font-bold">${statusText}</span>
          </div>
        `;
  }).join('');
}

function startNextRunCountdown() {
  const timerEl = document.getElementById('nextRunTimer');
  if (!timerEl) return;

  setInterval(() => {
    if (!window.lastAutoEnd) {
      timerEl.textContent = "-- : -- : --";
      return;
    }

    const intervalMs = 60 * 60 * 1000;
    const nextRun = new Date(window.lastAutoEnd.getTime() + intervalMs);
    const now = new Date();
    const diff = nextRun - now;

    if (diff <= 0) {
      timerEl.textContent = "00 : 00 : 00";
      return;
    }

    const h = Math.floor(diff / 3600000).toString().padStart(2, '0');
    const m = Math.floor((diff % 3600000) / 60000).toString().padStart(2, '0');
    const s = Math.floor((diff % 60000) / 1000).toString().padStart(2, '0');
    timerEl.textContent = `${h} : ${m} : ${s}`;

    const bar = timerEl.nextElementSibling?.firstElementChild;
    if (bar) {
      const pct = Math.max(0, Math.min(100, 100 - (diff / intervalMs) * 100));
      bar.style.width = `${pct}%`;
    }
  }, 1000);
}

function updateGauge(score) {
  const gauge = document.getElementById('scoreGauge');
  const scoreVal = document.getElementById('aiScore');
  const label = scoreVal?.nextElementSibling;

  if (!gauge) return;

  const circumference = 628.3;
  const offset = circumference - (score / 100) * circumference;
  gauge.style.strokeDashoffset = offset;

  if (scoreVal) scoreVal.textContent = Math.round(score);

  if (label) {
    if (score >= 80) { label.textContent = "極度看漲"; label.className = "text-xs font-bold text-primary tracking-widest mt-1"; }
    else if (score >= 60) { label.textContent = "看漲"; label.className = "text-xs font-bold text-success tracking-widest mt-1"; }
    else if (score >= 40) { label.textContent = "中立"; label.className = "text-xs font-bold text-slate-400 tracking-widest mt-1"; }
    else if (score >= 20) { label.textContent = "看跌"; label.className = "text-xs font-bold text-warning tracking-widest mt-1"; }
    else { label.textContent = "極度看跌"; label.className = "text-xs font-bold text-danger tracking-widest mt-1"; }
  }
}
