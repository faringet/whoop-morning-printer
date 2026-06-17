export type WakePlanStatus =
  | "scheduled"
  | "wake_receipt_ready"
  | "wake_receipt_printed"
  | "waiting_whoop"
  | "waiting_advice"
  | "final_report_ready"
  | "final_report_printed"
  | "fallback_printed"
  | "done"
  | "cancelled"
  | "failed";

export type WakePlanSource =
  | "manual"
  | "telegram"
  | "default"
  | "test";

export type WakePlan = Readonly<{
  id: number;
  userId: number;

  wakeAt: string;
  prepareAt: string;
  firstReceiptAt: string;
  finalDeadlineAt: string;

  status: WakePlanStatus;
  source: WakePlanSource;

  createdAt: string;
  updatedAt: string;
}>;

export type SaveWakePlanInput = Readonly<{
  wakeAt: string;
}>;

export function wakePlanStatusLabel(
  status: WakePlanStatus,
): string {
  switch (status) {
    case "scheduled":
      return "Scheduled";

    case "wake_receipt_ready":
      return "Wake receipt ready";

    case "wake_receipt_printed":
      return "Wake receipt printed";

    case "waiting_whoop":
      return "Waiting for WHOOP";

    case "waiting_advice":
      return "Waiting for advice";

    case "final_report_ready":
      return "Final report ready";

    case "final_report_printed":
      return "Final report printed";

    case "fallback_printed":
      return "Fallback printed";

    case "done":
      return "Done";

    case "cancelled":
      return "Cancelled";

    case "failed":
      return "Failed";
  }
}
