// We need to set this as a global constant, so that using fake timers in Jest and Vitest won't override this on the global object.
const TIMER = {
    setTimeout: globalThis.setTimeout.bind(globalThis),
    clearTimeout: globalThis.clearTimeout.bind(globalThis),
    clearImmediate: globalThis.clearImmediate.bind(globalThis)
};
/**
 * Handles async tasks.
 */
export default class AsyncTaskManager {
    static taskID = 0;
    runningTasks = {};
    runningTaskCount = 0;
    runningTimers = [];
    runningImmediates = [];
    waitUntilCompleteTimer = null;
    waitUntilCompleteResolvers = [];
    aborted = false;
    destroyed = false;
    #browserFrame;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame) {
        this.#browserFrame = browserFrame;
    }
    /**
     * Returns a promise that is resolved when async tasks are complete.
     *
     * @returns Promise.
     */
    waitUntilComplete() {
        return new Promise((resolve) => {
            this.waitUntilCompleteResolvers.push(resolve);
            this.resolveWhenComplete();
        });
    }
    /**
     * Aborts all tasks.
     */
    abort() {
        if (this.aborted) {
            return new Promise((resolve) => {
                this.waitUntilCompleteResolvers.push(resolve);
                this.resolveWhenComplete();
            });
        }
        return this.abortAll(false);
    }
    /**
     * Destroys the manager.
     */
    destroy() {
        if (this.aborted) {
            return new Promise((resolve) => {
                this.waitUntilCompleteResolvers.push(resolve);
                this.resolveWhenComplete();
            });
        }
        return this.abortAll(true);
    }
    /**
     * Starts a timer.
     *
     * @param timerID Timer ID.
     */
    startTimer(timerID) {
        if (this.aborted) {
            TIMER.clearTimeout(timerID);
            return;
        }
        if (this.waitUntilCompleteTimer) {
            TIMER.clearTimeout(this.waitUntilCompleteTimer);
            this.waitUntilCompleteTimer = null;
        }
        this.runningTimers.push(timerID);
    }
    /**
     * Ends a timer.
     *
     * @param timerID Timer ID.
     */
    endTimer(timerID) {
        if (this.aborted) {
            TIMER.clearTimeout(timerID);
            return;
        }
        const index = this.runningTimers.indexOf(timerID);
        if (index !== -1) {
            this.runningTimers.splice(index, 1);
            if (!this.runningTaskCount && !this.runningTimers.length && !this.runningImmediates.length) {
                this.resolveWhenComplete();
            }
        }
    }
    /**
     * Starts an immediate.
     *
     * @param immediateID Immediate ID.
     */
    startImmediate(immediateID) {
        if (this.aborted) {
            TIMER.clearImmediate(immediateID);
            return;
        }
        if (this.waitUntilCompleteTimer) {
            TIMER.clearTimeout(this.waitUntilCompleteTimer);
            this.waitUntilCompleteTimer = null;
        }
        this.runningImmediates.push(immediateID);
    }
    /**
     * Ends an immediate.
     *
     * @param immediateID Immediate ID.
     */
    endImmediate(immediateID) {
        if (this.aborted) {
            TIMER.clearImmediate(immediateID);
            return;
        }
        const index = this.runningImmediates.indexOf(immediateID);
        if (index !== -1) {
            this.runningImmediates.splice(index, 1);
            if (!this.runningTaskCount && !this.runningTimers.length && !this.runningImmediates.length) {
                this.resolveWhenComplete();
            }
        }
    }
    /**
     * Starts an async task.
     *
     * @param abortHandler Abort handler.
     * @returns Task ID.
     */
    startTask(abortHandler) {
        if (this.aborted) {
            if (abortHandler) {
                abortHandler(this.destroyed);
            }
            throw new this.#browserFrame.window.Error(`Failed to execute 'startTask()' on 'AsyncTaskManager': The asynchrounous task manager has been aborted.`);
        }
        if (this.waitUntilCompleteTimer) {
            TIMER.clearTimeout(this.waitUntilCompleteTimer);
            this.waitUntilCompleteTimer = null;
        }
        const taskID = this.newTaskID();
        this.runningTasks[taskID] = abortHandler ? abortHandler : () => { };
        this.runningTaskCount++;
        return taskID;
    }
    /**
     * Ends an async task.
     *
     * @param taskID Task ID.
     */
    endTask(taskID) {
        if (this.aborted) {
            return;
        }
        if (this.runningTasks[taskID]) {
            delete this.runningTasks[taskID];
            this.runningTaskCount--;
            if (!this.runningTaskCount && !this.runningTimers.length && !this.runningImmediates.length) {
                this.resolveWhenComplete();
            }
        }
    }
    /**
     * Returns the amount of running tasks.
     *
     * @returns Count.
     */
    getTaskCount() {
        return this.runningTaskCount;
    }
    /**
     * Returns a new task ID.
     *
     * @returns Task ID.
     */
    newTaskID() {
        this.constructor.taskID++;
        return this.constructor.taskID;
    }
    /**
     * Resolves when complete.
     */
    resolveWhenComplete() {
        if (this.runningTaskCount || this.runningTimers.length || this.runningImmediates.length) {
            return;
        }
        if (this.waitUntilCompleteTimer) {
            TIMER.clearTimeout(this.waitUntilCompleteTimer);
            this.waitUntilCompleteTimer = null;
        }
        // It is not possible to detect when all microtasks are complete (such as process.nextTick() or promises).
        // To cater for this we use setTimeout() which has the lowest priority and will be executed last.
        // @see https://nodejs.org/en/learn/asynchronous-work/event-loop-timers-and-nexttick
        this.waitUntilCompleteTimer = TIMER.setTimeout(() => {
            this.waitUntilCompleteTimer = null;
            if (!this.runningTaskCount && !this.runningTimers.length && !this.runningImmediates.length) {
                const resolvers = this.waitUntilCompleteResolvers;
                this.waitUntilCompleteResolvers = [];
                for (const resolver of resolvers) {
                    resolver();
                }
                this.aborted = false;
            }
        }, 1);
    }
    /**
     * Aborts all tasks.
     *
     * @param destroy Destroy.
     */
    abortAll(destroy) {
        const runningTimers = this.runningTimers;
        const runningImmediates = this.runningImmediates;
        const runningTasks = this.runningTasks;
        this.aborted = true;
        this.destroyed = destroy;
        this.runningTasks = {};
        this.runningTaskCount = 0;
        this.runningImmediates = [];
        this.runningTimers = [];
        if (this.waitUntilCompleteTimer) {
            TIMER.clearTimeout(this.waitUntilCompleteTimer);
            this.waitUntilCompleteTimer = null;
        }
        for (const immediate of runningImmediates) {
            TIMER.clearImmediate(immediate);
        }
        for (const timer of runningTimers) {
            TIMER.clearTimeout(timer);
        }
        for (const key of Object.keys(runningTasks)) {
            runningTasks[key](destroy);
        }
        // We need to wait for microtasks to complete before resolving.
        return new Promise((resolve) => {
            this.waitUntilCompleteResolvers.push(resolve);
            this.resolveWhenComplete();
        });
    }
}
//# sourceMappingURL=AsyncTaskManager.js.map