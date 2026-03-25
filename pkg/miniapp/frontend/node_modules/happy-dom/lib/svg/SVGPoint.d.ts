import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG point.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPoint
 */
export default class SVGPoint {
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: (() => string | null) | null;
    [PropertySymbol.setAttribute]: ((value: string) => void) | null;
    [PropertySymbol.attributeValue]: string | null;
    [PropertySymbol.readOnly]: boolean;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param [options] Options.
     * @param [options.readOnly] Read only.
     * @param [options.getAttribute] Get attribute.
     * @param [options.setAttribute] Set attribute
     */
    constructor(illegalConstructorSymbol: symbol, window: BrowserWindow, options?: {
        readOnly?: boolean;
        getAttribute?: () => string | null;
        setAttribute?: (value: string) => void;
    });
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x(): number;
    /**
     * Sets x.
     *
     * @param value X.
     */
    set x(value: number);
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y(): number;
    /**
     * Sets y.
     *
     * @param value Y.
     */
    set y(value: number);
}
//# sourceMappingURL=SVGPoint.d.ts.map