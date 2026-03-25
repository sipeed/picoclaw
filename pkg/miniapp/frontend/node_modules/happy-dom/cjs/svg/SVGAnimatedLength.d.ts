import * as PropertySymbol from '../PropertySymbol.cjs';
import SVGLength from './SVGLength.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * SVG Animated Length.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedLength
 */
export default class SVGAnimatedLength {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: () => string;
    [PropertySymbol.setAttribute]: (value: string) => void;
    [PropertySymbol.baseVal]: SVGLength | null;
    [PropertySymbol.animVal]: SVGLength | null;
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
    get animVal(): SVGLength;
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value: SVGLength);
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal(): SVGLength;
    /**
     * Returns base value.
     *
     * @param value Base value.
     */
    set baseVal(_value: SVGLength);
}
//# sourceMappingURL=SVGAnimatedLength.d.ts.map