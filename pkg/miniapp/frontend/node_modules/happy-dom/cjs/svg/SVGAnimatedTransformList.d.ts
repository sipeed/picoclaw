import SVGTransformList from './SVGTransformList.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * The SVGAnimatedTransformList interface is used for attributes which take a list of numbers and which can be animated.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedTransformList
 */
export default class SVGAnimatedTransformList {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: () => string;
    [PropertySymbol.setAttribute]: (value: string) => void;
    [PropertySymbol.baseVal]: SVGTransformList | null;
    [PropertySymbol.animVal]: SVGTransformList | null;
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
    get animVal(): SVGTransformList;
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value: SVGTransformList);
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal(): SVGTransformList;
    /**
     * Returns base value.
     *
     * @param value Base value.
     */
    set baseVal(_value: SVGTransformList);
}
//# sourceMappingURL=SVGAnimatedTransformList.d.ts.map