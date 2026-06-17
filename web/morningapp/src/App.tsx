import {
  useEffect,
  useState,
} from "react";

import {
  getReadableErrorMessage,
} from "./api/httpError";
import {
  wakePlanService,
} from "./api/wakePlanServiceFactory";
import {
  appViews,
  defaultAppView,
  type AppView,
} from "./app/view";
import CancelScheduleDialog from "./components/CancelScheduleDialog";
import EmptyWakePlanPanel from "./components/EmptyWakePlanPanel";
import ErrorWakePlanPanel from "./components/ErrorWakePlanPanel";
import LoadingWakePlanPanel from "./components/LoadingWakePlanPanel";
import SignalFlow from "./components/SignalFlow";
import SystemStatus from "./components/SystemStatus";
import WakePlanPanel from "./components/WakePlanPanel";
import type {
  SaveWakePlanInput,
  WakePlan,
} from "./model/wakePlan";
import SchedulePage from "./pages/SchedulePage";
import { logTelegramRuntime } from "./telegram/telegramDiagnostics";
import {
  notifyCancelDialogOpened,
  notifyOperationFailed,
  notifyScheduleOpened,
  notifyWakePlanCancelled,
  notifyWakePlanSaved,
} from "./telegram/telegramFeedback";
import { useTelegramApp } from "./telegram/useTelegramApp";
import {
  useTelegramBackButton,
} from "./telegram/useTelegramBackButton";
import {
  useTelegramClosingConfirmation,
} from "./telegram/useTelegramClosingConfirmation";
import "./styles/app.css";
import "./styles/cancelDialog.css";
import "./styles/emptyWakePlan.css";
import "./styles/errorWakePlan.css";
import "./styles/schedule.css";

function App() {
  const telegramAppState = useTelegramApp();

  const [currentView, setCurrentView] =
    useState<AppView>(defaultAppView);

  const [wakePlan, setWakePlan] =
    useState<WakePlan | null>(null);

  const [isLoading, setIsLoading] =
    useState(true);

  const [loadError, setLoadError] =
    useState<string | null>(null);

  const [
    isCancelDialogOpen,
    setIsCancelDialogOpen,
  ] = useState(false);

  const [isCancelling, setIsCancelling] =
    useState(false);

  const [cancelError, setCancelError] =
    useState<string | null>(null);

  useEffect(() => {
    logTelegramRuntime(telegramAppState);
  }, [telegramAppState]);

  useEffect(() => {
    let ignoreResult = false;

    wakePlanService
      .getCurrent()
      .then((currentWakePlan) => {
        if (ignoreResult) {
          return;
        }

        setWakePlan(currentWakePlan);
        setLoadError(null);
      })
      .catch((error: unknown) => {
        console.error(
          "Failed to load current wake plan",
          error,
        );

        if (ignoreResult) {
          return;
        }

        notifyOperationFailed();

        setLoadError(
          getReadableErrorMessage(
            error,
            "Could not read the current wake schedule.",
          ),
        );
      })
      .finally(() => {
        if (!ignoreResult) {
          setIsLoading(false);
        }
      });

    return () => {
      ignoreResult = true;
    };
  }, []);

  function handleOpenSchedule() {
    notifyScheduleOpened();
    setCurrentView(appViews.schedule);
  }

  function handleBackToTonight() {
    setCurrentView(appViews.tonight);
  }

  useTelegramBackButton({
    isVisible:
      currentView === appViews.schedule,
    onBack: handleBackToTonight,
  });

  useTelegramClosingConfirmation(
    currentView === appViews.schedule ||
      isCancelDialogOpen ||
      isCancelling,
  );

  async function handleSaveMorning(
    input: SaveWakePlanInput,
  ): Promise<void> {
    try {
      const savedWakePlan =
        await wakePlanService.save(input);

      setWakePlan(savedWakePlan);
      setLoadError(null);
      setCurrentView(appViews.tonight);

      notifyWakePlanSaved();
    } catch (error) {
      notifyOperationFailed();

      throw error;
    }
  }

  function handleOpenCancelDialog() {
    if (!wakePlan || isCancelling) {
      return;
    }

    setCancelError(null);
    setIsCancelDialogOpen(true);

    notifyCancelDialogOpened();
  }

  function handleCloseCancelDialog() {
    if (isCancelling) {
      return;
    }

    setCancelError(null);
    setIsCancelDialogOpen(false);
  }

  async function handleConfirmCancel(): Promise<void> {
    if (!wakePlan || isCancelling) {
      return;
    }

    setIsCancelling(true);
    setCancelError(null);

    try {
      await wakePlanService.cancel(
        wakePlan.id,
      );

      setWakePlan(null);
      setIsCancelDialogOpen(false);
      setCancelError(null);

      notifyWakePlanCancelled();
    } catch (error) {
      console.error(
        "Failed to cancel wake plan",
        error,
      );

      notifyOperationFailed();

      setCancelError(
        getReadableErrorMessage(
          error,
          "The wake plan is still active. Please try again.",
        ),
      );
    } finally {
      setIsCancelling(false);
    }
  }

  async function handleRetryLoad(): Promise<void> {
    setIsLoading(true);
    setLoadError(null);

    try {
      const currentWakePlan =
        await wakePlanService.getCurrent();

      setWakePlan(currentWakePlan);
    } catch (error) {
      console.error(
        "Failed to reload current wake plan",
        error,
      );

      notifyOperationFailed();

      setLoadError(
        getReadableErrorMessage(
          error,
          "Could not read the current wake schedule.",
        ),
      );
    } finally {
      setIsLoading(false);
    }
  }

  if (currentView === appViews.schedule) {
    return (
      <SchedulePage
        wakePlan={wakePlan}
        onBack={handleBackToTonight}
        onSave={handleSaveMorning}
      />
    );
  }

  return (
    <>
      <main className="app-shell">
        <header className="station-header">
          <div className="station-heading">
            <p className="terminal-label station-kicker">
              WHOOP MORNING PRINTER
            </p>

            <h1 className="station-title">
              Morning Station
            </h1>
          </div>

          <SystemStatus
            isLoading={isLoading}
            hasError={loadError !== null}
          />
        </header>

        {isLoading ? (
          <LoadingWakePlanPanel />
        ) : loadError ? (
          <ErrorWakePlanPanel
            message={loadError}
            isRetrying={false}
            onRetry={handleRetryLoad}
          />
        ) : wakePlan ? (
          <WakePlanPanel
            wakePlan={wakePlan}
            onEdit={handleOpenSchedule}
          />
        ) : (
          <EmptyWakePlanPanel
            onCreate={handleOpenSchedule}
          />
        )}

        <SignalFlow />

        {!isLoading &&
        !loadError &&
        wakePlan ? (
          <div className="station-actions">
            <button
              className="button button-primary"
              type="button"
              onClick={handleOpenSchedule}
            >
              Edit morning
            </button>

            <button
              className="button button-ghost"
              type="button"
              onClick={handleOpenCancelDialog}
            >
              Cancel schedule
            </button>
          </div>
        ) : null}

        <footer className="station-footer">
          {isLoading ? (
            <>
              <div>Reading wake schedule</div>
              <div>
                Morning system initializing
              </div>
            </>
          ) : loadError ? (
            <>
              <div>Connection fault detected</div>
              <div>
                Wake state requires retry
              </div>
            </>
          ) : wakePlan ? (
            <>
              <div>Receipt machine ready</div>
              <div>
                Phone optional · Morning armed
              </div>
            </>
          ) : (
            <>
              <div>Awaiting wake signal</div>
              <div>
                Configure your next morning
              </div>
            </>
          )}
        </footer>
      </main>

      <CancelScheduleDialog
        isOpen={isCancelDialogOpen}
        isCancelling={isCancelling}
        errorMessage={cancelError}
        onClose={handleCloseCancelDialog}
        onConfirm={handleConfirmCancel}
      />
    </>
  );
}

export default App;