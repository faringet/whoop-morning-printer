import type {
    SaveWakePlanInput,
    WakePlan,
} from "../model/wakePlan";

export interface WakePlanService {
    getCurrent(): Promise<WakePlan | null>;

    save(
        input: SaveWakePlanInput,
    ): Promise<WakePlan>;

    cancel(
        wakePlanId: number,
    ): Promise<void>;
}