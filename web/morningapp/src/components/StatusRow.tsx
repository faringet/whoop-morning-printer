export type StatusAccent =
    | "magenta"
    | "cyan"
    | "amber"
    | "success"
    | "danger";

type StatusRowProps = {
    label: string;
    value: string;
    accent: StatusAccent;
};

function StatusRow({
                       label,
                       value,
                       accent,
                   }: StatusRowProps) {
    return (
        <div className="status-row">
      <span className="status-row__label">
        {label}
      </span>

            <strong
                className={`status-row__value status-row__value--${accent}`}
            >
                {value}
            </strong>
        </div>
    );
}

export default StatusRow;