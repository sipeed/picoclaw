import BrowserPage from './BrowserPage.js';
import * as PropertySymbol from '../PropertySymbol.js';
import IBrowserFrame from './types/IBrowserFrame.js';
import BrowserWindow from '../window/BrowserWindow.js';
import CrossOriginBrowserWindow from '../window/CrossOriginBrowserWindow.js';
import Response from '../fetch/Response.js';
import IGoToOptions from './types/IGoToOptions.js';
import { Script } from 'vm';
import IReloadOptions from './types/IReloadOptions.js';
import Document from '../nodes/document/Document.js';
import IHistoryItem from '../history/IHistoryItem.js';
/**
 * Browser frame.
 */
export default class BrowserFrame implements IBrowserFrame {
    readonly childFrames: BrowserFrame[];
    readonly parentFrame: BrowserFrame | null;
    readonly page: BrowserPage;
    readonly window: BrowserWindow;
    [PropertySymbol.asyncTaskManager]: any;
    [PropertySymbol.listeners]: {
        navigation: Array<() => void>;
    };
    [PropertySymbol.openerFrame]: IBrowserFrame | null;
    [PropertySymbol.openerWindow]: BrowserWindow | CrossOriginBrowserWindow | null;
    [PropertySymbol.popup]: boolean;
    [PropertySymbol.history]: IHistoryItem[];
    /**
     * Constructor.
     *
     * @param page Page.
     */
    constructor(page: BrowserPage);
    /**
     * Returns the content.
     *
     * @returns Content.
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
     * Returns document.
     *
     * @returns Document.
     */
    get document(): Document;
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     */
    waitUntilComplete(): Promise<void>;
    /**
     * Returns a promise that is resolved when the frame has navigated and the response HTML has been written to the document.
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
     * @param steps Steps.
     * @param [options] Options.
     */
    goSteps(steps?: number, options?: IGoToOptions): Promise<Response | null>;
    /**
     * Reloads the current frame.
     *
     * @param [options] Options.
     * @returns Response.
     */
    reload(options?: IReloadOptions): Promise<Response | null>;
}
//# sourceMappingURL=BrowserFrame.d.ts.map