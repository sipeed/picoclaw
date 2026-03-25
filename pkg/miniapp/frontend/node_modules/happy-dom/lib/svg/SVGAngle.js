import SVGAngleTypeEnum from './SVGAngleTypeEnum.js';
import * as PropertySymbol from '../PropertySymbol.js';
const ATTRIBUTE_REGEXP = /^(\d+|\d+\.\d+)(deg|rad|grad|turn|)$/;
/**
 * SVG angle.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAngle
 */
export default class SVGAngle {
    // Static properties
    static SVG_ANGLETYPE_UNKNOWN = SVGAngleTypeEnum.unknown;
    static SVG_ANGLETYPE_UNSPECIFIED = SVGAngleTypeEnum.unspecified;
    static SVG_ANGLETYPE_DEG = SVGAngleTypeEnum.deg;
    static SVG_ANGLETYPE_RAD = SVGAngleTypeEnum.rad;
    static SVG_ANGLETYPE_GRAD = SVGAngleTypeEnum.grad;
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
    [PropertySymbol.attributeValue] = '';
    [PropertySymbol.readOnly] = false;
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
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        if (options) {
            this[PropertySymbol.readOnly] = !!options.readOnly;
            this[PropertySymbol.getAttribute] = options.getAttribute || null;
            this[PropertySymbol.setAttribute] = options.setAttribute || null;
        }
    }
    /**
     * Returns unit type.
     *
     * @returns Unit type.
     */
    get unitType() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        const match = attributeValue?.match(ATTRIBUTE_REGEXP);
        if (!match) {
            return SVGAngleTypeEnum.unknown;
        }
        if (isNaN(parseFloat(match[1]))) {
            return SVGAngleTypeEnum.unknown;
        }
        switch (match[2]) {
            case '':
                return SVGAngleTypeEnum.unspecified;
            case 'deg':
                return SVGAngleTypeEnum.deg;
            case 'rad':
                return SVGAngleTypeEnum.rad;
            case 'grad':
                return SVGAngleTypeEnum.grad;
            case 'turn':
                return SVGAngleTypeEnum.unknown;
            default:
                return SVGAngleTypeEnum.unspecified;
        }
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        const match = attributeValue?.match(ATTRIBUTE_REGEXP);
        if (!match) {
            return 0;
        }
        const parsedValue = parseFloat(match[1]);
        if (isNaN(parsedValue)) {
            return 0;
        }
        switch (match[2]) {
            case '':
                return parsedValue;
            case 'deg':
                return parsedValue;
            case 'rad':
                return parsedValue * (180 / Math.PI);
            case 'grad':
                return parsedValue * (180 / 200);
            case 'turn':
                return parsedValue * 360;
            default:
                return parsedValue;
        }
    }
    /**
     * Sets value.
     *
     * @param value Value in pixels.
     */
    set value(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'value' property on 'SVGAngle': The object is read-only.`);
        }
        // Value in pixels
        value = typeof value !== 'number' ? parseFloat(String(value)) : value;
        if (isNaN(value)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'value' property on 'SVGAngle': The provided float value is non-finite.`);
        }
        let unitType = '';
        let valueInSpecifiedUnits = value;
        switch (this.unitType) {
            case SVGAngleTypeEnum.unspecified:
                valueInSpecifiedUnits = value;
                unitType = '';
                break;
            case SVGAngleTypeEnum.deg:
                valueInSpecifiedUnits = value;
                unitType = 'deg';
                break;
            case SVGAngleTypeEnum.rad:
                valueInSpecifiedUnits = value / (180 / Math.PI);
                unitType = 'rad';
                break;
            case SVGAngleTypeEnum.grad:
                valueInSpecifiedUnits = value / (180 / 200);
                unitType = 'grad';
                break;
            case SVGAngleTypeEnum.unknown:
                valueInSpecifiedUnits = value / 360;
                unitType = 'turn';
            default:
                break;
        }
        this[PropertySymbol.attributeValue] = String(valueInSpecifiedUnits) + unitType;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Returns value as string.
     *
     * @returns Value as string.
     */
    get valueAsString() {
        return this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]() || '0'
            : this[PropertySymbol.attributeValue] || '0';
    }
    /**
     * Returns value in specified units.
     *
     * @returns Value in specified units.
     */
    get valueInSpecifiedUnits() {
        const attributeValue = this.valueAsString;
        return parseFloat(attributeValue) || 0;
    }
    /**
     * New value specific units.
     * @param unitType
     * @param value
     */
    newValueSpecifiedUnits(unitType, value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'newValueSpecifiedUnits' on 'SVGAngle': The object is read-only.`);
        }
        if (typeof unitType !== 'number') {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'newValueSpecifiedUnits' on 'SVGAngle': parameter 1 ('unitType') is not of type 'number'.`);
        }
        value = typeof value !== 'number' ? parseFloat(value) : value;
        if (isNaN(value)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'newValueSpecifiedUnits' on 'SVGAngle': The provided float value is non-finite.`);
        }
        let unit = '';
        switch (unitType) {
            case SVGAngleTypeEnum.unspecified:
                unit = '';
                break;
            case SVGAngleTypeEnum.deg:
                unit = 'deg';
                break;
            case SVGAngleTypeEnum.rad:
                unit = 'rad';
                break;
            case SVGAngleTypeEnum.grad:
                unit = 'grad';
                break;
            case SVGAngleTypeEnum.unknown:
                unit = 'turn';
                break;
            default:
                break;
        }
        this[PropertySymbol.attributeValue] = String(value) + unit;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Convert to specific units.
     * @param unitType
     */
    convertToSpecifiedUnits(unitType) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'convertToSpecifiedUnits' on 'SVGAngle': The object is read-only.`);
        }
        if (typeof unitType !== 'number') {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'convertToSpecifiedUnits' on 'SVGAngle': parameter 1 ('unitType') is not of type 'number'.`);
        }
        let value = this.value;
        let unit = '';
        switch (unitType) {
            case SVGAngleTypeEnum.unspecified:
                unit = '';
                break;
            case SVGAngleTypeEnum.deg:
                unit = 'deg';
                break;
            case SVGAngleTypeEnum.rad:
                unit = 'rad';
                value = value / (180 / Math.PI);
                break;
            case SVGAngleTypeEnum.grad:
                unit = 'grad';
                value = value / (180 / 200);
                break;
            case SVGAngleTypeEnum.unknown:
                unit = 'turn';
                value = value / 360;
                break;
            default:
                break;
        }
        this[PropertySymbol.attributeValue] = String(value) + unit;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
}
//# sourceMappingURL=SVGAngle.js.map