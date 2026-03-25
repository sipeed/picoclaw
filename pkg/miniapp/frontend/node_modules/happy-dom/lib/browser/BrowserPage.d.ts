import VirtualConsolePrinter from '../console/VirtualConsolePrinter.js';
import BrowserFrame from './BrowserFrame.js';
import BrowserContext from './BrowserContext.js';
import IBrowserPage from './types/IBrowserPage.js';
import { Script } from 'vm';
import IGoToOptions from './types/IGoToOptions.js';
import Response from '../fetch/Response.js';
import IReloadOptions from './types/IReloadOptions.js';
import IBrowserPageViewport from './types/IBrowserPageViewport.js';
import IOptionalBrowserPageViewport from './types/IOptionalBrowserPageViewport.js';
/**
 * Browser page.
 */
export default class BrowserPage implements IBrowserPage {
    readonly virtualConsolePrinter: VirtualConsolePrinter;
    readonly mainFrame: BrowserFrame;
    readonly context: BrowserContext;
    readonly console: Console;
    readonly viewport: IBrowserPageViewport;
    /**
     * Constructor.
     *
     * @param context Browser context.
     */
    constructor(context: BrowserContext);
    /**
     * Returns frames.
     */
    get frames(): BrowserFrame[];
    /**
     * Returns the viewport.
     */
    get content(): string;
    /**
     * Sets the content.
     *
     * @param content Content.
     */
    set content(content: string);
    /**
     * Returns the URL.
     *
     * @returns URL.
     */
    get url(): string;
    /**
     * Sets the content.
     *
     * @param url URL.
     */
    set url(url: string);
    /**
     * Aborts all ongoing operations and destroys the page.
     */
    close(): Promise<void>;
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     */
    waitUntilComplete(): Promise<void>;
    /**
     * Returns a promise that is resolved when the page has navigated and the response HTML has been written to the document.
     */
    waitForNavigation(): Promise<void>;
    /**
     * Aborts all ongoing operations.
     */
    abort(): Promise<void>;
    /**
     * Evaluates code or a VM Script in the page's context.
     *
     * @param script Script.
     * @returns Result.
     */
    evaluate(script: string | Script): any;
    /**
     * Sets the viewport.
     *
     * @param viewport Viewport.
     */
    setViewport(viewport: IOptionalBrowserPageViewport): void;
    /**
     * Go to a page.
     *
     * @param url URL.
     * @param [options] Options.
     * @returns Response.
     */
    goto(url: string, options?: IGoToOptions): Promise<Response | null>;
    /**
     * Navigates back in history.
     *
     * @param [options] Options.
     */
    goBack(options?: IGoToOptions): Promise<Response | null>;
    /**
     * Navigates forward in history.
     *
     * @param [options] Options.
     */
    goForward(options?: IGoToOptions): Promise<Response | null>;
    /**
     * Navigates a delta in history.
     *
     * @param delta Delta.
     * @param steps
     * @param [options] Options.
     */
    goSteps(steps?: number, options?: IGoToOptions): Promise<Response | null>;
    /**
     * Reloads the current page.
     *
     * @param [options] Options.
     * @returns Response.
     */
    reload(options?: IReloadOptions): Promise<Response | null>;
}
//# sourceMappingURL=BrowserPage.d.ts.map