/**
 * Document ready state manager.
 */
export default class DocumentReadyStateManager {
    totalTasks = 0;
    readyStateCallbacks = [];
    window = null;
    immediate = null;
    isComplete = false;
    /**
     * Constructor.
     *
     * @param window
     */
    constructor(window) {
        this.window = window;
    }
    /**
     * Returns a promise that is fulfilled when ready state is complete.
     *
     * @returns Promise.
     */
    waitUntilComplete() {
        return new Promise((resolve) => {
            if (this.isComplete) {
                resolve();
            }
            else {
                this.readyStateCallbacks.push(resolve);
                this.startTask();
                this.endTask();
            }
        });
    }
    /**
     * Starts a task.
     */
    startTask() {
        if (this.isComplete) {
            return;
        }
        if (this.immediate) {
            this.window.cancelAnimationFrame(this.immediate);
            this.immediate = null;
        }
        this.totalTasks++;
    }
    /**
     * Ends a task.
     */
    endTask() {
        if (this.isComplete) {
            return;
        }
        if (this.immediate) {
            this.window.cancelAnimationFrame(this.immediate);
            this.immediate = null;
        }
        this.totalTasks--;
        this.immediate = this.window.requestAnimationFrame(() => {
            this.immediate = null;
            if (this.totalTasks <= 0) {
                const callbacks = this.readyStateCallbacks;
                this.readyStateCallbacks = [];
                this.isComplete = true;
                for (const callback of callbacks) {
                    callback();
                }
            }
        });
    }
}
//# sourceMappingURL=DocumentReadyStateManager.js.map