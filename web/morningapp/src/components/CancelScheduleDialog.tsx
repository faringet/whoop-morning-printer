import {
    type KeyboardEvent,
    type MouseEvent,
    useEffect,
    useRef,
} from "react";

type CancelScheduleDialogProps = {
    isOpen: boolean;
    isCancelling: boolean;
    errorMessage: string | null;
    onClose: () => void;
    onConfirm: () => Promise<void>;
};

function CancelScheduleDialog({
                                  isOpen,
                                  isCancelling,
                                  errorMessage,
                                  onClose,
                                  onConfirm,
                              }: CancelScheduleDialogProps) {
    const keepButtonRef =
        useRef<HTMLButtonElement | null>(null);

    useEffect(() => {
        if (!isOpen) {
            return;
        }

        const previousActiveElement =
            document.activeElement;

        keepButtonRef.current?.focus();

        return () => {
            if (
                previousActiveElement instanceof HTMLElement
            ) {
                previousActiveElement.focus();
            }
        };
    }, [isOpen]);

    useEffect(() => {
        if (!isOpen) {
            return;
        }

        const previousOverflow =
            document.body.style.overflow;

        document.body.style.overflow = "hidden";

        return () => {
            document.body.style.overflow =
                previousOverflow;
        };
    }, [isOpen]);

    if (!isOpen) {
        return null;
    }

    function handleBackdropClick() {
        if (isCancelling) {
            return;
        }

        onClose();
    }

    function handleDialogClick(
        event: MouseEvent<HTMLDivElement>,
    ) {
        event.stopPropagation();
    }

    function handleKeyDown(
        event: KeyboardEvent<HTMLDivElement>,
    ) {
        if (
            event.key === "Escape" &&
            !isCancelling
        ) {
            onClose();
        }
    }

    async function handleConfirm() {
        if (isCancelling) {
            return;
        }

        await onConfirm();
    }

    const descriptionIds = [
        "cancel-dialog-description",
        errorMessage
            ? "cancel-dialog-error"
            : null,
    ]
        .filter(Boolean)
        .join(" ");

    return (
        <div
            className="cancel-dialog-backdrop"
            role="presentation"
            onClick={handleBackdropClick}
        >
            <div
                className="cancel-dialog panel panel-magenta"
                role="dialog"
                aria-modal="true"
                aria-labelledby="cancel-dialog-title"
                aria-describedby={descriptionIds}
                aria-busy={isCancelling}
                onClick={handleDialogClick}
                onKeyDown={handleKeyDown}
            >
                <div className="cancel-dialog__signal">
          <span
              className="cancel-dialog__signal-dot"
              aria-hidden="true"
          />

                    <span>Disarm sequence</span>
                </div>

                <div className="cancel-dialog__content">
                    <p className="terminal-label text-magenta">
                        Wake plan control
                    </p>

                    <h2
                        className="cancel-dialog__title"
                        id="cancel-dialog-title"
                    >
                        Cancel morning?
                    </h2>

                    <p
                        className="cancel-dialog__description"
                        id="cancel-dialog-description"
                    >
                        The current wake sequence will be
                        removed. Mac mini wake and receipt
                        printing will no longer be scheduled.
                    </p>
                </div>

                <div className="cancel-dialog__terminal">
                    <div className="cancel-dialog__terminal-row">
                        <span>Wake plan</span>
                        <strong>Armed</strong>
                    </div>

                    <div className="cancel-dialog__terminal-row">
                        <span>Requested action</span>
                        <strong>Disarm</strong>
                    </div>
                </div>

                {errorMessage ? (
                    <div
                        className="cancel-dialog__error"
                        id="cancel-dialog-error"
                        role="alert"
                    >
            <span
                className="cancel-dialog__error-marker"
                aria-hidden="true"
            >
              !
            </span>

                        <div>
                            <strong>
                                Could not disarm morning
                            </strong>

                            <p>{errorMessage}</p>
                        </div>
                    </div>
                ) : null}

                <div className="cancel-dialog__actions">
                    <button
                        ref={keepButtonRef}
                        className="button button-primary"
                        type="button"
                        disabled={isCancelling}
                        onClick={onClose}
                    >
                        Keep morning
                    </button>

                    <button
                        className="button button-danger"
                        type="button"
                        disabled={isCancelling}
                        onClick={() => {
                            void handleConfirm();
                        }}
                    >
                        {isCancelling
                            ? "Disarming..."
                            : errorMessage
                                ? "Retry disarm"
                                : "Cancel plan"}
                    </button>
                </div>
            </div>
        </div>
    );
}

export default CancelScheduleDialog;