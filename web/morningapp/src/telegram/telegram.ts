import {
    browserTelegramBridge,
} from "./browserTelegramBridge";
import {
    RealTelegramBridge,
} from "./realTelegramBridge";
import type {
    TelegramBridge,
} from "./telegramBridge";
import type {
    TelegramHostWindow,
    TelegramWebAppApi,
} from "./types";

export const telegramBridge: TelegramBridge =
    createTelegramBridge();

export function createTelegramBridge():
    TelegramBridge {
    const webApp = getTelegramWebApp();

    if (!webApp) {
        return browserTelegramBridge;
    }

    if (!hasTelegramLaunchContext(webApp)) {
        return browserTelegramBridge;
    }

    return new RealTelegramBridge(webApp);
}

function getTelegramWebApp():
    TelegramWebAppApi | null {
    if (typeof window === "undefined") {
        return null;
    }

    const hostWindow =
        window as TelegramHostWindow;

    const webApp =
        hostWindow.Telegram?.WebApp;

    if (!isTelegramWebAppApi(webApp)) {
        return null;
    }

    return webApp;
}

function hasTelegramLaunchContext(
    webApp: TelegramWebAppApi,
): boolean {
    return webApp.initData.trim().length > 0;
}

function isTelegramWebAppApi(
    value: unknown,
): value is TelegramWebAppApi {
    if (
        typeof value !== "object" ||
        value === null
    ) {
        return false;
    }

    const candidate =
        value as Partial<TelegramWebAppApi>;

    return (
        typeof candidate.ready === "function" &&
        typeof candidate.expand === "function" &&
        typeof candidate.close === "function" &&
        typeof candidate.initData === "string" &&
        typeof candidate.version === "string" &&
        typeof candidate.platform === "string"
    );
}