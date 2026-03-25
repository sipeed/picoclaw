import IBrowserFrame from '../types/IBrowserFrame.cjs';
import { URL } from 'url';
/**
 * Browser frame validator.
 */
export default class BrowserFrameValidator {
    /**
     * Returns true if the frame navigation complies with the cross origin policy.
     *
     * @param frame Frame.
     * @param toURL URL.
     * @returns True if the frame navigation complies with the cross origin policy.
     */
    static validateCrossOriginPolicy(frame: IBrowserFrame, toURL: URL): boolean;
    /**
     * Returns true if navigation is allowed for the frame.
     *
     * @param frame Frame.
     * @returns True if navigation is allowed for the frame.
     */
    static validateFrameNavigation(frame: IBrowserFrame): boolean;
}
//# sourceMappingURL=BrowserFrameValidator.d.ts.map