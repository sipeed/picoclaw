import EventTarget from '../event/EventTarget.js';
import * as PropertySymbol from '../PropertySymbol.js';
import MediaQueryListEvent from '../event/events/MediaQueryListEvent.js';
import MediaQueryParser from './MediaQueryParser.js';
/**
 * Media Query List.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/MediaQueryList.
 */
export default class MediaQueryList extends EventTarget {
    onchange = null;
    #window;
    #items = null;
    #media;
    #rootFontSize = null;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.window Owner window.
     * @param options.media Media.
     * @param [options.rootFontSize] Root font size.
     */
    constructor(options) {
        super();
        this.#window = options.window;
        this.#media = options.media;
        this.#rootFontSize = options.rootFontSize || null;
    }
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        this.#items =
            this.#items ||
                MediaQueryParser.parse({
                    window: this.#window,
                    mediaQuery: this.#media,
                    rootFontSize: this.#rootFontSize
                });
        return this.#items.map((item) => item.toString()).join(', ');
    }
    /**
     * Returns "true" if the document matches.
     *
     * @returns Matches.
     */
    get matches() {
        this.#items =
            this.#items ||
                MediaQueryParser.parse({
                    window: this.#window,
                    mediaQuery: this.#media,
                    rootFontSize: this.#rootFontSize
                });
        for (const item of this.#items) {
            if (!item.matches()) {
                return false;
            }
        }
        return true;
    }
    /**
     * Adds a listener.
     *
     * @deprecated
     * @param callback Callback.
     */
    addListener(callback) {
        this.addEventListener('change', callback);
    }
    /**
     * Removes listener.
     *
     * @deprecated
     * @param callback Callback.
     */
    removeListener(callback) {
        this.removeEventListener('change', callback);
    }
    /**
     * @override
     */
    addEventListener(type, listener) {
        super.addEventListener(type, listener);
        if (type === 'change') {
            let matchesState = false;
            const resizeListener = () => {
                const matches = this.matches;
                if (matches !== matchesState) {
                    matchesState = matches;
                    this.dispatchEvent(new MediaQueryListEvent('change', { matches, media: this.media }));
                }
            };
            listener[PropertySymbol.windowResizeListener] = resizeListener;
            this.#window.addEventListener('resize', resizeListener);
        }
    }
    /**
     * @override
     */
    removeEventListener(type, listener) {
        super.removeEventListener(type, listener);
        if (type === 'change' && listener[PropertySymbol.windowResizeListener]) {
            this.#window.removeEventListener('resize', listener[PropertySymbol.windowResizeListener]);
        }
    }
}
//# sourceMappingURL=MediaQueryList.js.map