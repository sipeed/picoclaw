import BrowserWindow from '../../window/BrowserWindow.js';

/**
 * Document ready state manager.
 */
export default class DocumentReadyStateManager {
	private totalTasks = 0;
	private readyStateCallbacks: (() => void)[] = [];
	private window: BrowserWindow = null;
	private immediate: NodeJS.Immediate | null = null;
	private isComplete = false;

	/**
	 * Constructor.
	 *
	 * @param window
	 */
	constructor(window: BrowserWindow) {
		this.window = window;
	}

	/**
	 * Returns a promise that is fulfilled when ready state is complete.
	 *
	 * @returns Promise.
	 */
	public waitUntilComplete(): Promise<void> {
		return new Promise((resolve) => {
			if (this.isComplete) {
				resolve();
			} else {
				this.readyStateCallbacks.push(resolve);
				this.startTask();
				this.endTask();
			}
		});
	}

	/**
	 * Starts a task.
	 */
	public startTask(): void {
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
	public endTask(): void {
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
