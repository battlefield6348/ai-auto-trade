/**
 * Common utilities for AI Auto Trade Web Console
 */

export function guardRoute() {
    // Check if current page is login.html to avoid infinite redirect
    if (window.location.pathname.endsWith('/login.html')) {
        return;
    }

    const token = localStorage.getItem("aat_token");
    if (!token) {
        console.warn("[Auth] No token found, redirecting to login...");
        window.location.href = "/login.html";
        return true;
    }
    return false;
}

export async function updateExchangeLink() {
    const link = document.getElementById('exchangeLink');
    if (!link) return;

    // Show it immediately to avoid "missing" button issues
    link.classList.remove('hidden');

    try {
        const res = await fetch('/api/health');
        const data = await res.json();

        if (data.use_testnet) {
            link.href = 'https://testnet.binance.vision/';
            link.innerHTML = '<span class="material-symbols-outlined text-sm">science</span> 測試帳戶 (Testnet)';
            link.classList.remove('bg-warning/20', 'text-warning', 'border-warning/40', 'hover:bg-warning/30');
            link.classList.add('bg-secondary/20', 'text-secondary', 'border-secondary/40', 'hover:bg-secondary/30');
        } else if (data.active_env === 'paper') {
            link.href = 'https://www.binance.com/zh-TW/trade/BTC_USDT?type=spot';
            link.innerHTML = '<span class="material-symbols-outlined text-sm">description</span> 模擬實盤 (Paper Mode)';
            link.classList.remove('bg-secondary/20', 'text-secondary', 'border-secondary/40', 'hover:bg-secondary/30', 'bg-warning/20', 'text-warning', 'border-warning/40');
            link.classList.add('bg-primary/20', 'text-primary', 'border-primary/40', 'hover:bg-primary/30');
        } else {
            link.href = 'https://www.binance.com/zh-TW/trade/BTC_USDT?type=spot';
            link.innerHTML = '<span class="material-symbols-outlined text-sm">currency_exchange</span> 正式交易所';
            link.classList.remove('bg-secondary/20', 'text-secondary', 'border-secondary/40', 'hover:bg-secondary/30', 'bg-primary/20', 'text-primary', 'border-primary/40');
            link.classList.add('bg-warning/20', 'text-warning', 'border-warning/40', 'hover:bg-warning/30');
        }
    } catch (err) {
        console.error('Failed to update exchange link:', err);
    }
}

export async function initBinanceConfigModal() {
    // Deprecated: Environment configuration is now handled globally in the top bar.
    // This function is kept to avoid import errors until all references are removed.
    const openBtn = document.getElementById('binanceConfigBtn');
    if (openBtn) openBtn.style.display = 'none'; // Hide the button instead
}

export function initSidebar() {
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
    const token = localStorage.getItem("aat_token");
    const email = localStorage.getItem("aat_email");
    const loginBtn = document.getElementById("loginBtn");
    const logoutBtn = document.getElementById("logoutBtn");
    const loginStatus = document.getElementById("loginStatus");

    if (token) {
        if (loginBtn) loginBtn.classList.add("hidden");
        if (logoutBtn) logoutBtn.classList.remove("hidden");
        if (loginStatus) loginStatus.textContent = email || "已登入";
    } else {
        if (loginBtn) loginBtn.classList.remove("hidden");
        if (logoutBtn) logoutBtn.classList.add("hidden");
        if (loginStatus) loginStatus.textContent = "未登入";
    }
}

export function initAuthModal(onSuccess) {
    const dialog = document.getElementById("loginModal");
    const openBtn = document.getElementById("loginBtn");
    const closeBtn = document.getElementById("closeAuth");
    const authForm = document.getElementById("authForm");
    const toggleMode = document.getElementById("toggleAuthMode");
    const nameField = document.getElementById("nameField");
    const authTitle = document.getElementById("authTitle");
    const authSubmit = document.getElementById("authSubmit");

    if (!dialog || !authForm) return;

    let isRegister = false;

    if (openBtn) openBtn.onclick = () => dialog.showModal();
    if (closeBtn) closeBtn.onclick = () => dialog.close();

    if (toggleMode) {
        toggleMode.onclick = (e) => {
            e.preventDefault();
            isRegister = !isRegister;
            if (isRegister) {
                authTitle.textContent = "建立新帳號 (Register)";
                nameField.classList.remove("hidden");
                authSubmit.textContent = "註冊並登入";
                toggleMode.textContent = "已有帳號？點此登入 (Login)";
            } else {
                authTitle.textContent = "系統登入 (Login)";
                nameField.classList.add("hidden");
                authSubmit.textContent = "Verify";
                toggleMode.textContent = "還沒有帳號？點此註冊 (Register)";
            }
        };
    }

    authForm.onsubmit = async (e) => {
        e.preventDefault();
        const email = document.getElementById("authEmail").value;
        const password = document.getElementById("authPassword").value;
        const name = document.getElementById("authName")?.value;

        try {
            if (isRegister) {
                const regRes = await fetch("/api/auth/register", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ email, password, name }),
                });
                const regData = await regRes.json();
                if (!regRes.ok) throw new Error(regData.message || regData.error || "註冊失敗");
            }

            // 無論是登入還是註冊後，都執行登入
            const res = await fetch("/api/auth/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email, password }),
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.message || data.error || "登入失敗");

            localStorage.setItem("aat_token", data.access_token);
            localStorage.setItem("aat_email", email);
            dialog.close();
            if (onSuccess) onSuccess(data);
        } catch (err) {
            alert(err.message);
        }
    };

    const logoutBtn = document.getElementById("logoutBtn");
    if (logoutBtn) {
        logoutBtn.onclick = () => {
            localStorage.removeItem("aat_token");
            localStorage.removeItem("aat_email");
            window.location.href = "/login.html";
        };
    }
}

export async function initGlobalEnvSelector(onEnvChange) {
    const envSelectors = document.querySelectorAll(".env-selector");
    if (envSelectors.length === 0) return;

    const token = localStorage.getItem("aat_token");
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers["Authorization"] = `Bearer ${token}`;

    const updateEnvUI = (activeEnv) => {
        let env = activeEnv;
        if (env === 'prod') env = 'real';

        envSelectors.forEach(btn => {
            const btnEnv = btn.dataset.env;
            const isMatch = (btnEnv === env) ||
                (env === 'prod' && btnEnv === 'real') ||
                (env === 'real' && btnEnv === 'prod');

            if (isMatch) {
                btn.classList.add("bg-primary", "text-background-dark");
                btn.classList.remove("text-slate-500", "hover:bg-white/5");
            } else {
                btn.classList.remove("bg-primary", "text-background-dark");
                btn.classList.add("text-slate-500", "hover:bg-white/5");
            }
        });
    };

    const setBackendEnv = async (env) => {
        const backendEnv = env === 'real' ? 'prod' : env;

        // Optimistic UI update
        updateEnvUI(backendEnv);

        try {
            await fetch('/api/admin/binance/config', {
                method: 'POST',
                headers: headers,
                body: JSON.stringify({ active_env: backendEnv })
            });
            updateExchangeLink();
            if (onEnvChange) onEnvChange(backendEnv);
        } catch (err) {
            console.error("Failed to set env:", err);
            // Revert UI if needed? Or just show error
        }
    };

    envSelectors.forEach(btn => {
        // Remove old listeners to be safe (though this runs once usually)
        const newBtn = btn.cloneNode(true);
        btn.parentNode.replaceChild(newBtn, btn);
        newBtn.addEventListener("click", () => setBackendEnv(newBtn.dataset.env));
    });

    // Re-select because we replaced nodes
    const refreshedSelectors = document.querySelectorAll(".env-selector");

    // Sync with backend on load
    try {
        const res = await fetch('/api/admin/binance/config', {
            headers: token ? { "Authorization": `Bearer ${token}` } : {}
        });
        const data = await res.json();
        if (data.success) {
            updateEnvUI(data.active_env);
            if (onEnvChange) onEnvChange(data.active_env);
        }
    } catch (err) {
        console.error("Failed to sync env:", err);
    }
}
