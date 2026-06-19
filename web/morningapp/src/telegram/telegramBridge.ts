import type {
  TelegramBackButtonHandler,
  TelegramColorScheme,
  TelegramHapticImpactStyle,
  TelegramHapticNotificationType,
  TelegramRuntimeEnvironment,
  TelegramSafeAreaInset,
  TelegramThemeParams,
  TelegramWebAppEventHandler,
  TelegramWebAppEventName,
} from "./types";

export const emptySafeAreaInset: TelegramSafeAreaInset =
  Object.freeze({
    top: 0,
    right: 0,
    bottom: 0,
    left: 0,
  });

export interface TelegramBridge {
  readonly environment:
    TelegramRuntimeEnvironment;

  readonly isAvailable: boolean;

  getInitData(): string;

  getColorScheme(): TelegramColorScheme;

  getThemeParams(): TelegramThemeParams;

  getSafeAreaInset(): TelegramSafeAreaInset;

  getContentSafeAreaInset():
    TelegramSafeAreaInset;

  ready(): void;

  expand(): void;

  requestFullscreen(): void;

  close(): void;

  setHeaderColor(color: string): void;

  setBackgroundColor(color: string): void;

  setBottomBarColor(color: string): void;

  enableClosingConfirmation(): void;

  disableClosingConfirmation(): void;

  setBackButtonHandler(
    handler: TelegramBackButtonHandler | null,
  ): void;

  onEvent(
    eventName: TelegramWebAppEventName,
    handler: TelegramWebAppEventHandler,
  ): void;

  offEvent(
    eventName: TelegramWebAppEventName,
    handler: TelegramWebAppEventHandler,
  ): void;

  impactOccurred(
    style: TelegramHapticImpactStyle,
  ): void;

  notificationOccurred(
    type: TelegramHapticNotificationType,
  ): void;

  selectionChanged(): void;
}