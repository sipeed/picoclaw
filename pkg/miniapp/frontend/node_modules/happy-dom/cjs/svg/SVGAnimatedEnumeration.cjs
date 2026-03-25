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
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * SVG Animated Enumaration.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedEnumeration
 */
class SVGAnimatedEnumeration {
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute];
    [PropertySymbol.setAttribute];
    [PropertySymbol.values];
    [PropertySymbol.defaultValue];
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
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        this[PropertySymbol.getAttribute] = options.getAttribute;
        this[PropertySymbol.setAttribute] = options.setAttribute;
        this[PropertySymbol.values] = options.values;
        this[PropertySymbol.defaultValue] = options.defaultValue;
    }
    /**
     * Returns animated value.
     *
     * @returns Animated value.
     */
    get animVal() {
        return this.baseVal;
    }
    /**
     * Returns animated value.
     *
     * @param _value Animated value.
     */
    set animVal(_value) {
        // Do nothing
    }
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal() {
        const value = this[PropertySymbol.getAttribute]();
        if (!value) {
            return this[PropertySymbol.values].indexOf(this[PropertySymbol.defaultValue]) + 1;
        }
        const index = this[PropertySymbol.values].indexOf(value);
        if (index === -1) {
            const anyValueIndex = this[PropertySymbol.values].indexOf(null);
            return anyValueIndex !== -1 ? anyValueIndex + 1 : 0;
        }
        return index + 1;
    }
    /**
     * Sets base value.
     *
     * @param value Base value.
     */
    set baseVal(value) {
        let parsedValue = Number(value);
        if (isNaN(parsedValue)) {
            parsedValue = 0;
        }
        if (parsedValue < 1) {
            throw new TypeError(`Failed to set the 'baseVal' property on 'SVGAnimatedEnumeration': The enumeration value provided is ${parsedValue}, which is not settable.`);
        }
        if (parsedValue > this[PropertySymbol.values].length) {
            throw new TypeError(`Failed to set the 'baseVal' property on 'SVGAnimatedEnumeration': The enumeration value provided (${parsedValue}) is larger than the largest allowed value (${this[PropertySymbol.values].length}).`);
        }
        const currentValue = this[PropertySymbol.getAttribute]();
        const isAnyValue = this[PropertySymbol.values][parsedValue - 1] === null;
        const newValue = isAnyValue ? '0' : this[PropertySymbol.values][parsedValue - 1];
        if (!currentValue ||
            (isAnyValue && this[PropertySymbol.values].includes(currentValue)) ||
            (!isAnyValue && currentValue !== newValue)) {
            this[PropertySymbol.setAttribute](newValue);
        }
    }
}
exports.default = SVGAnimatedEnumeration;
//# sourceMappingURL=SVGAnimatedEnumeration.cjs.map