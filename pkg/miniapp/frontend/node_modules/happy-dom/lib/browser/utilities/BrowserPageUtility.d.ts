import IBrowserFrame from '../types/IBrowserFrame.js';
import IBrowserPage from '../types/IBrowserPage.js';
/**
 * Browser page utility.
 */
export default class BrowserPageUtility {
    /**
     * Returns frames for a page.
     *
     * @param page Page.
     * @returns Frames.
     */
    static getFrames(page: IBrowserPage): IBrowserFrame[];
    /**
     * Aborts all ongoing operations and destroys the page.
     *
     * @param page Page.
     */
    static closePage(page: IBrowserPage): Promise<void>;
    /**
     * Returns all frames.
     *
     * @param parentFrame Parent frame.
     * @returns Frames, including the parent.
     */
    private static findFrames;
}
//# sourceMappingURL=BrowserPageUtility.d.ts.map