"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const SVGLengthTypeEnum_js_1 = __importDefault(require("./SVGLengthTypeEnum.cjs"));
const ATTRIBUTE_REGEXP = /^(\d+|\d+\.\d+)(px|em|ex|cm|mm|in|pt|pc|%|)$/;
/**
 * SVG length.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGLength
 */
class SVGLength {
    // Static properties
    static SVG_LENGTHTYPE_UNKNOWN = SVGLengthTypeEnum_js_1.default.unknown;
    static SVG_LENGTHTYPE_NUMBER = SVGLengthTypeEnum_js_1.default.number;
    static SVG_LENGTHTYPE_PERCENTAGE = SVGLengthTypeEnum_js_1.default.percentage;
    static SVG_LENGTHTYPE_EMS = SVGLengthTypeEnum_js_1.default.ems;
    static SVG_LENGTHTYPE_EXS = SVGLengthTypeEnum_js_1.default.exs;
    static SVG_LENGTHTYPE_PX = SVGLengthTypeEnum_js_1.default.px;
    static SVG_LENGTHTYPE_CM = SVGLengthTypeEnum_js_1.default.cm;
    static SVG_LENGTHTYPE_MM = SVGLengthTypeEnum_js_1.default.mm;
    static SVG_LENGTHTYPE_IN = SVGLengthTypeEnum_js_1.default.in;
    static SVG_LENGTHTYPE_PT = SVGLengthTypeEnum_js_1.default.pt;
    static SVG_LENGTHTYPE_PC = SVGLengthTypeEnum_js_1.default.pc;
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
    [PropertySymbol.attributeValue] = null;
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
            ? this[PropertySymbol.getAttribute]() || ''
            : this[PropertySymbol.attributeValue] || '';
        const match = attributeValue.match(ATTRIBUTE_REGEXP);
        if (!match) {
            return SVGLengthTypeEnum_js_1.default.unknown;
        }
        if (isNaN(parseFloat(match[1]))) {
            return SVGLengthTypeEnum_js_1.default.unknown;
        }
        switch (match[2]) {
            case '':
                return SVGLengthTypeEnum_js_1.default.number;
            case 'px':
                return SVGLengthTypeEnum_js_1.default.px;
            case 'cm':
                return SVGLengthTypeEnum_js_1.default.cm;
            case 'mm':
                return SVGLengthTypeEnum_js_1.default.mm;
            case 'in':
                return SVGLengthTypeEnum_js_1.default.in;
            case 'pt':
                return SVGLengthTypeEnum_js_1.default.pt;
            case 'pc':
                return SVGLengthTypeEnum_js_1.default.pc;
            case 'em':
            case 'ex':
            case '%':
                throw new this[PropertySymbol.window].TypeError(`Failed to execute 'value' on 'SVGLength': Could not resolve relative length.`);
            default:
                return SVGLengthTypeEnum_js_1.default.unknown;
        }
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]() || ''
            : this[PropertySymbol.attributeValue] || '';
        const match = attributeValue.match(ATTRIBUTE_REGEXP);
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
            case 'px':
                return parsedValue;
            case 'cm':
                return (parsedValue / 2.54) * 96;
            case 'mm':
                return (parsedValue / 25.4) * 96;
            case 'in':
                return parsedValue * 96;
            case 'pt':
                return parsedValue * 72;
            case 'pc':
                return parsedValue * 6;
            case 'em':
            case 'ex':
            case '%':
                throw new this[PropertySymbol.window].TypeError(`Failed to execute 'value' on 'SVGLength': Could not resolve relative length.`);
            default:
                return 0;
        }
    }
    /**
     * Sets value.
     *
     * @param value Value in pixels.
     */
    set value(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'value' property on 'SVGLength': The object is read-only.`);
        }
        // Value in pixels
        value = typeof value !== 'number' ? parseFloat(String(value)) : value;
        if (isNaN(value)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'value' property on 'SVGLength': The provided float value is non-finite.`);
        }
        let unitType = '';
        let valueInSpecifiedUnits = value;
        switch (this.unitType) {
            case SVGLengthTypeEnum_js_1.default.number:
                valueInSpecifiedUnits = value;
                unitType = '';
                break;
            case SVGLengthTypeEnum_js_1.default.px:
                valueInSpecifiedUnits = value;
                unitType = 'px';
                break;
            case SVGLengthTypeEnum_js_1.default.cm:
                valueInSpecifiedUnits = (value / 96) * 2.54;
                unitType = 'cm';
                break;
            case SVGLengthTypeEnum_js_1.default.mm:
                valueInSpecifiedUnits = (value / 96) * 25.4;
                unitType = 'mm';
                break;
            case SVGLengthTypeEnum_js_1.default.in:
                valueInSpecifiedUnits = value / 96;
                unitType = 'in';
                break;
            case SVGLengthTypeEnum_js_1.default.pt:
                valueInSpecifiedUnits = value / 72;
                unitType = 'pt';
                break;
            case SVGLengthTypeEnum_js_1.default.pc:
                valueInSpecifiedUnits = value / 6;
                unitType = 'pc';
                break;
            case SVGLengthTypeEnum_js_1.default.percentage:
            case SVGLengthTypeEnum_js_1.default.ems:
            case SVGLengthTypeEnum_js_1.default.exs:
                throw new this[PropertySymbol.window].TypeError(`Failed to set the 'value' property on 'SVGLength': Could not resolve relative length.`);
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
     *
     * @param unitType
     * @param value
     */
    newValueSpecifiedUnits(unitType, value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'newValueSpecifiedUnits' on 'SVGLength': The object is read-only.`);
        }
        if (typeof unitType !== 'number') {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'newValueSpecifiedUnits' on 'SVGLength': parameter 1 ('unitType') is not of type 'number'.`);
        }
        value = typeof value !== 'number' ? parseFloat(String(value)) : value;
        if (isNaN(value)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'newValueSpecifiedUnits' on 'SVGLength': The provided float value is non-finite.`);
        }
        let unit = '';
        switch (unitType) {
            case SVGLengthTypeEnum_js_1.default.number:
                unit = '';
                break;
            case SVGLengthTypeEnum_js_1.default.px:
                unit = 'px';
                break;
            case SVGLengthTypeEnum_js_1.default.cm:
                unit = 'cm';
                break;
            case SVGLengthTypeEnum_js_1.default.mm:
                unit = 'mm';
                break;
            case SVGLengthTypeEnum_js_1.default.in:
                unit = 'in';
                break;
            case SVGLengthTypeEnum_js_1.default.pt:
                unit = 'pt';
                break;
            case SVGLengthTypeEnum_js_1.default.pc:
                unit = 'pc';
                break;
            case SVGLengthTypeEnum_js_1.default.ems:
            case SVGLengthTypeEnum_js_1.default.exs:
            case SVGLengthTypeEnum_js_1.default.percentage:
                throw new this[PropertySymbol.window].TypeError(`Failed to execute 'newValueSpecifiedUnits' on 'SVGLength': Could not resolve relative length.`);
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
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'convertToSpecifiedUnits' on 'SVGLength': The object is read-only.`);
        }
        if (typeof unitType !== 'number') {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'convertToSpecifiedUnits' on 'SVGLength': parameter 1 ('unitType') is not of type 'number'.`);
        }
        let value = this.value;
        let unit = '';
        switch (unitType) {
            case SVGLengthTypeEnum_js_1.default.number:
                unit = '';
                break;
            case SVGLengthTypeEnum_js_1.default.px:
                unit = 'px';
                break;
            case SVGLengthTypeEnum_js_1.default.cm:
                value = (value / 96) * 2.54;
                unit = 'cm';
                break;
            case SVGLengthTypeEnum_js_1.default.mm:
                value = (value / 96) * 25.4;
                unit = 'mm';
                break;
            case SVGLengthTypeEnum_js_1.default.in:
                value = value / 96;
                unit = 'in';
                break;
            case SVGLengthTypeEnum_js_1.default.pt:
                value = value / 72;
                unit = 'pt';
                break;
            case SVGLengthTypeEnum_js_1.default.pc:
                value = value / 6;
                unit = 'pc';
                break;
            case SVGLengthTypeEnum_js_1.default.percentage:
            case SVGLengthTypeEnum_js_1.default.ems:
            case SVGLengthTypeEnum_js_1.default.exs:
                throw new this[PropertySymbol.window].TypeError(`Failed to execute 'convertToSpecifiedUnits' on 'SVGLength': Could not resolve relative length.`);
            default:
                break;
        }
        this[PropertySymbol.attributeValue] = String(value) + unit;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
}
exports.default = SVGLength;
//# sourceMappingURL=SVGLength.cjs.map