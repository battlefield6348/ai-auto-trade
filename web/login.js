/**
 * Login page logic
 */

const el = (id) => document.getElementById(id);

let isRegister = false;

function toggleMode() {
    isRegister = !isRegister;
    const title = el('formTitle');
    const subtitle = el('formSubtitle');
    const nameField = el('nameField');
    const toggleHint = el('toggleHint');
    const toggleBtn = el('toggleBtn');
    const submitBtn = el('loginForm').querySelector('button[type="submit"] span:first-child');

    if (isRegister) {
        title.textContent = "建立帳號";
        subtitle.textContent = "填寫資料以開始您的智能交易之旅";
        nameField.classList.remove('hidden');
        toggleHint.textContent = "已有帳號？";
        toggleBtn.textContent = "點此登入 (Login)";
        submitBtn.textContent = "註冊並進入";
    } else {
        title.textContent = "歡迎回來";
        subtitle.textContent = "請輸入您的憑據以訪問控制台";
        nameField.classList.add('hidden');
        toggleHint.textContent = "還沒有帳號？";
        toggleBtn.textContent = "點此註冊 (Register)";
        submitBtn.textContent = "進入控制台";
    }
}

async function handleAuth(e) {
    e.preventDefault();

    const email = el('userEmail').value;
    const password = el('userPassword').value;
    const name = el('userName')?.value;

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

        const res = await fetch("/api/auth/login", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email, password }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message || data.error || "登入失敗");

        localStorage.setItem("aat_token", data.access_token);
        localStorage.setItem("aat_email", email);

        // Redirect to dashboard
        window.location.href = "/";
    } catch (err) {
        alert(err.message);
    }
}

function bootstrap() {
    const form = el('loginForm');
    const toggle = el('toggleBtn');

    if (form) form.addEventListener('submit', handleAuth);
    if (toggle) toggle.addEventListener('click', (e) => {
        e.preventDefault();
        toggleMode();
    });

    // If already logged in, redirect to home
    if (localStorage.getItem('aat_token')) {
        window.location.href = "/";
    }
}

bootstrap();
