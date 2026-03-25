import SVGAngleTypeEnum from './SVGAngleTypeEnum.js';
import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * SVG angle.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAngle
 */
export default class SVGAngle {
    static SVG_ANGLETYPE_UNKNOWN: SVGAngleTypeEnum;
    static SVG_ANGLETYPE_UNSPECIFIED: SVGAngleTypeEnum;
    static SVG_ANGLETYPE_DEG: SVGAngleTypeEnum;
    static SVG_ANGLETYPE_RAD: SVGAngleTypeEnum;
    static SVG_ANGLETYPE_GRAD: SVGAngleTypeEnum;
    [PropertySymbol.window]: BrowserWindow;
    [PropertySymbol.getAttribute]: (() => string | null) | null;
    [PropertySymbol.setAttribute]: ((value: string) => void) | null;
    [PropertySymbol.attributeValue]: string;
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
    get unitType(): SVGAngleTypeEnum;
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
     * @param unitType
     * @param value
     */
    newValueSpecifiedUnits(unitType: number, value: number): void;
    /**
     * Convert to specific units.
     * @param unitType
     */
    convertToSpecifiedUnits(unitType: SVGAngleTypeEnum): void;
}
//# sourceMappingURL=SVGAngle.d.ts.map