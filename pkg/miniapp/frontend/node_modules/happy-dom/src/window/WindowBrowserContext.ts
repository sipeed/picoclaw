import AsyncTaskManager from '../async-task-manager/AsyncTaskManager.js';
import IBrowser from '../browser/types/IBrowser.js';
import IBrowserContext from '../browser/types/IBrowserContext.js';
import IBrowserFrame from '../browser/types/IBrowserFrame.js';
import IBrowserPage from '../browser/types/IBrowserPage.js';
import IBrowserSettings from '../browser/types/IBrowserSettings.js';
import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from './BrowserWindow.js';

/**
 * API for accessing the Browser in a Window context without exposing the Browser as accessible properties.
 *
 * The Browser should never be exposed to scripts, as the scripts could then manipulate it, which would lead to security issues.
 */
export default class WindowBrowserContext {
	private static [PropertySymbol.browserFrames]: Map<number, IBrowserFrame> = new Map();
	private static [PropertySymbol.windowInternalId] = 0;
	#window: BrowserWindow;

	/**
	 * Browser window.
	 *
	 * @param window Window.
	 */
	constructor(window: BrowserWindow) {
		this.#window = window;
	}

	/**
	 * Returns the browser settings of the window.
	 *
	 * @returns Browser settings.
	 */
	public getSettings(): IBrowserSettings | null {
		return this.getBrowserFrame()?.page?.context?.browser?.settings || null;
	}

	/**
	 * Returns the browser.
	 *
	 * @returns Browser.
	 */
	public getBrowser(): IBrowser | null {
		return this.getBrowserFrame()?.page?.context?.browser || null;
	}

	/**
	 * Returns the browser page.
	 *
	 * @returns Browser page.
	 */
	public getBrowserPage(): IBrowserPage | null {
		return this.getBrowserFrame()?.page || null;
	}

	/**
	 * Returns the browser context.
	 *
	 * @returns Browser context.
	 */
	public getBrowserContext(): IBrowserContext | null {
		return this.getBrowserFrame()?.page?.context || null;
	}

	/**
	 * Returns the browser frame of the window.
	 *
	 * @returns Browser frame.
	 */
	public getBrowserFrame(): IBrowserFrame | null {
		if (!this.#window) {
			return null;
		}
		return (
			(<typeof WindowBrowserContext>this.constructor)[PropertySymbol.browserFrames].get(
				this.#window[PropertySymbol.internalId]
			) || null
		);
	}

	/**
	 * Returns the async task manager of the window.
	 *
	 * @returns Async task manager.
	 */
	public getAsyncTaskManager(): AsyncTaskManager | null {
		return this.getBrowserFrame()?.[PropertySymbol.asyncTaskManager] || null;
	}

	/**
	 * Assigns the window to a browser frame.
	 *
	 * @param window Window.
	 * @param browserFrame Browser frame.
	 */
	public static setWindowBrowserFrameRelation(
		window: BrowserWindow,
		browserFrame: IBrowserFrame
	): void {
		const browserFrames = this[PropertySymbol.browserFrames];
		if (window[PropertySymbol.internalId] === -1) {
			window[PropertySymbol.internalId] = this[PropertySymbol.windowInternalId];
			this[PropertySymbol.windowInternalId]++;
		}
		browserFrames.set(window[PropertySymbol.internalId], browserFrame);
	}

	/**
	 * Assigns the window to a browser frame.
	 *
	 * @param window Window.
	 * @param browserFrame Browser frame.
	 */
	public static removeWindowBrowserFrameRelation(window: BrowserWindow): void {
		this[PropertySymbol.browserFrames].delete(window[PropertySymbol.internalId]);
	}
}
