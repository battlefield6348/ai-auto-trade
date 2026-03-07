import { initSidebar, initGlobalEnvSelector, apiFetch, showMessage, formatTime } from './common.js';

document.addEventListener('DOMContentLoaded', async () => {
  initSidebar();
  setupClock();

  initGlobalEnvSelector((env) => {
    updateDashboard();
  });

  await updateDashboard();
  setInterval(updateDashboard, 60000); // Update every minute
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

async function updateDashboard() {
  try {
    // Fetch user info from localStorage if present
    const email = localStorage.getItem('aat_email');
    if (email) {
      const loginStatus = document.getElementById('loginStatus');
      const roleLabel = document.getElementById('roleLabel');
      if (loginStatus) loginStatus.textContent = email;
      if (roleLabel) roleLabel.textContent = "ADMIN";
    }

    // Example AI Score update
    const score = 82;
    updateGauge(score);

    // In a real app, fetch /api/admin/jobs/history to populate the activity list
    // const history = await apiFetch('/admin/jobs/history');
    // renderActivity(history);

  } catch (err) {
    console.error('Dashboard update failed:', err);
  }
}

function updateGauge(score) {
  const gauge = document.getElementById('scoreGauge');
  const scoreVal = document.getElementById('aiScore');
  if (!gauge) return;

  const circumference = 628.3;
  const offset = circumference - (score / 100) * circumference;
  gauge.style.strokeDashoffset = offset;
  if (scoreVal) scoreVal.textContent = score;
}
