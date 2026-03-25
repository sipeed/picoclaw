import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG number.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGNumber
 */
export default class SVGNumber {
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
     * Returns value.
     *
     * @returns Value.
     */
    get value(): number;
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value: number);
}
//# sourceMappingURL=SVGNumber.d.ts.map