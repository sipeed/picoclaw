import DetachedWindowAPI from './DetachedWindowAPI.cjs';
import IOptionalBrowserSettings from '../browser/types/IOptionalBrowserSettings.cjs';
import BrowserWindow from './BrowserWindow.cjs';
/**
 * Window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
export default class Window extends BrowserWindow {
    readonly happyDOM: DetachedWindowAPI;
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.width] Window width. Defaults to "1024".
     * @param [options.height] Window height. Defaults to "768".
     * @param [options.innerWidth] Inner width. Deprecated. Defaults to "1024".
     * @param [options.innerHeight] Inner height. Deprecated. Defaults to "768".
     * @param [options.url] URL.
     * @param [options.console] Console.
     * @param [options.settings] Settings.
     */
    constructor(options?: {
        width?: number;
        height?: number;
        /** @deprecated Replaced by the "width" property. */
        innerWidth?: number;
        /** @deprecated Replaced by the "height" property. */
        innerHeight?: number;
        url?: string;
        console?: Console;
        settings?: IOptionalBrowserSettings;
    });
}
//# sourceMappingURL=Window.d.ts.map