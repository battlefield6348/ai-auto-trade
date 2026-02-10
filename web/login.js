/**
 * Login page logic
 */

const el = (id) => document.getElementById(id);

async function handleAuth(e) {
    e.preventDefault();

    const email = el('userEmail').value;
    const password = el('userPassword').value;

    try {
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

    if (form) form.addEventListener('submit', handleAuth);

    // If already logged in, redirect to home
    if (localStorage.getItem('aat_token')) {
        window.location.href = "/";
    }
}

bootstrap();
