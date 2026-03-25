import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * Browser frame factory.
 */
export default class BrowserFrameFactory {
    /**
     * Creates a new frame.
     *
     * @param parentFrame Parent frame.
     * @returns Frame.
     */
    static createChildFrame(parentFrame) {
        const frame = new parentFrame.constructor(parentFrame.page);
        frame.parentFrame = parentFrame;
        parentFrame.childFrames.push(frame);
        return frame;
    }
    /**
     * Aborts all ongoing operations and destroys the frame.
     *
     * @param frame Frame.
     */
    static destroyFrame(frame) {
        const exceptionObserver = frame.page?.context?.browser?.[PropertySymbol.exceptionObserver];
        // Using Promise instead of async/await to prevent usage of a microtask
        return new Promise((resolve, reject) => {
            if (!frame.window) {
                resolve();
                return;
            }
            if (frame.parentFrame) {
                const index = frame.parentFrame.childFrames.indexOf(frame);
                if (index !== -1) {
                    frame.parentFrame.childFrames.splice(index, 1);
                }
            }
            if (!frame.childFrames.length) {
                frame[PropertySymbol.asyncTaskManager]
                    .destroy()
                    .then(() => {
                    if (exceptionObserver && frame.window) {
                        exceptionObserver.disconnect(frame.window);
                    }
                    frame.page = null;
                    frame.window = null;
                    frame[PropertySymbol.openerFrame] = null;
                    frame[PropertySymbol.openerWindow] = null;
                    resolve();
                })
                    .catch((error) => reject(error));
                if (frame.window) {
                    frame.window[PropertySymbol.destroy]();
                }
                return;
            }
            Promise.all(frame.childFrames.slice().map((childFrame) => this.destroyFrame(childFrame)))
                .then(() => {
                frame[PropertySymbol.asyncTaskManager]
                    .destroy()
                    .then(() => {
                    if (exceptionObserver && frame.window) {
                        exceptionObserver.disconnect(frame.window);
                    }
                    frame.page = null;
                    frame.window = null;
                    frame[PropertySymbol.openerFrame] = null;
                    frame[PropertySymbol.openerWindow] = null;
                    resolve();
                })
                    .catch((error) => reject(error));
                if (frame.window) {
                    frame.window[PropertySymbol.destroy]();
                }
            })
                .catch((error) => reject(error));
        });
    }
}
//# sourceMappingURL=BrowserFrameFactory.js.map