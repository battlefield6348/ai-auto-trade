/**
 * Common utilities for AI Auto Trade Web Console
 */

// 處理登入逾時或權限不足
function handleUnauthorized() {
    console.warn("[Auth] Unauthorized or session expired, redirecting to login...");
    localStorage.removeItem("aat_token");
    localStorage.removeItem("aat_email");
    localStorage.removeItem("aat_user");

    // Use replace to avoid back-button loop from login page
    window.location.replace("/login.html");
}

function guardRoute() {
    // Check if current page is login.html to avoid infinite redirect
    if (window.location.pathname.endsWith('/login.html')) {
        return;
    }

    const token = localStorage.getItem("aat_token");
    if (!token) {
        handleUnauthorized();
        return true;
    }
    return false;
}

async function updateExchangeLink() {
    const link = document.getElementById('exchangeLink');
    if (!link) return;

    // Show it immediately to avoid "missing" button issues
    link.classList.remove('hidden');

    try {
        const res = await fetch('/api/health');
        const data = await res.json();

        if (data.use_testnet) {
            link.href = 'https://testnet.binance.vision/';
            link.classList.remove('bg-warning/20', 'text-warning', 'border-warning/40', 'hover:bg-warning/30');
            link.classList.add('bg-secondary/10', 'text-secondary', 'border-secondary/20', 'hover:bg-secondary/20');
        } else if (data.active_env === 'paper') {
            link.href = 'https://www.binance.com/zh-TW/trade/BTC_USDT?type=spot';
            link.classList.remove('bg-secondary/20', 'text-secondary', 'border-secondary/40', 'hover:bg-secondary/30', 'bg-warning/20', 'text-warning', 'border-warning/40');
            link.classList.add('bg-primary/10', 'text-primary', 'border-primary/20', 'hover:bg-primary/20');
        } else {
            link.href = 'https://www.binance.com/zh-TW/trade/BTC_USDT?type=spot';
            link.classList.remove('bg-secondary/20', 'text-secondary', 'border-secondary/40', 'hover:bg-secondary/30', 'bg-primary/20', 'text-primary', 'border-primary/40');
            link.classList.add('bg-warning/10', 'text-warning', 'border-warning/20', 'hover:bg-warning/20');
        }
    } catch (err) {
        console.error('Failed to update exchange link:', err);
    }
}

function initSidebar() {
    const sidebar = document.getElementById('sidebar');
    const toggle = document.getElementById('sidebarToggle');
    const mobileToggle = document.getElementById('mobileToggle');
    const content = document.getElementById('main-content');
    const toggleIcon = document.getElementById('sidebarToggleIcon');
    const overlay = document.getElementById('content-overlay');

    if (!sidebar || !content) return;

    const isCollapsed = localStorage.getItem('sidebar-collapsed') === 'true';

    function applyState(collapsed) {
        if (collapsed) {
            sidebar.classList.add('sidebar-collapsed');
            content.classList.add('content-collapsed-padding');
            content.classList.remove('lg:pl-64');
            if (toggleIcon) toggleIcon.textContent = 'menu';
        } else {
            sidebar.classList.remove('sidebar-collapsed');
            content.classList.remove('content-collapsed-padding');
            content.classList.add('lg:pl-64');
            if (toggleIcon) toggleIcon.textContent = 'menu_open';
        }
    }

    applyState(isCollapsed);

    toggle?.addEventListener('click', () => {
        const newState = !sidebar.classList.contains('sidebar-collapsed');
        applyState(newState);
        localStorage.setItem('sidebar-collapsed', newState);
    });

    mobileToggle?.addEventListener('click', () => {
        sidebar.classList.remove('-translate-x-full');
        sidebar.classList.add('translate-x-0');
        overlay?.classList.remove('hidden');
    });

    overlay?.addEventListener('click', () => {
        sidebar.classList.add('-translate-x-full');
        sidebar.classList.remove('translate-x-0');
        overlay.classList.add('hidden');
    });

    // Start Server Clock if element exists
    const clock = document.getElementById('serverClock') || document.getElementById('clock');
    if (clock) {
        const tick = () => {
            clock.textContent = new Date().toLocaleTimeString('zh-TW', { hour12: false });
        };
        tick();
        setInterval(tick, 1000);
    }

    // Auth State UI Update
    const email = localStorage.getItem("aat_email");
    const logoutBtn = document.getElementById("logoutBtn");
    const loginStatus = document.getElementById("loginStatus");
    const roleLabel = document.getElementById("roleLabel");

    if (email) {
        if (logoutBtn) logoutBtn.classList.remove("hidden");
        if (loginStatus) loginStatus.textContent = email;
        if (roleLabel) roleLabel.textContent = "ADMIN";
    }

    if (logoutBtn) {
        logoutBtn.onclick = () => {
            localStorage.removeItem("aat_token");
            localStorage.removeItem("aat_email");
            window.location.href = "/login.html";
        };
    }
}

async function initGlobalEnvSelector(onEnvChange) {
    const envSelectors = document.querySelectorAll(".env-selector");
    if (envSelectors.length === 0) return;

    const updateEnvUI = (activeEnv) => {
        let env = activeEnv;
        if (env === 'prod') env = 'real';

        localStorage.setItem('aat_env', env);

        const currentSelectors = document.querySelectorAll(".env-selector");
        currentSelectors.forEach(btn => {
            const btnEnv = btn.dataset.env;
            const isMatch = (btnEnv === env);

            btn.classList.remove(
                "bg-white", "bg-secondary", "bg-primary", "bg-warning",
                "text-background-dark", "text-white", "shadow-sm", "shadow-neon-glow",
                "shadow-neon-glow-cyan", "shadow-neon-glow-warning", "ring-2", "ring-white/20",
                "border-primary", "bg-primary/5"
            );
            btn.classList.add("text-slate-500", "hover:border-white/20");

            if (isMatch) {
                btn.classList.remove("text-slate-500");
                if (btnEnv === 'test') {
                    btn.classList.add("bg-secondary", "text-white", "ring-1", "ring-white/30");
                } else if (btnEnv === 'paper') {
                    btn.classList.add("bg-primary", "text-background-dark", "ring-1", "ring-primary/50");
                } else if (btnEnv === 'real') {
                    btn.classList.add("bg-warning", "text-background-dark", "ring-1", "ring-warning/50");
                }

                // Special case for cards in settings.html
                if (btn.id && btn.id.startsWith('card-')) {
                    btn.classList.add('border-primary', 'bg-primary/5');
                }
            }
        });

        // Update currentEnvStatus text if it exists (for settings.html)
        const statusText = document.getElementById('currentEnvStatus');
        if (statusText) {
            const names = { 'test': '測試網 (Testnet)', 'paper': '模擬交易 (Paper)', 'real': '實時交易 (Live)' };
            statusText.textContent = names[env] || env;
        }
    };

    const setBackendEnv = async (env) => {
        const backendEnv = env === 'real' ? 'prod' : env;
        updateEnvUI(backendEnv);

        try {
            await apiFetch('/admin/binance/config', {
                method: 'POST',
                body: JSON.stringify({ active_env: backendEnv })
            });
            updateExchangeLink();
            if (onEnvChange) onEnvChange(backendEnv);
        } catch (err) {
            console.error("Failed to set env:", err);
        }
    };

    envSelectors.forEach(btn => {
        btn.addEventListener("click", () => setBackendEnv(btn.dataset.env));
    });

    // Sync with backend on load
    try {
        const config = await apiFetch('/admin/binance/config');
        if (config.success) {
            updateEnvUI(config.active_env);
            if (onEnvChange) onEnvChange(config.active_env);
        }
    } catch (err) {
        console.error("Failed to sync env:", err);
    }
}

async function apiFetch(path, options = {}) {
    const token = localStorage.getItem("aat_token");
    const headers = options.headers || {};
    if (token) headers["Authorization"] = `Bearer ${token}`;

    let body = options.body;
    if (body && typeof body === 'object' && !(body instanceof FormData)) {
        headers["Content-Type"] = "application/json";
        body = JSON.stringify(body);
    }

    const fullPath = path.startsWith('/api') ? path : `/api${path}`;
    const response = await fetch(fullPath, {
        ...options,
        headers,
        body
    });

    if (response.status === 401) {
        handleUnauthorized();
        throw new Error("Unauthorized");
    }

    return response.json();
}

function showMessage(msg, type = 'info') {
    const alertEl = document.getElementById('alert');
    if (!alertEl) {
        // Fallback to alert if no element
        console.log(`[${type}] ${msg}`);
        return;
    }

    alertEl.textContent = msg;
    alertEl.classList.remove('hidden', 'bg-primary/10', 'text-primary', 'bg-danger/10', 'text-danger', 'bg-success/10', 'text-success', 'border-primary/30', 'border-danger/30', 'border-success/30');

    if (type === 'danger' || type === 'error') {
        alertEl.classList.add('bg-danger/10', 'text-danger', 'border-danger/30');
    } else if (type === 'success') {
        alertEl.classList.add('bg-success/10', 'text-success', 'border-success/30');
    } else {
        alertEl.classList.add('bg-primary/10', 'text-primary', 'border-primary/30');
    }

    alertEl.classList.remove('hidden');
    setTimeout(() => alertEl.classList.add('hidden'), 5000);
}

function formatTime(dateStr) {
    const d = new Date(dateStr);
    return d.toLocaleTimeString('zh-TW', { hour12: false });
}

// 統一導出所有工具函數
export {
    handleUnauthorized,
    guardRoute,
    updateExchangeLink,
    initSidebar,
    initGlobalEnvSelector,
    apiFetch,
    showMessage,
    formatTime
};
