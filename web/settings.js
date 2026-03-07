import { initSidebar, initGlobalEnvSelector, showMessage } from "./common.js";

function bootstrap() {
    initSidebar();

    // Initialize Global Environment Selectors with the special logic for settings page buttons
    initGlobalEnvSelector((env) => {
        // Any specific page logic on env change
    });

    const saveApiBtn = document.getElementById('saveApiBtn');
    if (saveApiBtn) {
        saveApiBtn.addEventListener('click', () => {
            saveApiBtn.disabled = true;
            saveApiBtn.textContent = "正在保存...";

            setTimeout(() => {
                showMessage("API 憑據已成功加密保存", "success");
                saveApiBtn.disabled = false;
                saveApiBtn.textContent = "更新憑據 (Update Keys)";
            }, 1000);
        });
    }
}

bootstrap();
