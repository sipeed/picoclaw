"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const BrowserFrameFactory_js_1 = __importDefault(require("./BrowserFrameFactory.cjs"));
/**
 * Browser page utility.
 */
class BrowserPageUtility {
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
            BrowserFrameFactory_js_1.default.destroyFrame(page.mainFrame)
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
exports.default = BrowserPageUtility;
//# sourceMappingURL=BrowserPageUtility.cjs.map