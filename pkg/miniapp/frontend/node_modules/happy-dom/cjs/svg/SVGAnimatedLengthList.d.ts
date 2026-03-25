import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
import SVGLengthList from './SVGLengthList.cjs';
/**
 * SVG Animated Length.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedLengthList
 */
export default class SVGAnimatedLengthList {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: () => string;
    [PropertySymbol.setAttribute]: (value: string) => void;
    [PropertySymbol.baseVal]: SVGLengthList | null;
    [PropertySymbol.animVal]: SVGLengthList | null;
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
    get animVal(): SVGLengthList;
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value: SVGLengthList);
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal(): SVGLengthList;
    /**
     * Returns base value.
     *
     * @param value Base value.
     */
    set baseVal(_value: SVGLengthList);
}
//# sourceMappingURL=SVGAnimatedLengthList.d.ts.map