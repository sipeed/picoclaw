"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const MediaQueryListEvent_js_1 = __importDefault(require("../event/events/MediaQueryListEvent.cjs"));
const MediaQueryParser_js_1 = __importDefault(require("./MediaQueryParser.cjs"));
/**
 * Media Query List.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/MediaQueryList.
 */
class MediaQueryList extends EventTarget_js_1.default {
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
                MediaQueryParser_js_1.default.parse({
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
                MediaQueryParser_js_1.default.parse({
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
                    this.dispatchEvent(new MediaQueryListEvent_js_1.default('change', { matches, media: this.media }));
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
exports.default = MediaQueryList;
//# sourceMappingURL=MediaQueryList.cjs.map