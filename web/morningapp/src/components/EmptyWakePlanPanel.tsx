type EmptyWakePlanPanelProps = {
    onCreate: () => void;
};

function EmptyWakePlanPanel({
                                onCreate,
                            }: EmptyWakePlanPanelProps) {
    return (
        <section className="panel panel-magenta empty-wake-panel">
            <div className="empty-wake-panel__signal">
        <span
            className="empty-wake-panel__signal-dot"
            aria-hidden="true"
        />

                <span>No active wake signal</span>
            </div>

            <div className="empty-wake-panel__content">
                <p className="terminal-label text-magenta">
                    Morning sequence
                </p>

                <h2 className="empty-wake-panel__title">
                    Morning not armed
                </h2>

                <p className="empty-wake-panel__description">
                    Set the time when you want the morning
                    receipt sequence to begin.
                </p>
            </div>

            <div className="empty-wake-panel__terminal">
                <div className="empty-wake-panel__terminal-line">
                    <span>Wake plan</span>
                    <strong>Not set</strong>
                </div>

                <div className="empty-wake-panel__terminal-line">
                    <span>Mac mini</span>
                    <strong>Standby</strong>
                </div>

                <div className="empty-wake-panel__terminal-line">
                    <span>Printer</span>
                    <strong>Waiting</strong>
                </div>
            </div>

            <button
                className="button button-primary empty-wake-panel__button"
                type="button"
                onClick={onCreate}
            >
                Arm morning
            </button>
        </section>
    );
}

export default EmptyWakePlanPanel;