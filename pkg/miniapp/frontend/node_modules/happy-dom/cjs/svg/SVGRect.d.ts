import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * Rect object.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGRect
 */
export default class SVGRect {
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
     * @param [options.setAttribute] Set attribute.
     */
    constructor(illegalConstructorSymbol: symbol, window: BrowserWindow, options?: {
        readOnly?: boolean;
        getAttribute?: () => string | null;
        setAttribute?: (value: string) => void;
    });
    /**
     * Returns x value.
     *
     * @returns X value.
     */
    get x(): number;
    /**
     * Sets x value.
     *
     * @param value X value.
     */
    set x(value: number);
    /**
     * Returns y value.
     *
     * @returns Y value.
     */
    get y(): number;
    /**
     * Sets y value.
     *
     * @param value Y value.
     */
    set y(value: number);
    /**
     * Returns width value.
     *
     * @returns Width value.
     */
    get width(): number;
    /**
     * Sets width value.
     *
     * @param value Width value.
     */
    set width(value: number);
    /**
     * Returns height value.
     *
     * @returns Height value.
     */
    get height(): number;
    /**
     * Sets height value.
     *
     * @param value Height value.
     */
    set height(value: number);
}
//# sourceMappingURL=SVGRect.d.ts.map