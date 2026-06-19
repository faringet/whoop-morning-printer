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
  TelegramWebAppEventHandler,
  TelegramWebAppEventName,
} from "./types";

const emptyThemeParams: TelegramThemeParams =
  Object.freeze({});

export class BrowserTelegramBridge
  implements TelegramBridge
{
  readonly environment = "browser" as const;

  readonly isAvailable = false;

  getInitData(): string {
    return "";
  }

  getColorScheme(): TelegramColorScheme {
    if (
      typeof window === "undefined" ||
      typeof window.matchMedia !== "function"
    ) {
      return "dark";
    }

    const prefersLight =
      window.matchMedia(
        "(prefers-color-scheme: light)",
      ).matches;

    return prefersLight
      ? "light"
      : "dark";
  }

  getThemeParams(): TelegramThemeParams {
    return emptyThemeParams;
  }

  getSafeAreaInset(): TelegramSafeAreaInset {
    return emptySafeAreaInset;
  }

  getContentSafeAreaInset():
    TelegramSafeAreaInset {
    return emptySafeAreaInset;
  }

  ready(): void {
  }

  expand(): void {
  }

  requestFullscreen(): void {
  }

  close(): void {
  }

  setHeaderColor(
    color: string,
  ): void {
    void color;
  }

  setBackgroundColor(
    color: string,
  ): void {
    void color;
  }

  setBottomBarColor(
    color: string,
  ): void {
    void color;
  }

  enableClosingConfirmation(): void {
  }

  disableClosingConfirmation(): void {
  }

  setBackButtonHandler(
    handler: TelegramBackButtonHandler | null,
  ): void {
    void handler;
  }

  onEvent(
    eventName: TelegramWebAppEventName,
    handler: TelegramWebAppEventHandler,
  ): void {
    void eventName;
    void handler;
  }

  offEvent(
    eventName: TelegramWebAppEventName,
    handler: TelegramWebAppEventHandler,
  ): void {
    void eventName;
    void handler;
  }

  impactOccurred(
    style: TelegramHapticImpactStyle,
  ): void {
    void style;
  }

  notificationOccurred(
    type: TelegramHapticNotificationType,
  ): void {
    void type;
  }

  selectionChanged(): void {
  }
}

export const browserTelegramBridge =
  new BrowserTelegramBridge();
