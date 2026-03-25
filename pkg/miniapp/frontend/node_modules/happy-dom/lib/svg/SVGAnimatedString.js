import * as PropertySymbol from '../PropertySymbol.js';
/**
 * SVG Animated String.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedString
 */
export default class SVGAnimatedString {
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param options Options.
     * @param options.getAttribute Get attribute.
     * @param options.setAttribute Set attribute.
     */
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        this[PropertySymbol.getAttribute] = options.getAttribute;
        this[PropertySymbol.setAttribute] = options.setAttribute;
    }
    /**
     * Returns animated value.
     *
     * @returns Animated value.
     */
    get animVal() {
        return this.baseVal;
    }
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value) {
        // Do nothing
    }
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal() {
        const attributeValue = this[PropertySymbol.getAttribute]();
        if (!attributeValue) {
            return '';
        }
        return attributeValue;
    }
    /**
     * Sets base value.
     *
     * @param value Base value.
     */
    set baseVal(value) {
        this[PropertySymbol.setAttribute](String(value));
    }
}
//# sourceMappingURL=SVGAnimatedString.js.map