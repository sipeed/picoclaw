import IBrowserFrame from '../browser/types/IBrowserFrame.cjs';
import HistoryScrollRestorationEnum from './HistoryScrollRestorationEnum.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * History API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/History.
 */
export default class History {
    #private;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     * @param window Owner window.
     */
    constructor(browserFrame: IBrowserFrame, window: BrowserWindow);
    /**
     * Returns the history length.
     *
     * @returns History length.
     */
    get length(): number;
    /**
     * Returns an any value representing the state at the top of the history stack. This is a way to look at the state without having to wait for a popstate event.
     *
     * @returns State.
     */
    get state(): object | null;
    /**
     * Returns scroll restoration.
     *
     * @returns Sroll restoration.
     */
    get scrollRestoration(): HistoryScrollRestorationEnum;
    /**
     * Sets scroll restoration.
     *
     * @param scrollRestoration Sroll restoration.
     */
    set scrollRestoration(scrollRestoration: HistoryScrollRestorationEnum);
    /**
     * Goes to the previous page in session history.
     */
    back(): void;
    /**
     * Goes to the next page in session history.
     */
    forward(): void;
    /**
     * Load a specific page from the session history.
     *
     * @param delta Delta.
     * @param _delta
     */
    go(delta: number): void;
    /**
     * Pushes the given data onto the session history stack.
     *
     * @param state State.
     * @param title Title.
     * @param [url] URL.
     */
    pushState(state: object, title: any, url?: string | URL): void;
    /**
     * This method modifies the current history entry, replacing it with a new state.
     *
     * @param state State.
     * @param title Title.
     * @param [url] URL.
     */
    replaceState(state: object, title: any, url?: string | URL): void;
    /**
     * Destroys the history.
     *
     * This will make sure that the History API can't access page data from the next history item.
     */
    [PropertySymbol.destroy](): void;
}
//# sourceMappingURL=History.d.ts.map