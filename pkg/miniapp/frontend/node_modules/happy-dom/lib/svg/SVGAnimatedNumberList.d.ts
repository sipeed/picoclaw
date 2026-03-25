import SVGNumberList from './SVGNumberList.js';
import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * The SVGAnimatedNumberList interface is used for attributes which take a list of numbers and which can be animated.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedNumberList
 */
export default class SVGAnimatedNumberList {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: () => string;
    [PropertySymbol.setAttribute]: (value: string) => void;
    [PropertySymbol.baseVal]: SVGNumberList | null;
    [PropertySymbol.animVal]: SVGNumberList | null;
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
    get animVal(): SVGNumberList;
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value: SVGNumberList);
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal(): SVGNumberList;
    /**
     * Sets base value.
     *
     * @param value Base value.
     */
    set baseVal(_value: SVGNumberList);
}
//# sourceMappingURL=SVGAnimatedNumberList.d.ts.map