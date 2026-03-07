import { apiFetch, showMessage, formatTime } from './common.js';

document.addEventListener('DOMContentLoaded', async () => {
  setupSidebar();
  setupClock();
  setupEventListeners();
  await updateDashboard();
  setInterval(updateDashboard, 60000); // Update every minute
});

function setupSidebar() {
  const sidebar = document.getElementById('sidebar');
  const toggle = document.getElementById('sidebarToggle');
  const toggleIcon = document.getElementById('sidebarToggleIcon');
  const mainContent = document.getElementById('main-content');

  if (toggle) {
    toggle.addEventListener('click', () => {
      sidebar.classList.toggle('sidebar-collapsed');
      mainContent.classList.toggle('lg:pl-64');
      mainContent.classList.toggle('lg:pl-[72px]');
      toggleIcon.textContent = sidebar.classList.contains('sidebar-collapsed') ? 'menu' : 'menu_open';
    });
  }
}

function setupClock() {
  const clock = document.getElementById('serverClock');
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
    startBtn.addEventListener('click', () => {
      showMessage('正在啟動交易核心引擎...', 'primary');
      // Logic for starting engine
    });
  }

  const envSelectors = document.querySelectorAll('.env-selector');
  envSelectors.forEach(btn => {
    btn.addEventListener('click', () => {
      envSelectors.forEach(b => {
        b.classList.remove('bg-primary', 'text-background-dark');
        b.classList.add('text-slate-400', 'hover:text-white');
      });
      btn.classList.add('bg-primary', 'text-background-dark');
      btn.classList.remove('text-slate-400', 'hover:text-white');
      const env = btn.getAttribute('data-env');
      localStorage.setItem('aat_env', env);
      updateDashboard();
    });
  });
}

async function updateDashboard() {
  try {
    // Update User Status
    const userJson = localStorage.getItem('aat_user');
    if (userJson) {
      const user = JSON.parse(userJson);
      document.getElementById('loginStatus').textContent = user.email;
      document.getElementById('roleLabel').textContent = user.role.toUpperCase();
      document.getElementById('logoutBtn').classList.remove('hidden');
    }

    // Fetch Real Market Data (Fallback to fake for demo UI)
    // In real app, we would fetch from /api/market/latest and /api/admin/analysis/status

    const score = 82; // Example score
    updateGauge(score);

    // Update other UI elements with dynamic data if available
    // ...
  } catch (err) {
    console.error('Dashboard update failed:', err);
  }
}

function updateGauge(score) {
  const gauge = document.getElementById('scoreGauge');
  const scoreVal = document.getElementById('aiScore');
  if (!gauge) return;

  // Circumference of circle with r=100 is 2 * PI * 100 = 628.3
  const circumference = 628.3;
  const offset = circumference - (score / 100) * circumference;
  gauge.style.strokeDashoffset = offset;
  scoreVal.textContent = score;
}
