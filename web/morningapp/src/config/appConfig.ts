export type WakePlanServiceMode =
    | "mock"
    | "http";

export type AppConfig = Readonly<{
    wakePlanServiceMode: WakePlanServiceMode;
    apiBaseUrl: string;
    apiRequestTimeoutMs: number;
}>;

const DEFAULT_WAKE_PLAN_SERVICE_MODE:
    WakePlanServiceMode = "mock";

const DEFAULT_API_BASE_URL =
    "http://localhost:8080";

const DEFAULT_API_REQUEST_TIMEOUT_MS =
    10_000;

export const appConfig: AppConfig =
    Object.freeze({
        wakePlanServiceMode:
            readWakePlanServiceMode(),

        apiBaseUrl:
            readApiBaseUrl(),

        apiRequestTimeoutMs:
            readApiRequestTimeoutMs(),
    });

function readWakePlanServiceMode():
    WakePlanServiceMode {
    const value =
        import.meta.env.VITE_WAKE_PLAN_SERVICE
            ?.trim()
            .toLowerCase();

    if (!value) {
        return DEFAULT_WAKE_PLAN_SERVICE_MODE;
    }

    if (
        value === "mock" ||
        value === "http"
    ) {
        return value;
    }

    throw new Error(
        [
            "Invalid VITE_WAKE_PLAN_SERVICE value.",
            'Expected "mock" or "http",',
            `received "${value}".`,
        ].join(" "),
    );
}

function readApiBaseUrl(): string {
    const rawValue =
        import.meta.env.VITE_API_BASE_URL
            ?.trim();

    const value =
        rawValue || DEFAULT_API_BASE_URL;

    validateApiBaseUrl(value);

    return removeTrailingSlashes(value);
}

function readApiRequestTimeoutMs(): number {
    const rawValue =
        import.meta.env
            .VITE_API_REQUEST_TIMEOUT_MS
            ?.trim();

    if (!rawValue) {
        return DEFAULT_API_REQUEST_TIMEOUT_MS;
    }

    const value = Number(rawValue);

    if (
        !Number.isInteger(value) ||
        value <= 0
    ) {
        throw new Error(
            [
                "Invalid VITE_API_REQUEST_TIMEOUT_MS value.",
                "Expected a positive integer,",
                `received "${rawValue}".`,
            ].join(" "),
        );
    }

    return value;
}

function validateApiBaseUrl(
    value: string,
): void {
    let url: URL;

    try {
        url = new URL(value);
    } catch {
        throw new Error(
            [
                "Invalid VITE_API_BASE_URL value.",
                `Received "${value}".`,
            ].join(" "),
        );
    }

    if (
        url.protocol !== "http:" &&
        url.protocol !== "https:"
    ) {
        throw new Error(
            [
                "Invalid VITE_API_BASE_URL protocol.",
                "Only http and https are supported.",
            ].join(" "),
        );
    }
}

function removeTrailingSlashes(
    value: string,
): string {
    return value.replace(/\/+$/, "");
}