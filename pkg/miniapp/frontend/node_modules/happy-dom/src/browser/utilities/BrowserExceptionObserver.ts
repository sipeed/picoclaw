import BrowserWindow from '../../window/BrowserWindow.js';

/**
 * Listens for uncaught exceptions coming from Happy DOM on the running Node process and dispatches error events on the Window instance.
 */
export default class BrowserExceptionObserver {
	private static listenerCount = 0;
	private observedWindows: BrowserWindow[] = [];
	private uncaughtExceptionListener:
		| ((error: Error, origin: 'uncaughtException' | 'unhandledRejection') => void)
		| null = null;
	private uncaughtRejectionListener: ((error: Error) => void) | null = null;

	/**
	 * Observes the Node process for uncaught exceptions.
	 *
	 * @param window Browser window.
	 */
	public observe(window: BrowserWindow): void {
		if (this.observedWindows.includes(window)) {
			throw new Error('Browser window is already being observed.');
		}

		this.observedWindows.push(window);

		if (this.uncaughtExceptionListener) {
			return;
		}

		this.uncaughtExceptionListener = (
			error: unknown,
			origin: 'uncaughtException' | 'unhandledRejection'
		): void => {
			if (origin === 'unhandledRejection') {
				return;
			}

			let targetWindow: BrowserWindow | null = null;

			for (const window of this.observedWindows) {
				if (error instanceof window.Error || error instanceof window.DOMException) {
					targetWindow = window;
					break;
				}
			}

			if (targetWindow) {
				targetWindow.console.error(error);
				targetWindow.dispatchEvent(
					new targetWindow.ErrorEvent('error', {
						error: <Error>error,
						message: (<Error>error).message
					})
				);
			} else if (
				process.listenerCount('uncaughtException') ===
				(<typeof BrowserExceptionObserver>this.constructor).listenerCount
			) {
				// eslint-disable-next-line no-console
				console.error(error);
				// Exit if there are no other listeners handling the error.
				process.exit(1);
			}
		};

		// The "uncaughtException" event is not always triggered for unhandled rejections.
		// Therefore we want to use the "unhandledRejection" event as well.
		this.uncaughtRejectionListener = (error: unknown): void => {
			let targetWindow: BrowserWindow | null = null;

			for (const window of this.observedWindows) {
				if (error instanceof window.Error || error instanceof window.DOMException) {
					targetWindow = window;
					break;
				}
			}

			if (targetWindow) {
				targetWindow.console.error(error);
				targetWindow.dispatchEvent(
					new targetWindow.ErrorEvent('error', {
						error: <Error>error,
						message: (<Error>error).message
					})
				);
			} else if (
				process.listenerCount('unhandledRejection') ===
				(<typeof BrowserExceptionObserver>this.constructor).listenerCount
			) {
				// eslint-disable-next-line no-console
				console.error(error);
				// Exit if there are no other listeners handling the error.
				process.exit(1);
			}
		};

		(<typeof BrowserExceptionObserver>this.constructor).listenerCount++;
		process.on('uncaughtException', this.uncaughtExceptionListener);
		process.on('unhandledRejection', this.uncaughtRejectionListener);
	}

	/**
	 * Disconnects observer.
	 *
	 * @param window Browser window.
	 */
	public disconnect(window: BrowserWindow): void {
		const index = this.observedWindows.indexOf(window);

		if (index === -1) {
			return;
		}

		this.observedWindows.splice(index, 1);

		if (this.observedWindows.length === 0 && this.uncaughtExceptionListener) {
			(<typeof BrowserExceptionObserver>this.constructor).listenerCount--;
			process.off('uncaughtException', this.uncaughtExceptionListener);
			if (this.uncaughtRejectionListener) {
				process.off('unhandledRejection', this.uncaughtRejectionListener);
			}
			this.uncaughtExceptionListener = null;
			this.uncaughtRejectionListener = null;
		}
	}
}
