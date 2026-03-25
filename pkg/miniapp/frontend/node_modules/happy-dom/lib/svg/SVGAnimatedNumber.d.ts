import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG Animated Number.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedNumber
 */
export default class SVGAnimatedNumber {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: (() => string | null) | null;
    [PropertySymbol.setAttribute]: ((value: string) => void) | null;
    [PropertySymbol.defaultValue]: number;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param options Options.
     * @param options.getAttribute Get attribute.
     * @param options.setAttribute Set attribute.
     * @param options.defaultValue
     */
    constructor(illegalConstructorSymbol: symbol, window: BrowserWindow, options: {
        getAttribute: () => string | null;
        setAttribute: (value: string) => void;
        defaultValue?: number;
    });
    /**
     * Returns animated value.
     *
     * @returns Animated value.
     */
    get animVal(): number;
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value: number);
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal(): number;
    /**
     * Sets base value.
     *
     * @param value Base value.
     */
    set baseVal(value: number);
}
//# sourceMappingURL=SVGAnimatedNumber.d.ts.map