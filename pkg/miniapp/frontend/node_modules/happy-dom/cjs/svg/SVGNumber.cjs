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
 * SVG number.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGNumber
 */
class SVGNumber {
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
     * @param [options.setAttribute] Set attribute
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
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        return parseFloat(attributeValue || '0');
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new TypeError(`Failed to set the 'value' property on 'SVGNumber': The object is read-only.`);
        }
        const parsedValue = typeof value !== 'number' ? parseFloat(String(value)) : value;
        if (isNaN(parsedValue)) {
            throw new TypeError(`Failed to set the 'value' property on 'SVGNumber': The provided value is not a number.`);
        }
        this[PropertySymbol.attributeValue] = String(value);
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue] || '');
        }
    }
}
exports.default = SVGNumber;
//# sourceMappingURL=SVGNumber.cjs.map