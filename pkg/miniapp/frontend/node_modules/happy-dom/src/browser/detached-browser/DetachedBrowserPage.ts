import VirtualConsolePrinter from '../../console/VirtualConsolePrinter.js';
import DetachedBrowserFrame from './DetachedBrowserFrame.js';
import DetachedBrowserContext from './DetachedBrowserContext.js';
import VirtualConsole from '../../console/VirtualConsole.js';
import IBrowserPage from '../types/IBrowserPage.js';
import { Script } from 'vm';
import IGoToOptions from '../types/IGoToOptions.js';
import Response from '../../fetch/Response.js';
import BrowserPageUtility from '../utilities/BrowserPageUtility.js';
import IReloadOptions from '../types/IReloadOptions.js';
import DefaultBrowserPageViewport from '../DefaultBrowserPageViewport.js';
import IOptionalBrowserPageViewport from '../types/IOptionalBrowserPageViewport.js';
import IBrowserPageViewport from '../types/IBrowserPageViewport.js';
import Event from '../../event/Event.js';

/**
 * Detached browser page used when constructing a Window instance without a browser.
 */
export default class DetachedBrowserPage implements IBrowserPage {
	public readonly virtualConsolePrinter = new VirtualConsolePrinter();
	public readonly mainFrame: DetachedBrowserFrame;
	public readonly context: DetachedBrowserContext;
	public readonly console: Console;
	public readonly viewport: IBrowserPageViewport = Object.assign({}, DefaultBrowserPageViewport);

	/**
	 * Constructor.
	 *
	 * @param context Browser context.
	 */
	constructor(context: DetachedBrowserContext) {
		this.context = context;
		this.console = context.browser.console ?? new VirtualConsole(this.virtualConsolePrinter);
		this.mainFrame = new DetachedBrowserFrame(this);
	}

	/**
	 * Returns frames.
	 */
	public get frames(): DetachedBrowserFrame[] {
		return <DetachedBrowserFrame[]>BrowserPageUtility.getFrames(this);
	}

	/**
	 * Returns the viewport.
	 */
	public get content(): string {
		return this.mainFrame.content;
	}

	/**
	 * Sets the content.
	 *
	 * @param content Content.
	 */
	public set content(content) {
		this.mainFrame.content = content;
	}

	/**
	 * Returns the URL.
	 *
	 * @returns URL.
	 */
	public get url(): string {
		return this.mainFrame.url;
	}

	/**
	 * Sets the content.
	 *
	 * @param url URL.
	 */
	public set url(url) {
		this.mainFrame.url = url;
	}

	/**
	 * Aborts all ongoing operations and destroys the page.
	 */
	public close(): Promise<void> {
		// Using Promise instead of async/await to prevent microtask
		return new Promise((resolve, reject) => {
			const context = this.context;
			BrowserPageUtility.closePage(this)
				.then(() => {
					// As we are in a detached page, a context or browser should not exist without a page as there are no references to them.
					if (context.pages[0] === this) {
						context.close().then(resolve).catch(reject);
					} else {
						resolve();
					}
				})
				.catch(reject);
		});
	}

	/**
	 * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
	 */
	public waitUntilComplete(): Promise<void> {
		return this.mainFrame.waitUntilComplete();
	}

	/**
	 * Returns a promise that is resolved when the page has navigated and the response HTML has been written to the document.
	 */
	public waitForNavigation(): Promise<void> {
		return this.mainFrame.waitForNavigation();
	}

	/**
	 * Aborts all ongoing operations.
	 */
	public abort(): Promise<void> {
		return this.mainFrame.abort();
	}

	/**
	 * Evaluates code or a VM Script in the page's context.
	 *
	 * @param script Script.
	 * @returns Result.
	 */
	public evaluate(script: string | Script): any {
		return this.mainFrame.evaluate(script);
	}

	/**
	 * Sets the viewport.
	 *
	 * @param viewport Viewport.
	 */
	public setViewport(viewport: IOptionalBrowserPageViewport): void {
		const previousViewport = Object.assign({}, this.viewport);
		Object.assign(this.viewport, viewport);
		if (
			previousViewport.width !== this.viewport.width ||
			previousViewport.height !== this.viewport.height ||
			previousViewport.devicePixelRatio !== this.viewport.devicePixelRatio
		) {
			this.mainFrame.window.dispatchEvent(new Event('resize'));
		}
	}

	/**
	 * Go to a page.
	 *
	 * @param url URL.
	 * @param [options] Options.
	 * @returns Response.
	 */
	public goto(url: string, options?: IGoToOptions): Promise<Response | null> {
		return this.mainFrame.goto(url, options);
	}

	/**
	 * Navigates back in history.
	 *
	 * @param [options] Options.
	 */
	public goBack(options?: IGoToOptions): Promise<Response | null> {
		return this.mainFrame.goBack(options);
	}

	/**
	 * Navigates forward in history.
	 *
	 * @param [options] Options.
	 */
	public goForward(options?: IGoToOptions): Promise<Response | null> {
		return this.mainFrame.goForward(options);
	}

	/**
	 * Navigates a delta in history.
	 *
	 * @param delta Delta.
	 * @param steps
	 * @param [options] Options.
	 */
	public goSteps(steps?: number, options?: IGoToOptions): Promise<Response | null> {
		return this.mainFrame.goSteps(steps, options);
	}

	/**
	 * Reloads the current page.
	 *
	 * @param [options] Options.
	 * @returns Response.
	 */
	public reload(options?: IReloadOptions): Promise<Response | null> {
		return this.mainFrame.reload(options);
	}
}
