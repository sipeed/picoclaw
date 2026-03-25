/**
 * Listens for uncaught exceptions coming from Happy DOM on the running Node process and dispatches error events on the Window instance.
 */
export default class BrowserExceptionObserver {
    static listenerCount = 0;
    observedWindows = [];
    uncaughtExceptionListener = null;
    uncaughtRejectionListener = null;
    /**
     * Observes the Node process for uncaught exceptions.
     *
     * @param window Browser window.
     */
    observe(window) {
        if (this.observedWindows.includes(window)) {
            throw new Error('Browser window is already being observed.');
        }
        this.observedWindows.push(window);
        if (this.uncaughtExceptionListener) {
            return;
        }
        this.uncaughtExceptionListener = (error, origin) => {
            if (origin === 'unhandledRejection') {
                return;
            }
            let targetWindow = null;
            for (const window of this.observedWindows) {
                if (error instanceof window.Error || error instanceof window.DOMException) {
                    targetWindow = window;
                    break;
                }
            }
            if (targetWindow) {
                targetWindow.console.error(error);
                targetWindow.dispatchEvent(new targetWindow.ErrorEvent('error', {
                    error: error,
                    message: error.message
                }));
            }
            else if (process.listenerCount('uncaughtException') ===
                this.constructor.listenerCount) {
                // eslint-disable-next-line no-console
                console.error(error);
                // Exit if there are no other listeners handling the error.
                process.exit(1);
            }
        };
        // The "uncaughtException" event is not always triggered for unhandled rejections.
        // Therefore we want to use the "unhandledRejection" event as well.
        this.uncaughtRejectionListener = (error) => {
            let targetWindow = null;
            for (const window of this.observedWindows) {
                if (error instanceof window.Error || error instanceof window.DOMException) {
                    targetWindow = window;
                    break;
                }
            }
            if (targetWindow) {
                targetWindow.console.error(error);
                targetWindow.dispatchEvent(new targetWindow.ErrorEvent('error', {
                    error: error,
                    message: error.message
                }));
            }
            else if (process.listenerCount('unhandledRejection') ===
                this.constructor.listenerCount) {
                // eslint-disable-next-line no-console
                console.error(error);
                // Exit if there are no other listeners handling the error.
                process.exit(1);
            }
        };
        this.constructor.listenerCount++;
        process.on('uncaughtException', this.uncaughtExceptionListener);
        process.on('unhandledRejection', this.uncaughtRejectionListener);
    }
    /**
     * Disconnects observer.
     *
     * @param window Browser window.
     */
    disconnect(window) {
        const index = this.observedWindows.indexOf(window);
        if (index === -1) {
            return;
        }
        this.observedWindows.splice(index, 1);
        if (this.observedWindows.length === 0 && this.uncaughtExceptionListener) {
            this.constructor.listenerCount--;
            process.off('uncaughtException', this.uncaughtExceptionListener);
            if (this.uncaughtRejectionListener) {
                process.off('unhandledRejection', this.uncaughtRejectionListener);
            }
            this.uncaughtExceptionListener = null;
            this.uncaughtRejectionListener = null;
        }
    }
}
//# sourceMappingURL=BrowserExceptionObserver.js.map