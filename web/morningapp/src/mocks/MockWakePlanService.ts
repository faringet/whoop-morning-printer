import type { WakePlanService } from "../api/wakePlanService";
import type {
    SaveWakePlanInput,
    WakePlan,
    WakePlanSource,
    WakePlanStatus,
} from "../model/wakePlan";

const MOCK_USER_ID = 1;
const PREPARE_BEFORE_MINUTES = 5;
const FINAL_DEADLINE_AFTER_MINUTES = 90;
const MOCK_REQUEST_DELAY_MS = 650;

const STORAGE_KEY =
    "wmp.morningapp.mock.wake-plan.v1";

const MOCK_ERROR_QUERY_PARAMETER =
    "mockError";

const MOCK_ONCE_SESSION_KEY_PREFIX =
    "wmp.morningapp.mock.error-once";

type MockOperation =
    | "load"
    | "save"
    | "cancel";

type OneTimeMockErrorMode =
    | "load-once"
    | "save-once"
    | "cancel-once";

type MockErrorMode =
    | MockOperation
    | OneTimeMockErrorMode
    | "all";

export class MockWakePlanService
    implements WakePlanService
{
    private currentWakePlan: WakePlan | null;

    constructor(
        initialWakePlan?: WakePlan | null,
    ) {
        if (initialWakePlan !== undefined) {
            this.currentWakePlan =
                cloneWakePlan(initialWakePlan);

            persistWakePlan(this.currentWakePlan);

            return;
        }

        this.currentWakePlan =
            readWakePlanFromStorage();
    }

    async getCurrent():
        Promise<WakePlan | null> {
        await simulateRequestDelay();

        throwMockErrorIfRequested(
            "load",
            "Mock gateway is unavailable.",
        );

        return cloneWakePlan(
            this.currentWakePlan,
        );
    }

    async save(
        input: SaveWakePlanInput,
    ): Promise<WakePlan> {
        await simulateRequestDelay();

        throwMockErrorIfRequested(
            "save",
            "Mock wake plan could not be saved.",
        );

        validateWakeAt(input.wakeAt);

        const now = new Date().toISOString();

        const wakePlan: WakePlan = {
            id:
                this.currentWakePlan?.id ??
                createMockWakePlanId(),

            userId:
                this.currentWakePlan?.userId ??
                MOCK_USER_ID,

            wakeAt: input.wakeAt,

            prepareAt: addMinutes(
                input.wakeAt,
                -PREPARE_BEFORE_MINUTES,
            ),

            firstReceiptAt: input.wakeAt,

            finalDeadlineAt: addMinutes(
                input.wakeAt,
                FINAL_DEADLINE_AFTER_MINUTES,
            ),

            status: "scheduled",
            source: "telegram",

            createdAt:
                this.currentWakePlan?.createdAt ??
                now,

            updatedAt: now,
        };

        this.currentWakePlan = wakePlan;

        persistWakePlan(wakePlan);

        return cloneRequiredWakePlan(wakePlan);
    }

    async cancel(): Promise<void> {
        await simulateRequestDelay();

        throwMockErrorIfRequested(
            "cancel",
            "Mock wake plan could not be cancelled.",
        );

        if (!this.currentWakePlan) {
            return;
        }

        this.currentWakePlan = null;

        persistWakePlan(null);
    }
}

export const mockWakePlanService =
    new MockWakePlanService();

function validateWakeAt(
    wakeAt: string,
): void {
    if (!isValidDateString(wakeAt)) {
        throw new Error(
            `Invalid wake time: ${wakeAt}`,
        );
    }
}

function addMinutes(
    isoValue: string,
    minutes: number,
): string {
    const date = new Date(isoValue);

    if (Number.isNaN(date.getTime())) {
        throw new Error(
            `Invalid ISO date: ${isoValue}`,
        );
    }

    return new Date(
        date.getTime() + minutes * 60_000,
    ).toISOString();
}

function createMockWakePlanId(): number {
    return Date.now();
}

function simulateRequestDelay():
    Promise<void> {
    return new Promise((resolve) => {
        window.setTimeout(
            resolve,
            MOCK_REQUEST_DELAY_MS,
        );
    });
}

function cloneWakePlan(
    wakePlan: WakePlan | null,
): WakePlan | null {
    if (wakePlan === null) {
        return null;
    }

    return {
        ...wakePlan,
    };
}

function cloneRequiredWakePlan(
    wakePlan: WakePlan,
): WakePlan {
    return {
        ...wakePlan,
    };
}

function persistWakePlan(
    wakePlan: WakePlan | null,
): void {
    const storage = getLocalStorage();

    if (!storage) {
        return;
    }

    try {
        if (wakePlan === null) {
            storage.removeItem(STORAGE_KEY);

            return;
        }

        storage.setItem(
            STORAGE_KEY,
            JSON.stringify(wakePlan),
        );
    } catch (error) {
        console.warn(
            "Failed to persist mock wake plan",
            error,
        );
    }
}

function readWakePlanFromStorage():
    WakePlan | null {
    const storage = getLocalStorage();

    if (!storage) {
        return null;
    }

    try {
        const value = storage.getItem(
            STORAGE_KEY,
        );

        if (!value) {
            return null;
        }

        const parsed: unknown =
            JSON.parse(value);

        const wakePlan =
            normalizeStoredWakePlan(parsed);

        if (!wakePlan) {
            storage.removeItem(STORAGE_KEY);

            return null;
        }

        persistWakePlan(wakePlan);

        return cloneRequiredWakePlan(wakePlan);
    } catch (error) {
        console.warn(
            "Failed to read mock wake plan",
            error,
        );

        try {
            storage.removeItem(STORAGE_KEY);
        } catch {
            return null;
        }

        return null;
    }
}

function normalizeStoredWakePlan(
    value: unknown,
): WakePlan | null {
    if (
        typeof value !== "object" ||
        value === null ||
        Array.isArray(value)
    ) {
        return null;
    }

    const candidate =
        value as Record<string, unknown>;

    const id = candidate.id;
    const userId = candidate.userId;
    const wakeAt = candidate.wakeAt;
    const status = candidate.status;
    const createdAt = candidate.createdAt;
    const updatedAt = candidate.updatedAt;

    if (
        typeof id !== "number" ||
        !Number.isSafeInteger(id) ||
        id <= 0 ||
        typeof userId !== "number" ||
        !Number.isSafeInteger(userId) ||
        userId <= 0 ||
        typeof wakeAt !== "string" ||
        !isValidDateString(wakeAt) ||
        !isWakePlanStatus(status) ||
        typeof createdAt !== "string" ||
        !isValidDateString(createdAt) ||
        typeof updatedAt !== "string" ||
        !isValidDateString(updatedAt)
    ) {
        return null;
    }

    const prepareAt =
        readValidDateString(
            candidate.prepareAt,
        ) ??
        readValidDateString(
            candidate.macWakeAt,
        ) ??
        addMinutes(
            wakeAt,
            -PREPARE_BEFORE_MINUTES,
        );

    const firstReceiptAt =
        readValidDateString(
            candidate.firstReceiptAt,
        ) ??
        wakeAt;

    const finalDeadlineAt =
        readValidDateString(
            candidate.finalDeadlineAt,
        ) ??
        readValidDateString(
            candidate.finalReportAt,
        ) ??
        addMinutes(
            wakeAt,
            FINAL_DEADLINE_AFTER_MINUTES,
        );

    const source =
        isWakePlanSource(candidate.source)
            ? candidate.source
            : "telegram";

    return {
        id,
        userId,
        wakeAt,
        prepareAt,
        firstReceiptAt,
        finalDeadlineAt,
        status,
        source,
        createdAt,
        updatedAt,
    };
}

function readValidDateString(
    value: unknown,
): string | null {
    if (
        typeof value !== "string" ||
        !isValidDateString(value)
    ) {
        return null;
    }

    return value;
}

function isValidDateString(
    value: string,
): boolean {
    return !Number.isNaN(
        new Date(value).getTime(),
    );
}

function getLocalStorage():
    Storage | null {
    if (typeof window === "undefined") {
        return null;
    }

    try {
        return window.localStorage;
    } catch {
        return null;
    }
}

function getSessionStorage():
    Storage | null {
    if (typeof window === "undefined") {
        return null;
    }

    try {
        return window.sessionStorage;
    } catch {
        return null;
    }
}

function throwMockErrorIfRequested(
    operation: MockOperation,
    message: string,
): void {
    if (!shouldFailMockOperation(operation)) {
        return;
    }

    throw new Error(message);
}

function shouldFailMockOperation(
    operation: MockOperation,
): boolean {
    if (!import.meta.env.DEV) {
        return false;
    }

    const mode = getMockErrorMode();

    if (!mode) {
        return false;
    }

    if (
        mode === "all" ||
        mode === operation
    ) {
        return true;
    }

    if (!isOneTimeMockErrorMode(mode)) {
        return false;
    }

    if (mode !== `${operation}-once`) {
        return false;
    }

    return consumeOneTimeFailure(mode);
}

function isOneTimeMockErrorMode(
    mode: MockErrorMode,
): mode is OneTimeMockErrorMode {
    return (
        mode === "load-once" ||
        mode === "save-once" ||
        mode === "cancel-once"
    );
}

function getMockErrorMode():
    MockErrorMode | null {
    if (typeof window === "undefined") {
        return null;
    }

    const value = new URLSearchParams(
        window.location.search,
    ).get(MOCK_ERROR_QUERY_PARAMETER);

    if (
        value === "load" ||
        value === "load-once" ||
        value === "save" ||
        value === "save-once" ||
        value === "cancel" ||
        value === "cancel-once" ||
        value === "all"
    ) {
        return value;
    }

    return null;
}

function consumeOneTimeFailure(
    mode: OneTimeMockErrorMode,
): boolean {
    const storage = getSessionStorage();

    if (!storage) {
        return true;
    }

    const key =
        `${MOCK_ONCE_SESSION_KEY_PREFIX}.${mode}`;

    try {
        if (
            storage.getItem(key) === "consumed"
        ) {
            return false;
        }

        storage.setItem(key, "consumed");

        return true;
    } catch {
        return true;
    }
}

function isWakePlanStatus(
    value: unknown,
): value is WakePlanStatus {
    return (
        value === "scheduled" ||
        value === "wake_receipt_ready" ||
        value === "wake_receipt_printed" ||
        value === "waiting_whoop" ||
        value === "waiting_advice" ||
        value === "final_report_ready" ||
        value === "final_report_printed" ||
        value === "fallback_printed" ||
        value === "done" ||
        value === "cancelled" ||
        value === "failed"
    );
}

function isWakePlanSource(
    value: unknown,
): value is WakePlanSource {
    return (
        value === "manual" ||
        value === "telegram" ||
        value === "default" ||
        value === "test"
    );
}