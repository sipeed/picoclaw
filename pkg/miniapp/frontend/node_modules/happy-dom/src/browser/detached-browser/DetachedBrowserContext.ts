import DetachedBrowser from './DetachedBrowser.js';
import DetachedBrowserPage from './DetachedBrowserPage.js';
import IBrowserContext from '../types/IBrowserContext.js';
import ICookieContainer from '../../cookie/ICookieContainer.js';
import CookieContainer from '../../cookie/CookieContainer.js';
import ResponseCache from '../../fetch/cache/response/ResponseCache.js';
import IResponseCache from '../../fetch/cache/response/IResponseCache.js';
import IPreflightResponseCache from '../../fetch/cache/preflight/IPreflightResponseCache.js';
import PreflightResponseCache from '../../fetch/cache/preflight/PreflightResponseCache.js';

/**
 * Detached browser context used when constructing a Window instance without a browser.
 */
export default class DetachedBrowserContext implements IBrowserContext {
	public readonly pages: DetachedBrowserPage[];
	public readonly browser: DetachedBrowser;
	public readonly cookieContainer: ICookieContainer = new CookieContainer();
	public readonly responseCache: IResponseCache = new ResponseCache();
	public readonly preflightResponseCache: IPreflightResponseCache = new PreflightResponseCache();

	/**
	 * Constructor.
	 *
	 * @param browser Browser.
	 */
	constructor(browser: DetachedBrowser) {
		this.browser = browser;
		this.pages = [];
		this.pages.push(new DetachedBrowserPage(this));
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
		(<DetachedBrowserPage[]>this.pages) = [];
		(<DetachedBrowser | null>this.browser) = null;
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
	 * @param [opener] Opener.
	 * @returns Page.
	 */
	public newPage(): DetachedBrowserPage {
		const page = new DetachedBrowserPage(this);
		this.pages.push(page);
		return page;
	}
}
