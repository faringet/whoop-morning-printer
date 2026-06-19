type ErrorWakePlanPanelProps = {
    isRetrying: boolean;
    onRetry: () => Promise<void>;
    message?: string;
};

function ErrorWakePlanPanel({
                                isRetrying,
                                onRetry,
                                message = "Could not read the current wake schedule.",
                            }: ErrorWakePlanPanelProps) {
    function handleRetry() {
        if (isRetrying) {
            return;
        }

        void onRetry();
    }

    return (
        <section
            className="panel error-wake-panel"
            role="alert"
            aria-live="assertive"
            aria-busy={isRetrying}
        >
            <div className="error-wake-panel__signal">
        <span
            className="error-wake-panel__signal-dot"
            aria-hidden="true"
        />

                <span>Connection fault</span>
            </div>

            <div className="error-wake-panel__content">
                <p className="terminal-label text-danger">
                    Morning system error
                </p>

                <h2 className="error-wake-panel__title">
                    Wake state unavailable
                </h2>

                <p className="error-wake-panel__description">
                    {message}
                </p>
            </div>

            <div className="error-wake-panel__terminal">
                <div className="error-wake-panel__terminal-line">
                    <span>Gateway</span>
                    <strong>Unavailable</strong>
                </div>

                <div className="error-wake-panel__terminal-line">
                    <span>Wake plan</span>
                    <strong>Unknown</strong>
                </div>

                <div className="error-wake-panel__terminal-line">
                    <span>Local state</span>
                    <strong>Preserved</strong>
                </div>
            </div>

            <button
                className="button button-danger error-wake-panel__button"
                type="button"
                disabled={isRetrying}
                onClick={handleRetry}
            >
                {isRetrying
                    ? "Reconnecting..."
                    : "Retry connection"}
            </button>
        </section>
    );
}

export default ErrorWakePlanPanel;