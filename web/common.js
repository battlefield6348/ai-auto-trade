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
    const modal = document.getElementById('binanceConfigModal');
    const openBtn = document.getElementById('binanceConfigBtn');
    const closeBtn = document.getElementById('closeBinanceModal');
    const saveBtn = document.getElementById('saveBinanceConfig');

    const modeLiveBtn = document.getElementById('modeLiveBtn');
    const modePaperBtn = document.getElementById('modePaperBtn');
    const modeTestBtn = document.getElementById('modeTestBtn');

    let currentEnv = 'test'; // default

    if (!modal || !openBtn) return;

    const updateEnvUI = (env) => {
        currentEnv = env;
        const buttons = [
            { el: modeLiveBtn, val: 'prod' },
            { el: modePaperBtn, val: 'paper' },
            { el: modeTestBtn, val: 'test' }
        ];

        buttons.forEach(btn => {
            if (!btn.el) return;
            if (btn.val === env) {
                btn.el.classList.add('bg-primary', 'text-background-dark', 'border-primary');
                btn.el.classList.remove('border-surface-border', 'text-slate-400');
            } else {
                btn.el.classList.remove('bg-primary', 'text-background-dark', 'border-primary');
                btn.el.classList.add('border-surface-border', 'text-slate-400');
            }
        });
    };

    openBtn.onclick = async () => {
        modal.classList.remove('hidden');
        try {
            const token = localStorage.getItem("aat_token");
            const headers = {};
            if (token) headers["Authorization"] = `Bearer ${token}`;

            const res = await fetch('/api/admin/binance/config', { headers });
            const data = await res.json();
            if (data.success) {
                updateEnvUI(data.active_env);
            }
        } catch (err) {
            console.error('Failed to fetch config:', err);
        }
    };

    closeBtn.onclick = () => modal.classList.add('hidden');
    window.onclick = (e) => { if (e.target === modal) modal.classList.add('hidden'); };

    if (modeLiveBtn) modeLiveBtn.onclick = () => updateEnvUI('prod');
    if (modePaperBtn) modePaperBtn.onclick = () => updateEnvUI('paper');
    if (modeTestBtn) modeTestBtn.onclick = () => updateEnvUI('test');

    saveBtn.onclick = async () => {
        const payload = { active_env: currentEnv };

        const token = localStorage.getItem("aat_token");
        const headers = { 'Content-Type': 'application/json' };
        if (token) headers["Authorization"] = `Bearer ${token}`;

        saveBtn.disabled = true;
        saveBtn.innerText = 'Switching...';

        try {
            const res = await fetch('/api/admin/binance/config', {
                method: 'POST',
                headers: headers,
                body: JSON.stringify(payload)
            });
            const data = await res.json().catch(() => ({}));
            if (res.ok && data.success) {
                alert('系統執行模式已切換至 ' + currentEnv);
                modal.classList.add('hidden');
                updateExchangeLink();
                if (window.onBinanceConfigUpdate) window.onBinanceConfigUpdate();
            } else {
                const errorMsg = data.message || data.error || 'Unknown server error';
                alert('錯誤 (' + res.status + '): ' + errorMsg);
            }
        } catch (err) {
            alert('切換失敗: ' + err.message);
        } finally {
            saveBtn.disabled = false;
            saveBtn.innerText = '切換並更新連線 (Switch)';
        }
    };
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
}

