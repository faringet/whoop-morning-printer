import { appConfig } from "../config/appConfig";
import {
    mockWakePlanService,
} from "../mocks/MockWakePlanService";
import {
    httpWakePlanService,
} from "./HttpWakePlanService";
import type {
    WakePlanService,
} from "./wakePlanService";

export const wakePlanService: WakePlanService =
    createWakePlanService();

export function createWakePlanService():
    WakePlanService {
    switch (appConfig.wakePlanServiceMode) {
        case "mock":
            return mockWakePlanService;

        case "http":
            return httpWakePlanService;

        default:
            return assertNever(
                appConfig.wakePlanServiceMode,
            );
    }
}

function assertNever(
    value: never,
): never {
    throw new Error(
        `Unsupported wake plan service mode: ${String(value)}`,
    );
}