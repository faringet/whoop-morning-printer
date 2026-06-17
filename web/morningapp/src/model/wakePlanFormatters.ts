const DEFAULT_TIME_ZONE = "Europe/Moscow";
const DEFAULT_LOCALE = "en-GB";

type FormatOptions = {
    locale?: string;
    timeZone?: string;
};

function parseDate(value: string): Date {
    const date = new Date(value);

    if (Number.isNaN(date.getTime())) {
        throw new Error(`Invalid ISO date: ${value}`);
    }

    return date;
}

export function formatWakeTime(
    value: string,
    options: FormatOptions = {},
): string {
    const date = parseDate(value);

    return new Intl.DateTimeFormat(
        options.locale ?? DEFAULT_LOCALE,
        {
            hour: "2-digit",
            minute: "2-digit",
            hour12: false,
            timeZone: options.timeZone ?? DEFAULT_TIME_ZONE,
        },
    ).format(date);
}

export function formatWakeDate(
    value: string,
    options: FormatOptions = {},
): string {
    const date = parseDate(value);

    return new Intl.DateTimeFormat(
        options.locale ?? DEFAULT_LOCALE,
        {
            weekday: "long",
            day: "2-digit",
            month: "long",
            timeZone: options.timeZone ?? DEFAULT_TIME_ZONE,
        },
    ).format(date);
}

export function formatWakeDateCompact(
    value: string,
    options: FormatOptions = {},
): string {
    const date = parseDate(value);

    return new Intl.DateTimeFormat(
        options.locale ?? DEFAULT_LOCALE,
        {
            weekday: "short",
            day: "2-digit",
            month: "short",
            timeZone: options.timeZone ?? DEFAULT_TIME_ZONE,
        },
    ).format(date);
}

export function minutesBetween(
    from: string,
    to: string,
): number {
    const fromDate = parseDate(from);
    const toDate = parseDate(to);

    return Math.round(
        (toDate.getTime() - fromDate.getTime()) / 60_000,
    );
}

export function formatDurationMinutes(
    totalMinutes: number,
): string {
    if (!Number.isFinite(totalMinutes)) {
        throw new Error(
            `Duration must be a finite number, got ${totalMinutes}`,
        );
    }

    const normalizedMinutes = Math.max(
        0,
        Math.round(totalMinutes),
    );

    const hours = Math.floor(normalizedMinutes / 60);
    const minutes = normalizedMinutes % 60;

    if (hours === 0) {
        return `${minutes} min`;
    }

    if (minutes === 0) {
        return `${hours}h`;
    }

    return `${hours}h ${minutes}m`;
}