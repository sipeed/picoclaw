import BrowserWindow from '../../window/BrowserWindow.cjs';
/**
 * Listens for uncaught exceptions coming from Happy DOM on the running Node process and dispatches error events on the Window instance.
 */
export default class BrowserExceptionObserver {
    private static listenerCount;
    private observedWindows;
    private uncaughtExceptionListener;
    private uncaughtRejectionListener;
    /**
     * Observes the Node process for uncaught exceptions.
     *
     * @param window Browser window.
     */
    observe(window: BrowserWindow): void;
    /**
     * Disconnects observer.
     *
     * @param window Browser window.
     */
    disconnect(window: BrowserWindow): void;
}
//# sourceMappingURL=BrowserExceptionObserver.d.ts.map