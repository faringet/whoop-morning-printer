export type HttpErrorKind =
    | "timeout"
    | "network"
    | "aborted"
    | "unauthorized"
    | "forbidden"
    | "not_found"
    | "conflict"
    | "validation"
    | "rate_limited"
    | "server"
    | "invalid_response"
    | "unknown";

export type HttpErrorOptions = Readonly<{
    kind: HttpErrorKind;
    message: string;
    status?: number;
    responseBody?: unknown;
    originalError?: unknown;
}>;

export class HttpError extends Error {
    readonly kind: HttpErrorKind;

    readonly status: number | null;

    readonly responseBody: unknown;

    readonly originalError: unknown;

    constructor(
        options: HttpErrorOptions,
    ) {
        super(options.message);

        this.name = "HttpError";
        this.kind = options.kind;
        this.status = options.status ?? null;
        this.responseBody =
            options.responseBody ?? null;
        this.originalError =
            options.originalError ?? null;
    }
}

export function isHttpError(
    error: unknown,
): error is HttpError {
    return error instanceof HttpError;
}

export function getHttpErrorKind(
    status: number,
): HttpErrorKind {
    if (status === 401) {
        return "unauthorized";
    }

    if (status === 403) {
        return "forbidden";
    }

    if (status === 404) {
        return "not_found";
    }

    if (status === 409) {
        return "conflict";
    }

    if (
        status === 400 ||
        status === 422
    ) {
        return "validation";
    }

    if (status === 429) {
        return "rate_limited";
    }

    if (status >= 500) {
        return "server";
    }

    return "unknown";
}

export function getDefaultHttpErrorMessage(
    kind: HttpErrorKind,
): string {
    switch (kind) {
        case "timeout":
            return "The request timed out. Please try again.";

        case "network":
            return "Could not connect to the Morning App server.";

        case "aborted":
            return "The request was cancelled.";

        case "unauthorized":
            return "Telegram authorization is missing or expired.";

        case "forbidden":
            return "You are not allowed to perform this action.";

        case "not_found":
            return "The requested resource was not found.";

        case "conflict":
            return "The wake plan has changed. Please refresh and try again.";

        case "validation":
            return "The server rejected the request data.";

        case "rate_limited":
            return "Too many requests. Please try again shortly.";

        case "server":
            return "The Morning App server is temporarily unavailable.";

        case "invalid_response":
            return "The server returned an invalid response.";

        case "unknown":
            return "The request could not be completed.";
    }
}


export function getReadableErrorMessage(
    error: unknown,
    fallback: string,
): string {
    if (
        error instanceof Error &&
        error.message.trim().length > 0
    ) {
        return error.message;
    }

    return fallback;
}