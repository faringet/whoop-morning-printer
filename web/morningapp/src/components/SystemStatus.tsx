type SystemStatusProps = {
    isLoading: boolean;
    hasError: boolean;
};

function SystemStatus({
                          isLoading,
                          hasError,
                      }: SystemStatusProps) {
    let label = "System online";
    let modifierClassName = "";

    if (isLoading) {
        label = "Connecting";
        modifierClassName =
            "system-status--loading";
    } else if (hasError) {
        label = "System fault";
        modifierClassName =
            "system-status--error";
    }

    const className = [
        "system-status",
        modifierClassName,
    ]
        .filter(Boolean)
        .join(" ");

    return (
        <div
            className={className}
            role="status"
            aria-live="polite"
        >
      <span
          className="system-status__dot"
          aria-hidden="true"
      />

            <span>{label}</span>
        </div>
    );
}

export default SystemStatus;