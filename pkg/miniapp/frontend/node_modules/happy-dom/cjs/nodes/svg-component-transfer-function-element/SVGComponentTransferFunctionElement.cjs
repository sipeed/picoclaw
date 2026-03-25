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
const SVGElement_js_1 = __importDefault(require("../svg-element/SVGElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGAnimatedNumber_js_1 = __importDefault(require("../../svg/SVGAnimatedNumber.cjs"));
const SVGAnimatedEnumeration_js_1 = __importDefault(require("../../svg/SVGAnimatedEnumeration.cjs"));
const SVGAnimatedNumberList_js_1 = __importDefault(require("../../svg/SVGAnimatedNumberList.cjs"));
/**
 * SVGComponentTransferFunctionElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGComponentTransferFunctionElement
 */
class SVGComponentTransferFunctionElement extends SVGElement_js_1.default {
    // Static properties
    static SVG_FECOMPONENTTRANSFER_TYPE_UNKNOWN = 0;
    static SVG_FECOMPONENTTRANSFER_TYPE_IDENTITY = 1;
    static SVG_FECOMPONENTTRANSFER_TYPE_TABLE = 2;
    static SVG_FECOMPONENTTRANSFER_TYPE_DISCRETE = 3;
    static SVG_FECOMPONENTTRANSFER_TYPE_LINEAR = 4;
    static SVG_FECOMPONENTTRANSFER_TYPE_GAMMA = 5;
    // Internal properties
    [PropertySymbol.type] = null;
    [PropertySymbol.tableValues] = null;
    [PropertySymbol.slope] = null;
    [PropertySymbol.intercept] = null;
    [PropertySymbol.amplitude] = null;
    [PropertySymbol.exponent] = null;
    [PropertySymbol.offset] = null;
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        if (!this[PropertySymbol.type]) {
            this[PropertySymbol.type] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('type'),
                setAttribute: (value) => this.setAttribute('type', value),
                values: ['identity', 'table', 'discrete', 'linear', 'gamma'],
                defaultValue: 'identity'
            });
        }
        return this[PropertySymbol.type];
    }
    /**
     * Returns table values.
     *
     * @returns Table values.
     */
    get tableValues() {
        if (!this[PropertySymbol.tableValues]) {
            this[PropertySymbol.tableValues] = new SVGAnimatedNumberList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('tableValues'),
                setAttribute: (value) => this.setAttribute('tableValues', value)
            });
        }
        return this[PropertySymbol.tableValues];
    }
    /**
     * Returns slope.
     *
     * @returns Slope.
     */
    get slope() {
        if (!this[PropertySymbol.slope]) {
            this[PropertySymbol.slope] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('slope'),
                setAttribute: (value) => this.setAttribute('slope', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.slope];
    }
    /**
     * Returns intercept.
     *
     * @returns Intercept.
     */
    get intercept() {
        if (!this[PropertySymbol.intercept]) {
            this[PropertySymbol.intercept] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('intercept'),
                setAttribute: (value) => this.setAttribute('intercept', value)
            });
        }
        return this[PropertySymbol.intercept];
    }
    /**
     * Returns amplitude.
     *
     * @returns Amplitude.
     */
    get amplitude() {
        if (!this[PropertySymbol.amplitude]) {
            this[PropertySymbol.amplitude] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('amplitude'),
                setAttribute: (value) => this.setAttribute('amplitude', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.amplitude];
    }
    /**
     * Returns exponent.
     *
     * @returns Exponent.
     */
    get exponent() {
        if (!this[PropertySymbol.exponent]) {
            this[PropertySymbol.exponent] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('exponent'),
                setAttribute: (value) => this.setAttribute('exponent', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.exponent];
    }
    /**
     * Returns offset.
     *
     * @returns Offset.
     */
    get offset() {
        if (!this[PropertySymbol.offset]) {
            this[PropertySymbol.offset] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('offset'),
                setAttribute: (value) => this.setAttribute('offset', value)
            });
        }
        return this[PropertySymbol.offset];
    }
}
exports.default = SVGComponentTransferFunctionElement;
//# sourceMappingURL=SVGComponentTransferFunctionElement.cjs.map