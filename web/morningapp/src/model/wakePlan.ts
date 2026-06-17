export type WakePlanStatus =
  | "scheduled"
  | "processing"
  | "completed"
  | "cancelled"
  | "failed";

export type WakePlan = {
  id: number;
  userId: number;

  wakeAt: string;
  macWakeAt: string;
  firstReceiptAt: string;
  finalReportAt: string;

  status: WakePlanStatus;

  createdAt: string;
  updatedAt: string;
};

/*
 * Пользователь выбирает только дату и время пробуждения.
 *
 * Остальные значения:
 * - macWakeAt;
 * - firstReceiptAt;
 * - finalReportAt;
 *
 * рассчитывает backend на основании системного конфига.
 */
export type SaveWakePlanInput = {
  wakeAt: string;
};

export function isActiveWakePlan(
  wakePlan: WakePlan,
): boolean {
  return (
    wakePlan.status === "scheduled" ||
    wakePlan.status === "processing"
  );
}

export function wakePlanStatusLabel(
  status: WakePlanStatus,
): string {
  switch (status) {
    case "scheduled":
      return "Armed";

    case "processing":
      return "Processing";

    case "completed":
      return "Completed";

    case "cancelled":
      return "Cancelled";

    case "failed":
      return "Failed";
  }
}
