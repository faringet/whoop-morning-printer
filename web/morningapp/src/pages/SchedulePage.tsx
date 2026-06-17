import {
    type FormEvent,
    useMemo,
    useState,
} from "react";

import type {
    SaveWakePlanInput,
    WakePlan,
} from "../model/wakePlan";

const MOSCOW_TIME_ZONE = "Europe/Moscow";
const MOSCOW_UTC_OFFSET = "+03:00";
const DEFAULT_WAKE_TIME = "08:00";

type SchedulePageProps = {
    wakePlan: WakePlan | null;
    onBack: () => void;
    onSave: (
        input: SaveWakePlanInput,
    ) => Promise<void>;
};

function SchedulePage({
                          wakePlan,
                          onBack,
                          onSave,
                      }: SchedulePageProps) {
    const initialValues = useMemo(
        () => getInitialWakeValues(wakePlan),
        [wakePlan],
    );

    const [wakeDate, setWakeDate] = useState(
        initialValues.date,
    );

    const [wakeTime, setWakeTime] = useState(
        initialValues.time,
    );

    const [isSaving, setIsSaving] =
        useState(false);

    const [saveError, setSaveError] =
        useState<string | null>(null);

    const wakeAt = useMemo(
        () => buildMoscowISODate(wakeDate, wakeTime),
        [wakeDate, wakeTime],
    );

    const canSubmit =
        !isSaving &&
        wakeDate.length > 0 &&
        wakeTime.length > 0 &&
        wakeAt !== null;

    async function handleSubmit(
        event: FormEvent<HTMLFormElement>,
    ) {
        event.preventDefault();

        if (!wakeAt || isSaving) {
            return;
        }

        setIsSaving(true);
        setSaveError(null);

        try {
            await onSave({
                wakeAt,
            });
        } catch (error) {
            console.error(
                "Failed to save wake plan",
                error,
            );

            setSaveError(
                "Could not arm the morning. Please try again.",
            );
        } finally {
            setIsSaving(false);
        }
    }

    function handleBack() {
        if (isSaving) {
            return;
        }

        onBack();
    }

    return (
        <main className="app-shell">
            <header className="schedule-header">
                <button
                    className="schedule-back-button"
                    type="button"
                    onClick={handleBack}
                    aria-label="Return to Morning Station"
                    disabled={isSaving}
                >
                    ←
                </button>

                <div>
                    <p className="terminal-label text-cyan">
                        Morning configuration
                    </p>

                    <h1 className="schedule-title">
                        {wakePlan
                            ? "Edit morning"
                            : "Set morning"}
                    </h1>
                </div>
            </header>

            <form
                className="schedule-form"
                onSubmit={(event) => {
                    void handleSubmit(event);
                }}
            >
                <section className="panel panel-magenta schedule-panel">
                    <div className="schedule-panel__heading">
                        <p className="terminal-label text-magenta">
                            Wake signal
                        </p>

                        <span className="schedule-panel__index">
              01
            </span>
                    </div>

                    <label className="schedule-field">
            <span className="schedule-field__label">
              Date
            </span>

                        <input
                            className="input schedule-field__input"
                            type="date"
                            value={wakeDate}
                            min={getTodayInMoscow()}
                            disabled={isSaving}
                            onChange={(event) => {
                                setWakeDate(event.target.value);
                                setSaveError(null);
                            }}
                            required
                        />
                    </label>

                    <label className="schedule-field">
            <span className="schedule-field__label">
              Wake time
            </span>

                        <input
                            className="input schedule-field__input schedule-field__input--time"
                            type="time"
                            value={wakeTime}
                            disabled={isSaving}
                            onChange={(event) => {
                                setWakeTime(event.target.value);
                                setSaveError(null);
                            }}
                            step={60}
                            required
                        />
                    </label>
                </section>

                {saveError ? (
                    <div
                        className="schedule-error"
                        role="alert"
                    >
            <span
                className="schedule-error__marker"
                aria-hidden="true"
            >
              !
            </span>

                        <span>{saveError}</span>
                    </div>
                ) : null}

                <div className="schedule-actions">
                    <button
                        className="button button-primary"
                        type="submit"
                        disabled={!canSubmit}
                    >
                        {isSaving
                            ? "Arming..."
                            : "Arm morning"}
                    </button>

                    <button
                        className="button button-ghost"
                        type="button"
                        onClick={handleBack}
                        disabled={isSaving}
                    >
                        Cancel editing
                    </button>
                </div>
            </form>
        </main>
    );
}

type WakeInputValues = {
    date: string;
    time: string;
};

function getInitialWakeValues(
    wakePlan: WakePlan | null,
): WakeInputValues {
    if (wakePlan) {
        return {
            date: formatDateInputValue(
                wakePlan.wakeAt,
            ),
            time: formatTimeInputValue(
                wakePlan.wakeAt,
            ),
        };
    }

    return {
        date: getTomorrowInMoscow(),
        time: DEFAULT_WAKE_TIME,
    };
}

function buildMoscowISODate(
    dateValue: string,
    timeValue: string,
): string | null {
    if (!dateValue || !timeValue) {
        return null;
    }

    const value =
        `${dateValue}T${timeValue}:00` +
        MOSCOW_UTC_OFFSET;

    const date = new Date(value);

    if (Number.isNaN(date.getTime())) {
        return null;
    }

    return value;
}

function formatDateInputValue(
    isoValue: string,
): string {
    const parts = getMoscowDateParts(
        new Date(isoValue),
    );

    return `${parts.year}-${parts.month}-${parts.day}`;
}

function formatTimeInputValue(
    isoValue: string,
): string {
    const parts = getMoscowDateParts(
        new Date(isoValue),
    );

    return `${parts.hour}:${parts.minute}`;
}

function getTodayInMoscow(): string {
    const parts = getMoscowDateParts(
        new Date(),
    );

    return `${parts.year}-${parts.month}-${parts.day}`;
}

function getTomorrowInMoscow(): string {
    const today = getMoscowDateParts(
        new Date(),
    );

    const date = new Date(
        Date.UTC(
            Number(today.year),
            Number(today.month) - 1,
            Number(today.day) + 1,
        ),
    );

    return [
        date.getUTCFullYear(),
        padDatePart(date.getUTCMonth() + 1),
        padDatePart(date.getUTCDate()),
    ].join("-");
}

function getMoscowDateParts(date: Date) {
    if (Number.isNaN(date.getTime())) {
        throw new Error("Invalid date");
    }

    const formatter = new Intl.DateTimeFormat(
        "en-GB",
        {
            timeZone: MOSCOW_TIME_ZONE,
            year: "numeric",
            month: "2-digit",
            day: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
            hour12: false,
        },
    );

    const values = Object.fromEntries(
        formatter
            .formatToParts(date)
            .filter(
                (part) => part.type !== "literal",
            )
            .map((part) => [
                part.type,
                part.value,
            ]),
    );

    return {
        year: values.year,
        month: values.month,
        day: values.day,
        hour: values.hour,
        minute: values.minute,
    };
}

function padDatePart(
    value: number,
): string {
    return String(value).padStart(2, "0");
}

export default SchedulePage;