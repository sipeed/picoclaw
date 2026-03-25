import IBrowserSettings from '../types/IBrowserSettings.js';
import DetachedBrowserContext from './DetachedBrowserContext.js';
import IOptionalBrowserSettings from '../types/IOptionalBrowserSettings.js';
import BrowserSettingsFactory from '../BrowserSettingsFactory.js';
import DetachedBrowserPage from './DetachedBrowserPage.js';
import IBrowser from '../types/IBrowser.js';
import IBrowserFrame from '../types/IBrowserFrame.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import BrowserErrorCaptureEnum from '../enums/BrowserErrorCaptureEnum.js';
import BrowserExceptionObserver from '../utilities/BrowserExceptionObserver.js';

/**
 * Detached browser used when constructing a Window instance without a browser.
 *
 * Much of the interface for the browser has been taken from Puppeteer and Playwright, so that the API is familiar.
 */
export default class DetachedBrowser implements IBrowser {
	public readonly contexts: DetachedBrowserContext[];
	public readonly settings: IBrowserSettings;
	public readonly console: Console | null;
	public readonly windowClass: new (
		browserFrame: IBrowserFrame,
		options?: { url?: string; width?: number; height?: number }
	) => BrowserWindow | null;
	public [PropertySymbol.exceptionObserver]: BrowserExceptionObserver | null = null;

	/**
	 * Constructor.
	 *
	 * @param windowClass Window class.
	 * @param [options] Options.
	 * @param [options.settings] Browser settings.
	 * @param [options.console] Console.
	 */
	constructor(
		windowClass: new (
			browserFrame: IBrowserFrame,
			options?: { url?: string; width?: number; height?: number }
		) => BrowserWindow,
		options?: { settings?: IOptionalBrowserSettings; console?: Console }
	) {
		this.windowClass = windowClass;
		this.console = options?.console || null;
		this.settings = BrowserSettingsFactory.createSettings(options?.settings);
		if (this.settings.errorCapture === BrowserErrorCaptureEnum.processLevel) {
			this[PropertySymbol.exceptionObserver] = new BrowserExceptionObserver();
		}
		this.contexts = [];
		this.contexts.push(new DetachedBrowserContext(this));
	}

	/**
	 * Returns the default context.
	 *
	 * @returns Default context.
	 */
	public get defaultContext(): DetachedBrowserContext {
		if (this.contexts.length === 0) {
			throw new Error('No default context. The browser has been closed.');
		}
		return this.contexts[0];
	}

	/**
	 * Aborts all ongoing operations and destroys the browser.
	 */
	public async close(): Promise<void> {
		await Promise.all(this.contexts.slice().map((context) => context.close()));
		(<DetachedBrowserContext[]>this.contexts) = [];
		(<Console | null>this.console) = null;
		(<(new (browserFrame: IBrowserFrame) => BrowserWindow) | null>this.windowClass) = null;
	}

	/**
	 * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
	 *
	 * @returns Promise.
	 */
	public async waitUntilComplete(): Promise<void> {
		await Promise.all(this.contexts.map((page) => page.waitUntilComplete()));
	}

	/**
	 * Aborts all ongoing operations.
	 */
	public abort(): Promise<void> {
		// Using Promise instead of async/await to prevent microtask
		return new Promise((resolve, reject) => {
			if (!this.contexts.length) {
				resolve();
				return;
			}
			Promise.all(this.contexts.slice().map((context) => context.abort()))
				.then(() => resolve())
				.catch((error) => reject(error));
		});
	}

	/**
	 * Creates a new incognito context.
	 */
	public newIncognitoContext(): DetachedBrowserContext {
		throw new Error('Not possible to create a new context on a detached browser.');
	}

	/**
	 * Creates a new page.
	 *
	 * @returns Page.
	 */
	public newPage(): DetachedBrowserPage {
		if (this.contexts.length === 0) {
			throw new Error('No default context. The browser has been closed.');
		}
		return this.contexts[0].newPage();
	}
}
