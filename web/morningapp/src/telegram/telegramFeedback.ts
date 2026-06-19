import {
    telegramBridge,
} from "./telegram";

export function notifyScheduleOpened(): void {
    telegramBridge.impactOccurred("light");
}

export function notifyWakePlanSaved(): void {
    telegramBridge.notificationOccurred(
        "success",
    );
}

export function notifyCancelDialogOpened(): void {
    telegramBridge.notificationOccurred(
        "warning",
    );
}

export function notifyWakePlanCancelled(): void {
    telegramBridge.notificationOccurred(
        "success",
    );
}

export function notifyOperationFailed(): void {
    telegramBridge.notificationOccurred(
        "error",
    );
}

export function notifySelectionChanged(): void {
    telegramBridge.selectionChanged();
}