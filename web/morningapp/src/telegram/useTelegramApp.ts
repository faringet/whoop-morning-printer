import {
    useEffect,
    useState,
} from "react";

import {
    telegramBridge,
} from "./telegram";
import type {
    TelegramColorScheme,
    TelegramRuntimeEnvironment,
    TelegramSafeAreaInset,
    TelegramThemeParams,
} from "./types";

const DEFAULT_BACKGROUND_COLOR = "#08080d";
const DEFAULT_HEADER_COLOR = "#08080d";
const DEFAULT_BOTTOM_BAR_COLOR = "#08080d";

const TELEGRAM_THEME_VARIABLES: ReadonlyArray<
    readonly [
        keyof TelegramThemeParams,
        string,
    ]
> = [
    ["bg_color", "--telegram-bg-color"],
    ["text_color", "--telegram-text-color"],
    ["hint_color", "--telegram-hint-color"],
    ["link_color", "--telegram-link-color"],
    ["button_color", "--telegram-button-color"],
    [
        "button_text_color",
        "--telegram-button-text-color",
    ],
    [
        "secondary_bg_color",
        "--telegram-secondary-bg-color",
    ],
    [
        "header_bg_color",
        "--telegram-header-bg-color",
    ],
    [
        "accent_text_color",
        "--telegram-accent-text-color",
    ],
    [
        "section_bg_color",
        "--telegram-section-bg-color",
    ],
    [
        "section_header_text_color",
        "--telegram-section-header-text-color",
    ],
    [
        "subtitle_text_color",
        "--telegram-subtitle-text-color",
    ],
    [
        "destructive_text_color",
        "--telegram-destructive-text-color",
    ],
    [
        "section_separator_color",
        "--telegram-section-separator-color",
    ],
    [
        "bottom_bar_bg_color",
        "--telegram-bottom-bar-bg-color",
    ],
];

export type TelegramAppState = Readonly<{
    environment: TelegramRuntimeEnvironment;
    isAvailable: boolean;
    initData: string;
    colorScheme: TelegramColorScheme;
    themeParams: TelegramThemeParams;
}>;

export function useTelegramApp():
    TelegramAppState {
    const [telegramState, setTelegramState] =
        useState<TelegramAppState>(
            readTelegramAppState,
        );

    useEffect(() => {
        const root = document.documentElement;

        function synchronizeAppearance() {
            const colorScheme =
                telegramBridge.getColorScheme();

            const themeParams =
                telegramBridge.getThemeParams();

            applyEnvironmentAttributes(
                root,
                colorScheme,
            );

            applyThemeVariables(
                root,
                themeParams,
            );

            applyTelegramChromeColors(root);

            setTelegramState(
                readTelegramAppState(),
            );
        }

        function synchronizeSafeArea() {
            applySafeAreaVariables(
                root,
                telegramBridge.getSafeAreaInset(),
                telegramBridge
                    .getContentSafeAreaInset(),
            );
        }

        function handleThemeChanged() {
            synchronizeAppearance();
        }

        function handleSafeAreaChanged() {
            synchronizeSafeArea();
        }

        function handleContentSafeAreaChanged() {
            synchronizeSafeArea();
        }

        applyEnvironmentAttributes(
            root,
            telegramBridge.getColorScheme(),
        );

        applyThemeVariables(
            root,
            telegramBridge.getThemeParams(),
        );

        synchronizeSafeArea();
        applyTelegramChromeColors(root);

        telegramBridge.onEvent(
            "themeChanged",
            handleThemeChanged,
        );

        telegramBridge.onEvent(
            "safeAreaChanged",
            handleSafeAreaChanged,
        );

        telegramBridge.onEvent(
            "contentSafeAreaChanged",
            handleContentSafeAreaChanged,
        );

        telegramBridge.expand();
        telegramBridge.ready();

        return () => {
            telegramBridge.offEvent(
                "themeChanged",
                handleThemeChanged,
            );

            telegramBridge.offEvent(
                "safeAreaChanged",
                handleSafeAreaChanged,
            );

            telegramBridge.offEvent(
                "contentSafeAreaChanged",
                handleContentSafeAreaChanged,
            );

            delete root.dataset.telegramEnvironment;
            delete root.dataset.telegramColorScheme;

            resetSafeAreaVariables(root);
            resetThemeVariables(root);
        };
    }, []);

    return telegramState;
}

function readTelegramAppState():
    TelegramAppState {
    return {
        environment: telegramBridge.environment,
        isAvailable: telegramBridge.isAvailable,
        initData: telegramBridge.getInitData(),
        colorScheme:
            telegramBridge.getColorScheme(),
        themeParams:
            telegramBridge.getThemeParams(),
    };
}

function applyEnvironmentAttributes(
    root: HTMLElement,
    colorScheme: TelegramColorScheme,
): void {
    root.dataset.telegramEnvironment =
        telegramBridge.environment;

    root.dataset.telegramColorScheme =
        colorScheme;
}

function applyTelegramChromeColors(
    root: HTMLElement,
): void {
    const backgroundColor =
        readCssColor(
            root,
            "--color-background",
            DEFAULT_BACKGROUND_COLOR,
        );

    const headerColor =
        readCssColor(
            root,
            "--color-background",
            DEFAULT_HEADER_COLOR,
        );

    const bottomBarColor =
        readCssColor(
            root,
            "--color-background",
            DEFAULT_BOTTOM_BAR_COLOR,
        );

    telegramBridge.setHeaderColor(
        headerColor,
    );

    telegramBridge.setBackgroundColor(
        backgroundColor,
    );

    telegramBridge.setBottomBarColor(
        bottomBarColor,
    );

    updateThemeColorMeta(backgroundColor);
}

function applyThemeVariables(
    root: HTMLElement,
    themeParams: TelegramThemeParams,
): void {
    TELEGRAM_THEME_VARIABLES.forEach(
        ([themeKey, cssVariable]) => {
            const value = themeParams[themeKey];

            if (
                typeof value === "string" &&
                value.trim().length > 0
            ) {
                root.style.setProperty(
                    cssVariable,
                    value,
                );

                return;
            }

            root.style.removeProperty(
                cssVariable,
            );
        },
    );
}

function resetThemeVariables(
    root: HTMLElement,
): void {
    TELEGRAM_THEME_VARIABLES.forEach(
        ([, cssVariable]) => {
            root.style.removeProperty(
                cssVariable,
            );
        },
    );
}

function applySafeAreaVariables(
    root: HTMLElement,
    safeArea: TelegramSafeAreaInset,
    contentSafeArea: TelegramSafeAreaInset,
): void {
    setPixelVariable(
        root,
        "--telegram-safe-area-top",
        safeArea.top,
    );

    setPixelVariable(
        root,
        "--telegram-safe-area-right",
        safeArea.right,
    );

    setPixelVariable(
        root,
        "--telegram-safe-area-bottom",
        safeArea.bottom,
    );

    setPixelVariable(
        root,
        "--telegram-safe-area-left",
        safeArea.left,
    );

    setPixelVariable(
        root,
        "--telegram-content-safe-area-top",
        contentSafeArea.top,
    );

    setPixelVariable(
        root,
        "--telegram-content-safe-area-right",
        contentSafeArea.right,
    );

    setPixelVariable(
        root,
        "--telegram-content-safe-area-bottom",
        contentSafeArea.bottom,
    );

    setPixelVariable(
        root,
        "--telegram-content-safe-area-left",
        contentSafeArea.left,
    );

    setPixelVariable(
        root,
        "--safe-area-top",
        Math.max(
            safeArea.top,
            contentSafeArea.top,
        ),
    );

    setPixelVariable(
        root,
        "--safe-area-right",
        Math.max(
            safeArea.right,
            contentSafeArea.right,
        ),
    );

    setPixelVariable(
        root,
        "--safe-area-bottom",
        Math.max(
            safeArea.bottom,
            contentSafeArea.bottom,
        ),
    );

    setPixelVariable(
        root,
        "--safe-area-left",
        Math.max(
            safeArea.left,
            contentSafeArea.left,
        ),
    );
}

function resetSafeAreaVariables(
    root: HTMLElement,
): void {
    const variables = [
        "--telegram-safe-area-top",
        "--telegram-safe-area-right",
        "--telegram-safe-area-bottom",
        "--telegram-safe-area-left",
        "--telegram-content-safe-area-top",
        "--telegram-content-safe-area-right",
        "--telegram-content-safe-area-bottom",
        "--telegram-content-safe-area-left",
    ];

    variables.forEach((variable) => {
        root.style.removeProperty(variable);
    });

    root.style.setProperty(
        "--safe-area-top",
        "0px",
    );

    root.style.setProperty(
        "--safe-area-right",
        "0px",
    );

    root.style.setProperty(
        "--safe-area-bottom",
        "0px",
    );

    root.style.setProperty(
        "--safe-area-left",
        "0px",
    );
}

function setPixelVariable(
    root: HTMLElement,
    name: string,
    value: number,
): void {
    const normalizedValue =
        Number.isFinite(value)
            ? Math.max(0, value)
            : 0;

    root.style.setProperty(
        name,
        `${normalizedValue}px`,
    );
}

function readCssColor(
    root: HTMLElement,
    propertyName: string,
    fallback: string,
): string {
    const value = getComputedStyle(root)
        .getPropertyValue(propertyName)
        .trim();

    return value || fallback;
}

function updateThemeColorMeta(
    color: string,
): void {
    const metaElement =
        document.querySelector<HTMLMetaElement>(
            'meta[name="theme-color"]',
        );

    if (!metaElement) {
        return;
    }

    metaElement.content = color;
}