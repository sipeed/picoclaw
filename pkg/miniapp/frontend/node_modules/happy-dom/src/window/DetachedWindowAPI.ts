import VirtualConsolePrinter from '../console/VirtualConsolePrinter.js';
import IOptionalBrowserPageViewport from '../browser/types/IOptionalBrowserPageViewport.js';
import IBrowserFrame from '../browser/types/IBrowserFrame.js';
import IBrowserSettings from '../browser/types/IBrowserSettings.js';

/**
 * API for detached windows to be able to access features of the browser.
 */
export default class DetachedWindowAPI {
	#browserFrame?: IBrowserFrame;

	/**
	 * Constructor.
	 *
	 * @param browserFrame Browser frame.
	 */
	constructor(browserFrame: IBrowserFrame) {
		this.#browserFrame = browserFrame;
	}

	/**
	 * Returns settings.
	 *
	 * @returns Settings.
	 */
	public get settings(): IBrowserSettings {
		return this.#browserFrame.page.context.browser.settings;
	}

	/**
	 * Returns virtual console printer.
	 *
	 * @returns Virtual console printer.
	 */
	public get virtualConsolePrinter(): VirtualConsolePrinter {
		return this.#browserFrame.page.virtualConsolePrinter;
	}

	/**
	 * Waits for all async tasks to complete.
	 *
	 * @returns Promise.
	 */
	public waitUntilComplete(): Promise<void> {
		return this.#browserFrame.waitUntilComplete();
	}

	/**
	 * Waits for all async tasks to complete.
	 *
	 * @deprecated Use waitUntilComplete() instead.
	 * @returns Promise.
	 */
	public whenAsyncComplete(): Promise<void> {
		return this.waitUntilComplete();
	}

	/**
	 * Aborts all async tasks.
	 */
	public abort(): Promise<void> {
		return this.#browserFrame.abort();
	}

	/**
	 * Aborts all async tasks.
	 *
	 * @deprecated Use abort() instead.
	 */
	public cancelAsync(): Promise<void> {
		return this.abort();
	}

	/**
	 * Aborts all async tasks and closes the window.
	 */
	public close(): Promise<void> {
		return this.#browserFrame.page.close();
	}

	/**
	 * Sets the URL without navigating the browser.
	 *
	 * @param url URL.
	 */
	public setURL(url: string): void {
		this.#browserFrame.url = url;
	}

	/**
	 * Sets the viewport.
	 *
	 * @param viewport Viewport.
	 */
	public setViewport(viewport: IOptionalBrowserPageViewport): void {
		this.#browserFrame.page.setViewport(viewport);
	}

	/**
	 * Sets the window size.
	 *
	 * @deprecated Use setViewport() instead.
	 * @param options Options.
	 * @param options.width Width.
	 * @param options.height Height.
	 */
	public setWindowSize(options: { width?: number; height?: number }): void {
		this.setViewport({
			width: options?.width,
			height: options?.height
		});
	}

	/**
	 * Sets the window width.
	 *
	 * @deprecated Use setViewport() instead.
	 * @param width Width.
	 */
	public setInnerWidth(width: number): void {
		this.setViewport({ width });
	}

	/**
	 * Sets the window height.
	 *
	 * @deprecated Use setViewport() instead.
	 * @param height Height.
	 */
	public setInnerHeight(height: number): void {
		this.setViewport({ height });
	}
}
