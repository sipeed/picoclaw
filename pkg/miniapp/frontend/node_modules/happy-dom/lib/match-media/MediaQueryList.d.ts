import EventTarget from '../event/EventTarget.js';
import Event from '../event/Event.js';
import BrowserWindow from '../window/BrowserWindow.js';
import TEventListener from '../event/TEventListener.js';
/**
 * Media Query List.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/MediaQueryList.
 */
export default class MediaQueryList extends EventTarget {
    #private;
    onchange: (event: Event) => void;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.window Owner window.
     * @param options.media Media.
     * @param [options.rootFontSize] Root font size.
     */
    constructor(options: {
        window: BrowserWindow;
        media: string;
        rootFontSize?: string | number;
    });
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media(): string;
    /**
     * Returns "true" if the document matches.
     *
     * @returns Matches.
     */
    get matches(): boolean;
    /**
     * Adds a listener.
     *
     * @deprecated
     * @param callback Callback.
     */
    addListener(callback: (event: Event) => void): void;
    /**
     * Removes listener.
     *
     * @deprecated
     * @param callback Callback.
     */
    removeListener(callback: (event: Event) => void): void;
    /**
     * @override
     */
    addEventListener(type: string, listener: TEventListener): void;
    /**
     * @override
     */
    removeEventListener(type: string, listener: TEventListener): void;
}
//# sourceMappingURL=MediaQueryList.d.ts.map