function LoadingWakePlanPanel() {
    return (
        <section
            className="panel panel-cyan empty-wake-panel"
            aria-live="polite"
            aria-busy="true"
        >
            <div className="empty-wake-panel__signal">
        <span
            className="empty-wake-panel__signal-dot"
            aria-hidden="true"
        />

                <span>Reading wake schedule</span>
            </div>

            <div className="empty-wake-panel__content">
                <p className="terminal-label text-cyan">
                    System boot
                </p>

                <h2 className="empty-wake-panel__title">
                    Loading morning state
                </h2>

                <p className="empty-wake-panel__description">
                    Checking the current wake sequence.
                </p>
            </div>

            <div className="empty-wake-panel__terminal">
                <div className="empty-wake-panel__terminal-line">
                    <span>Gateway</span>
                    <strong>Connecting</strong>
                </div>

                <div className="empty-wake-panel__terminal-line">
                    <span>Wake plan</span>
                    <strong>Reading</strong>
                </div>

                <div className="empty-wake-panel__terminal-line">
                    <span>Morning state</span>
                    <strong>Initializing</strong>
                </div>
            </div>
        </section>
    );
}

export default LoadingWakePlanPanel;