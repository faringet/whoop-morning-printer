import type {
  SaveWakePlanInput,
  WakePlan,
  WakePlanSource,
  WakePlanStatus,
} from "../model/wakePlan";
import { HttpError } from "./httpError";

export type WakePlanDto = {
  id: number;
  user_id: number;
  wake_at: string;
  prepare_at: string;
  first_receipt_at: string;
  final_deadline_at: string;
  status: WakePlanStatus;
  source: WakePlanSource;
  created_at: string;
  updated_at: string;
};

export type SaveWakePlanRequestDto = {
  wake_at: string;
};

export function toSaveWakePlanRequestDto(
    input: SaveWakePlanInput,
): SaveWakePlanRequestDto {
  return {
    wake_at: input.wakeAt,
  };
}

export function parseWakePlanDto(
    value: unknown,
): WakePlan {
  const dto = readWakePlanDto(value);

  return {
    id: dto.id,
    userId: dto.user_id,
    wakeAt: dto.wake_at,
    prepareAt: dto.prepare_at,
    firstReceiptAt: dto.first_receipt_at,
    finalDeadlineAt: dto.final_deadline_at,
    status: dto.status,
    source: dto.source,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  };
}

function readWakePlanDto(
    value: unknown,
): WakePlanDto {
  const record = requireRecord(value);

  return {
    id: requirePositiveInteger(
        record.id,
        "id",
    ),
    user_id: requirePositiveInteger(
        record.user_id,
        "user_id",
    ),
    wake_at: requireIsoDateString(
        record.wake_at,
        "wake_at",
    ),
    prepare_at: requireIsoDateString(
        record.prepare_at,
        "prepare_at",
    ),
    first_receipt_at: requireIsoDateString(
        record.first_receipt_at,
        "first_receipt_at",
    ),
    final_deadline_at: requireIsoDateString(
        record.final_deadline_at,
        "final_deadline_at",
    ),
    status: requireWakePlanStatus(
        record.status,
    ),
    source: requireWakePlanSource(
        record.source,
    ),
    created_at: requireIsoDateString(
        record.created_at,
        "created_at",
    ),
    updated_at: requireIsoDateString(
        record.updated_at,
        "updated_at",
    ),
  };
}

function requireRecord(
    value: unknown,
): Record<string, unknown> {
  if (
      typeof value !== "object" ||
      value === null ||
      Array.isArray(value)
  ) {
    throwInvalidResponse(
        "Wake plan response must be an object.",
        value,
    );
  }

  return value as Record<string, unknown>;
}

function requirePositiveInteger(
    value: unknown,
    fieldName: string,
): number {
  if (
      typeof value !== "number" ||
      !Number.isSafeInteger(value) ||
      value <= 0
  ) {
    throwInvalidResponse(
        `Wake plan field "${fieldName}" must be a positive integer.`,
        value,
    );
  }

  return value;
}

function requireIsoDateString(
    value: unknown,
    fieldName: string,
): string {
  if (
      typeof value !== "string" ||
      value.trim().length === 0 ||
      Number.isNaN(Date.parse(value))
  ) {
    throwInvalidResponse(
        `Wake plan field "${fieldName}" must be a valid date string.`,
        value,
    );
  }

  return value;
}

function requireWakePlanStatus(
    value: unknown,
): WakePlanStatus {
  switch (value) {
    case "scheduled":
    case "wake_receipt_ready":
    case "wake_receipt_printed":
    case "waiting_whoop":
    case "waiting_advice":
    case "final_report_ready":
    case "final_report_printed":
    case "fallback_printed":
    case "done":
    case "cancelled":
    case "failed":
      return value;

    default:
      throwInvalidResponse(
          'Wake plan field "status" contains an unsupported value.',
          value,
      );
  }
}

function requireWakePlanSource(
    value: unknown,
): WakePlanSource {
  switch (value) {
    case "manual":
    case "telegram":
    case "default":
    case "test":
      return value;

    default:
      throwInvalidResponse(
          'Wake plan field "source" contains an unsupported value.',
          value,
      );
  }
}

function throwInvalidResponse(
    message: string,
    responseBody: unknown,
): never {
  throw new HttpError({
    kind: "invalid_response",
    message,
    responseBody,
  });
}