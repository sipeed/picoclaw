import IBrowserFrame from '../types/IBrowserFrame.js';
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
    static createChildFrame(parentFrame: IBrowserFrame): IBrowserFrame;
    /**
     * Aborts all ongoing operations and destroys the frame.
     *
     * @param frame Frame.
     */
    static destroyFrame(frame: IBrowserFrame): Promise<void>;
}
//# sourceMappingURL=BrowserFrameFactory.d.ts.map