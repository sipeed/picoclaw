import * as PropertySymbol from '../PropertySymbol.js';
import SVGRect from './SVGRect.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG Animated Number.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedRect
 */
export default class SVGAnimatedRect {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: () => string;
    [PropertySymbol.setAttribute]: (value: string) => void;
    [PropertySymbol.baseVal]: SVGRect | null;
    [PropertySymbol.animVal]: SVGRect | null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param options Options.
     * @param options.getAttribute Get attribute.
     * @param options.setAttribute Set attribute.
     */
    constructor(illegalConstructorSymbol: symbol, window: BrowserWindow, options: {
        getAttribute: () => string | null;
        setAttribute: (value: string) => void;
    });
    /**
     * Returns animated value.
     *
     * @returns Animated value.
     */
    get animVal(): SVGRect;
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value: SVGRect);
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal(): SVGRect;
    /**
     * Returns base value.
     *
     * @param value Base value.
     */
    set baseVal(_value: SVGRect);
}
//# sourceMappingURL=SVGAnimatedRect.d.ts.map