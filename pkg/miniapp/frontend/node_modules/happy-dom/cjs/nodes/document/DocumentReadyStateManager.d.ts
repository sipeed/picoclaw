import BrowserWindow from '../../window/BrowserWindow.cjs';
/**
 * Document ready state manager.
 */
export default class DocumentReadyStateManager {
    private totalTasks;
    private readyStateCallbacks;
    private window;
    private immediate;
    private isComplete;
    /**
     * Constructor.
     *
     * @param window
     */
    constructor(window: BrowserWindow);
    /**
     * Returns a promise that is fulfilled when ready state is complete.
     *
     * @returns Promise.
     */
    waitUntilComplete(): Promise<void>;
    /**
     * Starts a task.
     */
    startTask(): void;
    /**
     * Ends a task.
     */
    endTask(): void;
}
//# sourceMappingURL=DocumentReadyStateManager.d.ts.map