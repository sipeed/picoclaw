import BrowserFrameFactory from './BrowserFrameFactory.js';
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
    static getFrames(page) {
        return this.findFrames(page.mainFrame);
    }
    /**
     * Aborts all ongoing operations and destroys the page.
     *
     * @param page Page.
     */
    static closePage(page) {
        // Using Promise instead of async/await to prevent microtask
        return new Promise((resolve, reject) => {
            if (!page.mainFrame) {
                resolve();
                return;
            }
            const index = page.context.pages.indexOf(page);
            if (index !== -1) {
                page.context.pages.splice(index, 1);
            }
            BrowserFrameFactory.destroyFrame(page.mainFrame)
                .then(() => {
                page.virtualConsolePrinter = null;
                page.mainFrame = null;
                page.context = null;
                resolve();
            })
                .catch((error) => reject(error));
        });
    }
    /**
     * Returns all frames.
     *
     * @param parentFrame Parent frame.
     * @returns Frames, including the parent.
     */
    static findFrames(parentFrame) {
        let frames = [parentFrame];
        for (const frame of parentFrame.childFrames) {
            frames = frames.concat(this.findFrames(frame));
        }
        return frames;
    }
}
//# sourceMappingURL=BrowserPageUtility.js.map