import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG Animated Enumaration.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedEnumeration
 */
export default class SVGAnimatedEnumeration {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: () => string;
    [PropertySymbol.setAttribute]: (value: string) => void;
    [PropertySymbol.values]: Array<string | null>;
    [PropertySymbol.defaultValue]: string;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param options Options.
     * @param options.getAttribute Get attribute.
     * @param options.setAttribute Set attribute.
     * @param options.values Values.
     * @param options.defaultValue Default value.
     */
    constructor(illegalConstructorSymbol: symbol, window: BrowserWindow, options: {
        getAttribute: () => string | null;
        setAttribute: (value: string) => void;
        values: Array<string | null>;
        defaultValue: string;
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
     * @param _value Animated value.
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
//# sourceMappingURL=SVGAnimatedEnumeration.d.ts.map