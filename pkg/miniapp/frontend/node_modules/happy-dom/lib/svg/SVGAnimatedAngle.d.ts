import * as PropertySymbol from '../PropertySymbol.js';
import SVGAngle from './SVGAngle.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG Animated Angle.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedAngle
 */
export default class SVGAnimatedAngle {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: () => string;
    [PropertySymbol.setAttribute]: (value: string) => void;
    [PropertySymbol.baseVal]: SVGAngle | null;
    [PropertySymbol.animVal]: SVGAngle | null;
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
    get animVal(): SVGAngle;
    /**
     * Sets animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value: SVGAngle);
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal(): SVGAngle;
    /**
     * Sets base value.
     *
     * @param value Base value.
     */
    set baseVal(_value: SVGAngle);
}
//# sourceMappingURL=SVGAnimatedAngle.d.ts.map