import type {
    TelegramAppState,
} from "./useTelegramApp";

export function logTelegramRuntime(
    state: TelegramAppState,
): void {
    if (!import.meta.env.DEV) {
        return;
    }

    console.info(
        "[Morning App] Telegram runtime",
        {
            environment: state.environment,
            available: state.isAvailable,
            hasInitData:
                state.initData.trim().length > 0,
            colorScheme: state.colorScheme,
        },
    );
}