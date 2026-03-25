import IBrowserFrame from '../types/IBrowserFrame.cjs';
import IGoToOptions from '../types/IGoToOptions.cjs';
import Response from '../../fetch/Response.cjs';
import BrowserWindow from '../../window/BrowserWindow.cjs';
import FormData from '../../form-data/FormData.cjs';
/**
 * Browser frame navigation utility.
 */
export default class BrowserFrameNavigator {
    /**
     * Navigates to a page.
     *
     * @throws Error if the request can't be resolved (because of SSL error or similar). It will not throw if the response is not ok.
     * @param options Options.
     * @param options.windowClass Window class.
     * @param options.frame Frame.
     * @param options.url URL.
     * @param [options.goToOptions] Go to options.
     * @param [options.method] Method.
     * @param [options.formData] Form data.
     * @param [options.disableHistory] Disables adding the navigation to the history.
     * @returns Response.
     */
    static navigate(options: {
        windowClass: new (browserFrame: IBrowserFrame, options?: {
            url?: string;
            width?: number;
            height?: number;
        }) => BrowserWindow | null;
        frame: IBrowserFrame;
        url: string;
        goToOptions?: IGoToOptions;
        method?: string;
        formData?: FormData | null;
        disableHistory?: boolean;
    }): Promise<Response | null>;
    /**
     * Navigates back in history.
     *
     * @param options Options.
     * @param options.windowClass Window class.
     * @param options.frame Frame.
     * @param [options.goToOptions] Go to options.
     */
    static navigateBack(options: {
        windowClass: new (browserFrame: IBrowserFrame, options?: {
            url?: string;
            width?: number;
            height?: number;
        }) => BrowserWindow | null;
        frame: IBrowserFrame;
        goToOptions?: IGoToOptions;
    }): Promise<Response | null>;
    /**
     * Navigates forward in history.
     *
     * @param options Options.
     * @param options.windowClass Window class.
     * @param options.frame Frame.
     * @param [options.goToOptions] Go to options.
     */
    static navigateForward(options: {
        windowClass: new (browserFrame: IBrowserFrame, options?: {
            url?: string;
            width?: number;
            height?: number;
        }) => BrowserWindow | null;
        frame: IBrowserFrame;
        goToOptions?: IGoToOptions;
    }): Promise<Response | null>;
    /**
     * Navigates steps in history.
     *
     * @param options Options.
     * @param options.windowClass Window class.
     * @param options.frame Frame.
     * @param options.goToOptions Go to options.
     * @param options.steps Steps.
     */
    static navigateSteps(options: {
        windowClass: new (browserFrame: IBrowserFrame, options?: {
            url?: string;
            width?: number;
            height?: number;
        }) => BrowserWindow | null;
        frame: IBrowserFrame;
        goToOptions?: IGoToOptions;
        steps?: number;
    }): Promise<Response | null>;
    /**
     * Reloads the current history item.
     *
     * @param options Options.
     * @param options.windowClass Window class.
     * @param options.frame Frame.
     * @param options.goToOptions Go to options.
     */
    static reload(options: {
        windowClass: new (browserFrame: IBrowserFrame, options?: {
            url?: string;
            width?: number;
            height?: number;
        }) => BrowserWindow | null;
        frame: IBrowserFrame;
        goToOptions?: IGoToOptions;
    }): Promise<Response | null>;
}
//# sourceMappingURL=BrowserFrameNavigator.d.ts.map