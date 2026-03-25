import BrowserWindow from './BrowserWindow.js';
import * as PropertySymbol from '../PropertySymbol.js';
import ErrorEvent from '../event/events/ErrorEvent.js';
import Element from '../nodes/element/Element.js';

/**
 * Error utility.
 */
export default class WindowErrorUtility {
	/**
	 * Calls a function synchronously wrapped in a try/catch block to capture errors and dispatch error events.
	 * If the callback returns a Promise, it will catch errors from the promise.
	 *
	 * It will also output the errors to the console.
	 *
	 * @param elementOrWindow Element or Window.
	 * @param callback Callback.
	 * @param [cleanup] Cleanup callback on error.
	 * @returns Result.
	 */
	public static captureError<T>(
		elementOrWindow: BrowserWindow | Element,
		callback: () => T,
		cleanup?: () => void
	): T | null {
		let result = null;

		try {
			result = callback();
		} catch (error) {
			this.dispatchError(elementOrWindow, error);
			if (cleanup) {
				cleanup();
			}
		}

		if (result && result instanceof Promise) {
			result.catch((error) => {
				this.dispatchError(elementOrWindow, error);
				if (cleanup) {
					cleanup();
				}
			});
		}

		return result;
	}

	/**
	 * Dispatches an error event and outputs it to the console.
	 *
	 * @param elementOrWindow Element or Window.
	 * @param error Error.
	 */
	public static dispatchError(elementOrWindow: BrowserWindow | Element, error: Error): void {
		if ((<BrowserWindow>elementOrWindow).console) {
			(<BrowserWindow>elementOrWindow).console.error(error);
			elementOrWindow.dispatchEvent(new ErrorEvent('error', { message: error.message, error }));
		} else {
			(<Element>elementOrWindow)[PropertySymbol.ownerDocument][
				PropertySymbol.defaultView
			]?.console.error(error);
			(<Element>elementOrWindow).dispatchEvent(
				new ErrorEvent('error', { message: error.message, error })
			);
		}
	}
}
