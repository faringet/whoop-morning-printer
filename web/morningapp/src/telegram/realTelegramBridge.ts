import {
  emptySafeAreaInset,
  type TelegramBridge,
} from "./telegramBridge";
import type {
  TelegramBackButtonHandler,
  TelegramColorScheme,
  TelegramHapticImpactStyle,
  TelegramHapticNotificationType,
  TelegramSafeAreaInset,
  TelegramThemeParams,
  TelegramWebAppApi,
  TelegramWebAppEventHandler,
  TelegramWebAppEventName,
} from "./types";

export class RealTelegramBridge
  implements TelegramBridge
{
  readonly environment = "telegram" as const;

  readonly isAvailable = true;

  private readonly webApp: TelegramWebAppApi;

  private currentBackButtonHandler:
    TelegramBackButtonHandler | null = null;

  constructor(
    webApp: TelegramWebAppApi,
  ) {
    this.webApp = webApp;
  }

  getInitData(): string {
    return this.webApp.initData ?? "";
  }

  getColorScheme(): TelegramColorScheme {
    return this.webApp.colorScheme === "light"
      ? "light"
      : "dark";
  }

  getThemeParams(): TelegramThemeParams {
    return {
      ...this.webApp.themeParams,
    };
  }

  getSafeAreaInset(): TelegramSafeAreaInset {
    return normalizeSafeAreaInset(
      this.webApp.safeAreaInset,
    );
  }

  getContentSafeAreaInset():
    TelegramSafeAreaInset {
    return normalizeSafeAreaInset(
      this.webApp.contentSafeAreaInset,
    );
  }

  ready(): void {
    runTelegramAction(
      "ready",
      () => {
        this.webApp.ready();
      },
    );
  }

  expand(): void {
    runTelegramAction(
      "expand",
      () => {
        this.webApp.expand();
      },
    );
  }

  close(): void {
    runTelegramAction(
      "close",
      () => {
        this.webApp.close();
      },
    );
  }

  setHeaderColor(
    color: string,
  ): void {
    if (!this.webApp.setHeaderColor) {
      return;
    }

    runTelegramAction(
      "setHeaderColor",
      () => {
        this.webApp.setHeaderColor?.(
          color,
        );
      },
    );
  }

  setBackgroundColor(
    color: string,
  ): void {
    if (!this.webApp.setBackgroundColor) {
      return;
    }

    runTelegramAction(
      "setBackgroundColor",
      () => {
        this.webApp.setBackgroundColor?.(
          color,
        );
      },
    );
  }

  setBottomBarColor(
    color: string,
  ): void {
    if (!this.webApp.setBottomBarColor) {
      return;
    }

    runTelegramAction(
      "setBottomBarColor",
      () => {
        this.webApp.setBottomBarColor?.(
          color,
        );
      },
    );
  }

  enableClosingConfirmation(): void {
    if (
      !this.webApp.enableClosingConfirmation
    ) {
      return;
    }

    runTelegramAction(
      "enableClosingConfirmation",
      () => {
        this.webApp
          .enableClosingConfirmation?.();
      },
    );
  }

  disableClosingConfirmation(): void {
    if (
      !this.webApp.disableClosingConfirmation
    ) {
      return;
    }

    runTelegramAction(
      "disableClosingConfirmation",
      () => {
        this.webApp
          .disableClosingConfirmation?.();
      },
    );
  }

  setBackButtonHandler(
    handler: TelegramBackButtonHandler | null,
  ): void {
    const backButton =
      this.webApp.BackButton;

    if (!backButton) {
      this.currentBackButtonHandler =
        handler;

      return;
    }

    if (this.currentBackButtonHandler) {
      const previousHandler =
        this.currentBackButtonHandler;

      runTelegramAction(
        "BackButton.offClick",
        () => {
          backButton.offClick(
            previousHandler,
          );
        },
      );
    }

    this.currentBackButtonHandler =
      handler;

    if (!handler) {
      runTelegramAction(
        "BackButton.hide",
        () => {
          backButton.hide();
        },
      );

      return;
    }

    runTelegramAction(
      "BackButton.onClick",
      () => {
        backButton.onClick(handler);
      },
    );

    runTelegramAction(
      "BackButton.show",
      () => {
        backButton.show();
      },
    );
  }

  onEvent(
    eventName: TelegramWebAppEventName,
    handler: TelegramWebAppEventHandler,
  ): void {
    if (!this.webApp.onEvent) {
      return;
    }

    runTelegramAction(
      `onEvent:${eventName}`,
      () => {
        this.webApp.onEvent?.(
          eventName,
          handler,
        );
      },
    );
  }

  offEvent(
    eventName: TelegramWebAppEventName,
    handler: TelegramWebAppEventHandler,
  ): void {
    if (!this.webApp.offEvent) {
      return;
    }

    runTelegramAction(
      `offEvent:${eventName}`,
      () => {
        this.webApp.offEvent?.(
          eventName,
          handler,
        );
      },
    );
  }

  impactOccurred(
    style: TelegramHapticImpactStyle,
  ): void {
    const hapticFeedback =
      this.webApp.HapticFeedback;

    if (!hapticFeedback) {
      return;
    }

    runTelegramAction(
      "HapticFeedback.impactOccurred",
      () => {
        hapticFeedback.impactOccurred(
          style,
        );
      },
    );
  }

  notificationOccurred(
    type: TelegramHapticNotificationType,
  ): void {
    const hapticFeedback =
      this.webApp.HapticFeedback;

    if (!hapticFeedback) {
      return;
    }

    runTelegramAction(
      "HapticFeedback.notificationOccurred",
      () => {
        hapticFeedback.notificationOccurred(
          type,
        );
      },
    );
  }

  selectionChanged(): void {
    const hapticFeedback =
      this.webApp.HapticFeedback;

    if (!hapticFeedback) {
      return;
    }

    runTelegramAction(
      "HapticFeedback.selectionChanged",
      () => {
        hapticFeedback.selectionChanged();
      },
    );
  }
}

function normalizeSafeAreaInset(
  inset: TelegramSafeAreaInset | undefined,
): TelegramSafeAreaInset {
  if (!inset) {
    return emptySafeAreaInset;
  }

  return {
    top: normalizeInsetValue(inset.top),
    right: normalizeInsetValue(inset.right),
    bottom: normalizeInsetValue(inset.bottom),
    left: normalizeInsetValue(inset.left),
  };
}

function normalizeInsetValue(
  value: number,
): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.max(0, value);
}

function runTelegramAction(
  actionName: string,
  action: () => void,
): void {
  try {
    action();
  } catch (error) {
    console.warn(
      `Telegram WebApp action failed: ${actionName}`,
      error,
    );
  }
}
