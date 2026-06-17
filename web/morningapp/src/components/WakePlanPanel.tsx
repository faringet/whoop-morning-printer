import type {
  WakePlan,
  WakePlanStatus,
} from "../model/wakePlan";
import { wakePlanStatusLabel } from "../model/wakePlan";
import {
  formatWakeDate,
  formatWakeTime,
} from "../model/wakePlanFormatters";
import StatusRow, {
  type StatusAccent,
} from "./StatusRow";

type WakePlanPanelProps = {
  wakePlan: WakePlan;
  onEdit?: () => void;
};

function WakePlanPanel({
  wakePlan,
  onEdit,
}: WakePlanPanelProps) {
  const wakeTime = formatWakeTime(wakePlan.wakeAt);
  const wakeDate = formatWakeDate(wakePlan.wakeAt);

  const macWakeTime = formatWakeTime(
    wakePlan.macWakeAt,
  );

  const firstReceiptTime = formatWakeTime(
    wakePlan.firstReceiptAt,
  );

  const finalReportTime = formatWakeTime(
    wakePlan.finalReportAt,
  );

  const statusLabel = wakePlanStatusLabel(
    wakePlan.status,
  );

  return (
    <section className="panel panel-magenta wake-panel">
      <p className="terminal-label wake-panel__label">
        Next morning signal
      </p>

      <div className="wake-panel__hero">
        <button
          className="wake-panel__time-button"
          type="button"
          aria-label={`Edit wake time ${wakeTime}`}
          onClick={onEdit}
        >
          <span className="wake-panel__time">
            {wakeTime}
          </span>
        </button>

        <p className="wake-panel__date">
          {wakeDate}
        </p>
      </div>

      <div className="wake-panel__status-list">
        <StatusRow
          label="Mac mini wake"
          value={macWakeTime}
          accent="cyan"
        />

        <StatusRow
          label="First receipt"
          value={firstReceiptTime}
          accent="magenta"
        />

        <StatusRow
          label="Final report"
          value={finalReportTime}
          accent="amber"
        />

        <StatusRow
          label="Morning status"
          value={statusLabel}
          accent={statusAccent(wakePlan.status)}
        />
      </div>
    </section>
  );
}

function statusAccent(
  status: WakePlanStatus,
): StatusAccent {
  switch (status) {
    case "scheduled":
      return "success";

    case "processing":
      return "cyan";

    case "completed":
      return "success";

    case "cancelled":
      return "amber";

    case "failed":
      return "danger";
  }
}

export default WakePlanPanel;
