import {
  useEffect,
  useState,
} from "react";

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
import { mockWakePlanService } from "./mocks/MockWakePlanService";
import type {
  SaveWakePlanInput,
  WakePlan,
} from "./model/wakePlan";
import SchedulePage from "./pages/SchedulePage";
import "./styles/app.css";
import "./styles/cancelDialog.css";
import "./styles/emptyWakePlan.css";
import "./styles/errorWakePlan.css";
import "./styles/schedule.css";

function App() {
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
    let ignoreResult = false;

    mockWakePlanService
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

          setLoadError(
              getErrorMessage(
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
    setCurrentView(appViews.schedule);
  }

  function handleBackToTonight() {
    setCurrentView(appViews.tonight);
  }

  async function handleSaveMorning(
      input: SaveWakePlanInput,
  ): Promise<void> {
    /*
     * Ошибку здесь намеренно не перехватываем:
     * SchedulePage обработает её и покажет
     * сообщение пользователю.
     */
    const savedWakePlan =
        await mockWakePlanService.save(input);

    setWakePlan(savedWakePlan);
    setLoadError(null);
    setCurrentView(appViews.tonight);
  }

  function handleOpenCancelDialog() {
    if (!wakePlan || isCancelling) {
      return;
    }

    setCancelError(null);
    setIsCancelDialogOpen(true);
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
      await mockWakePlanService.cancel(
          wakePlan.id,
      );

      setWakePlan(null);
      setIsCancelDialogOpen(false);
      setCancelError(null);
    } catch (error) {
      console.error(
          "Failed to cancel wake plan",
          error,
      );

      setCancelError(
          getErrorMessage(
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
          await mockWakePlanService.getCurrent();

      setWakePlan(currentWakePlan);
    } catch (error) {
      console.error(
          "Failed to reload current wake plan",
          error,
      );

      setLoadError(
          getErrorMessage(
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

function getErrorMessage(
    error: unknown,
    fallback: string,
): string {
  if (
      error instanceof Error &&
      error.message.trim().length > 0
  ) {
    return error.message;
  }

  return fallback;
}

export default App;