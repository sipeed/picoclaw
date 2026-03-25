import IBrowserPageViewport from '../types/IBrowserPageViewport.js';
import VirtualConsolePrinter from '../../console/VirtualConsolePrinter.js';
import IBrowserFrame from './IBrowserFrame.js';
import IBrowserContext from './IBrowserContext.js';
import { Script } from 'vm';
import IGoToOptions from './IGoToOptions.js';
import Response from '../../fetch/Response.js';
import IReloadOptions from './IReloadOptions.js';
import IOptionalBrowserPageViewport from './IOptionalBrowserPageViewport.js';
/**
 * Browser page.
 */
export default interface IBrowserPage {
    readonly virtualConsolePrinter: VirtualConsolePrinter;
    readonly mainFrame: IBrowserFrame;
    readonly context: IBrowserContext;
    readonly console: Console;
    readonly frames: IBrowserFrame[];
    readonly viewport: IBrowserPageViewport;
    content: string;
    url: string;
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
     * Reloads the current page.
     *
     * @param [options] Options.
     * @returns Response.
     */
    reload(options?: IReloadOptions): Promise<Response | null>;
}
//# sourceMappingURL=IBrowserPage.d.ts.map