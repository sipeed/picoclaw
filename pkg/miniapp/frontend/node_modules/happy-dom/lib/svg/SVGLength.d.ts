import * as PropertySymbol from '../PropertySymbol.js';
import SVGLengthTypeEnum from './SVGLengthTypeEnum.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG length.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGLength
 */
export default class SVGLength {
    static SVG_LENGTHTYPE_UNKNOWN: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_NUMBER: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_PERCENTAGE: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_EMS: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_EXS: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_PX: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_CM: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_MM: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_IN: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_PT: SVGLengthTypeEnum;
    static SVG_LENGTHTYPE_PC: SVGLengthTypeEnum;
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
     * Returns unit type.
     *
     * @returns Unit type.
     */
    get unitType(): SVGLengthTypeEnum;
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value(): number;
    /**
     * Sets value.
     *
     * @param value Value in pixels.
     */
    set value(value: number);
    /**
     * Returns value as string.
     *
     * @returns Value as string.
     */
    get valueAsString(): string;
    /**
     * Returns value in specified units.
     *
     * @returns Value in specified units.
     */
    get valueInSpecifiedUnits(): number;
    /**
     * New value specific units.
     *
     * @param unitType
     * @param value
     */
    newValueSpecifiedUnits(unitType: number, value: number): void;
    /**
     * Convert to specific units.
     * @param unitType
     */
    convertToSpecifiedUnits(unitType: SVGLengthTypeEnum): void;
}
//# sourceMappingURL=SVGLength.d.ts.map