import CookieContainer from '../cookie/CookieContainer.js';
import ICookieContainer from '../cookie/ICookieContainer.js';
import IResponseCache from '../fetch/cache/response/IResponseCache.js';
import ResponseCache from '../fetch/cache/response/ResponseCache.js';
import Browser from './Browser.js';
import BrowserPage from './BrowserPage.js';
import IBrowserContext from './types/IBrowserContext.js';
import IPreflightResponseCache from '../fetch/cache/preflight/IPreflightResponseCache.js';
import PreflightResponseCache from '../fetch/cache/preflight/PreflightResponseCache.js';

/**
 * Browser context.
 */
export default class BrowserContext implements IBrowserContext {
	public readonly pages: BrowserPage[] = [];
	public readonly browser: Browser;
	public readonly cookieContainer: ICookieContainer = new CookieContainer();
	public readonly responseCache: IResponseCache = new ResponseCache();
	public readonly preflightResponseCache: IPreflightResponseCache = new PreflightResponseCache();

	/**
	 * Constructor.
	 *
	 * @param browser
	 */
	constructor(browser: Browser) {
		this.browser = browser;
	}

	/**
	 * Aborts all ongoing operations and destroys the context.
	 */
	public async close(): Promise<void> {
		if (!this.browser) {
			return;
		}
		await Promise.all(this.pages.slice().map((page) => page.close()));
		const browser = this.browser;
		const index = browser.contexts.indexOf(this);
		if (index !== -1) {
			browser.contexts.splice(index, 1);
		}
		(<BrowserPage[]>this.pages) = [];
		(<Browser | null>this.browser) = null;
		(<ICookieContainer | null>this.cookieContainer) = null;
		this.responseCache.clear();
		this.preflightResponseCache.clear();
		(<ResponseCache | null>this.responseCache) = null;
		(<PreflightResponseCache | null>this.preflightResponseCache) = null;
		if (browser.contexts.length === 0) {
			browser.close();
		}
	}

	/**
	 * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
	 *
	 * @returns Promise.
	 */
	public async waitUntilComplete(): Promise<void> {
		await Promise.all(this.pages.map((page) => page.waitUntilComplete()));
	}

	/**
	 * Aborts all ongoing operations.
	 */
	public abort(): Promise<void> {
		return new Promise((resolve, reject) => {
			if (!this.pages.length) {
				resolve();
				return;
			}
			Promise.all(this.pages.slice().map((page) => page.abort()))
				.then(() => resolve())
				.catch((error) => reject(error));
		});
	}

	/**
	 * Creates a new page.
	 *
	 * @returns Page.
	 */
	public newPage(): BrowserPage {
		const page = new BrowserPage(this);
		this.pages.push(page);
		return page;
	}
}
