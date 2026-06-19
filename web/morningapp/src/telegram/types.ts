export type TelegramRuntimeEnvironment =
    | "telegram"
    | "browser";

export type TelegramColorScheme =
    | "light"
    | "dark";

export type TelegramHapticImpactStyle =
    | "light"
    | "medium"
    | "heavy"
    | "rigid"
    | "soft";

export type TelegramHapticNotificationType =
    | "error"
    | "success"
    | "warning";

export type TelegramBackButtonHandler =
    () => void;

export type TelegramSafeAreaInset = Readonly<{
    top: number;
    right: number;
    bottom: number;
    left: number;
}>;

export type TelegramThemeParams = Readonly<{
    bg_color?: string;
    text_color?: string;
    hint_color?: string;
    link_color?: string;
    button_color?: string;
    button_text_color?: string;
    secondary_bg_color?: string;

    header_bg_color?: string;
    accent_text_color?: string;
    section_bg_color?: string;
    section_header_text_color?: string;
    subtitle_text_color?: string;
    destructive_text_color?: string;
    section_separator_color?: string;
    bottom_bar_bg_color?: string;
}>;

export type TelegramWebAppEventName =
    | "themeChanged"
    | "safeAreaChanged"
    | "contentSafeAreaChanged";

export type TelegramWebAppEventHandler =
    () => void;

export type TelegramBackButtonApi = {
    readonly isVisible: boolean;

    show(): TelegramBackButtonApi;

    hide(): TelegramBackButtonApi;

    onClick(
        handler: TelegramBackButtonHandler,
    ): TelegramBackButtonApi;

    offClick(
        handler: TelegramBackButtonHandler,
    ): TelegramBackButtonApi;
};

export type TelegramHapticFeedbackApi = {
    impactOccurred(
        style: TelegramHapticImpactStyle,
    ): TelegramHapticFeedbackApi;

    notificationOccurred(
        type: TelegramHapticNotificationType,
    ): TelegramHapticFeedbackApi;

    selectionChanged(): TelegramHapticFeedbackApi;
};

export type TelegramWebAppApi = {
    readonly initData: string;
    readonly version: string;
    readonly platform: string;

    readonly colorScheme: TelegramColorScheme;
    readonly themeParams: TelegramThemeParams;

    readonly isFullscreen?: boolean;

    readonly safeAreaInset?:
        TelegramSafeAreaInset;

    readonly contentSafeAreaInset?:
        TelegramSafeAreaInset;

    readonly BackButton?:
        TelegramBackButtonApi;

    readonly HapticFeedback?:
        TelegramHapticFeedbackApi;

    ready(): void;

    expand(): void;

    close(): void;

    requestFullscreen?(): void;

    onEvent?(
        eventType: TelegramWebAppEventName,
        eventHandler: TelegramWebAppEventHandler,
    ): void;

    offEvent?(
        eventType: TelegramWebAppEventName,
        eventHandler: TelegramWebAppEventHandler,
    ): void;

    isVersionAtLeast?(
        version: string,
    ): boolean;

    setHeaderColor?(
        color: string,
    ): void;

    setBackgroundColor?(
        color: string,
    ): void;

    setBottomBarColor?(
        color: string,
    ): void;

    enableClosingConfirmation?(): void;

    disableClosingConfirmation?(): void;
};

export type TelegramHostWindow =
    Window & {
    Telegram?: {
        WebApp?: TelegramWebAppApi;
    };
};