import { useEffect } from "react";

import { telegramBridge } from "./telegram";

export function useTelegramClosingConfirmation(
    isEnabled: boolean,
): void {
    useEffect(() => {
        if (isEnabled) {
            telegramBridge.enableClosingConfirmation();
        } else {
            telegramBridge.disableClosingConfirmation();
        }

        return () => {
            telegramBridge.disableClosingConfirmation();
        };
    }, [isEnabled]);
}