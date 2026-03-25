import DetachedBrowserPage from './DetachedBrowserPage.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import IBrowserFrame from '../types/IBrowserFrame.cjs';
import Response from '../../fetch/Response.cjs';
import IGoToOptions from '../types/IGoToOptions.cjs';
import { Script } from 'vm';
import BrowserWindow from '../../window/BrowserWindow.cjs';
import IReloadOptions from '../types/IReloadOptions.cjs';
import Document from '../../nodes/document/Document.cjs';
import CrossOriginBrowserWindow from '../../window/CrossOriginBrowserWindow.cjs';
import IHistoryItem from '../../history/IHistoryItem.cjs';
/**
 * Browser frame used when constructing a Window instance without a browser.
 */
export default class DetachedBrowserFrame implements IBrowserFrame {
    readonly childFrames: DetachedBrowserFrame[];
    readonly parentFrame: DetachedBrowserFrame | null;
    readonly page: DetachedBrowserPage;
    window: BrowserWindow;
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
     * @param [window] Window.
     */
    constructor(page: DetachedBrowserPage);
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
//# sourceMappingURL=DetachedBrowserFrame.d.ts.map