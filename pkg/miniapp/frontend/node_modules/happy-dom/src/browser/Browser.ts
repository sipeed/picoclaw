import IBrowserSettings from './types/IBrowserSettings.js';
import BrowserContext from './BrowserContext.js';
import IOptionalBrowserSettings from './types/IOptionalBrowserSettings.js';
import BrowserSettingsFactory from './BrowserSettingsFactory.js';
import BrowserPage from './BrowserPage.js';
import IBrowser from './types/IBrowser.js';
import BrowserExceptionObserver from './utilities/BrowserExceptionObserver.js';
import * as PropertySymbol from '../PropertySymbol.js';
import BrowserErrorCaptureEnum from './enums/BrowserErrorCaptureEnum.js';

/**
 * Browser.
 *
 * Much of the interface for the browser has been taken from Puppeteer and Playwright, so that the API is familiar.
 */
export default class Browser implements IBrowser {
	public readonly contexts: BrowserContext[];
	public readonly settings: IBrowserSettings;
	public readonly console: Console | null;
	public [PropertySymbol.exceptionObserver]: BrowserExceptionObserver | null = null;

	/**
	 * Constructor.
	 *
	 * @param [options] Options.
	 * @param [options.settings] Browser settings.
	 * @param [options.console] Console.
	 */
	constructor(options?: { settings?: IOptionalBrowserSettings; console?: Console }) {
		this.console = options?.console || null;
		this.settings = BrowserSettingsFactory.createSettings(options?.settings);
		if (this.settings.errorCapture === BrowserErrorCaptureEnum.processLevel) {
			this[PropertySymbol.exceptionObserver] = new BrowserExceptionObserver();
		}
		this.contexts = [new BrowserContext(this)];
	}

	/**
	 * Returns the default context.
	 *
	 * @returns Default context.
	 */
	public get defaultContext(): BrowserContext {
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
		(<BrowserContext[]>this.contexts) = [];
	}

	/**
	 * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
	 *
	 * @returns Promise.
	 */
	public async waitUntilComplete(): Promise<void> {
		if (this.contexts.length === 0) {
			throw new Error('No default context. The browser has been closed.');
		}
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
	 *
	 * @returns Context.
	 */
	public newIncognitoContext(): BrowserContext {
		if (this.contexts.length === 0) {
			throw new Error('No default context. The browser has been closed.');
		}
		const context = new BrowserContext(this);
		this.contexts.push(context);
		return context;
	}

	/**
	 * Creates a new page.
	 *
	 * @returns Page.
	 */
	public newPage(): BrowserPage {
		if (this.contexts.length === 0) {
			throw new Error('No default context. The browser has been closed.');
		}
		return this.contexts[0].newPage();
	}
}
