import AbortSignal from './AbortSignal.js';
import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * AbortController.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/AbortController
 */
export default class AbortController {
	// Injected by WindowContextClassExtender
	protected declare [PropertySymbol.window]: BrowserWindow;

	// Public properties
	public readonly signal: AbortSignal = new this[PropertySymbol.window].AbortSignal();

	/**
	 * Aborts the signal.
	 *
	 * @param [reason] Reason.
	 */
	public abort(reason?: Error): void {
		this.signal[PropertySymbol.abort](reason);
	}
}
