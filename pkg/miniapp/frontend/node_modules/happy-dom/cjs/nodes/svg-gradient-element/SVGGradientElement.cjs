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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGAnimatedEnumeration_js_1 = __importDefault(require("../../svg/SVGAnimatedEnumeration.cjs"));
const SVGAnimatedString_js_1 = __importDefault(require("../../svg/SVGAnimatedString.cjs"));
const SVGAnimatedTransformList_js_1 = __importDefault(require("../../svg/SVGAnimatedTransformList.cjs"));
const SVGGraphicsElement_js_1 = __importDefault(require("../svg-graphics-element/SVGGraphicsElement.cjs"));
/**
 * SVG Gradient Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGradientElement
 */
class SVGGradientElement extends SVGGraphicsElement_js_1.default {
    // Public static properties
    static SVG_SPREADMETHOD_UNKNOWN = 0;
    static SVG_SPREADMETHOD_PAD = 1;
    static SVG_SPREADMETHOD_REFLECT = 2;
    static SVG_SPREADMETHOD_REPEAT = 3;
    // Internal properties
    [PropertySymbol.href] = null;
    [PropertySymbol.gradientUnits] = null;
    [PropertySymbol.gradientTransform] = null;
    [PropertySymbol.spreadMethod] = null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this[PropertySymbol.href]) {
            this[PropertySymbol.href] = new SVGAnimatedString_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('href'),
                setAttribute: (value) => this.setAttribute('href', value)
            });
        }
        return this[PropertySymbol.href];
    }
    /**
     * Returns gradient units.
     *
     * @returns Gradient units.
     */
    get gradientUnits() {
        if (!this[PropertySymbol.gradientUnits]) {
            this[PropertySymbol.gradientUnits] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('gradientUnits'),
                setAttribute: (value) => this.setAttribute('gradientUnits', value),
                values: ['userSpaceOnUse', 'objectBoundingBox'],
                defaultValue: 'objectBoundingBox'
            });
        }
        return this[PropertySymbol.gradientUnits];
    }
    /**
     * Returns gradient transform.
     *
     * @returns Gradient transform.
     */
    get gradientTransform() {
        if (!this[PropertySymbol.gradientTransform]) {
            this[PropertySymbol.gradientTransform] = new SVGAnimatedTransformList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('gradientTransform'),
                setAttribute: (value) => this.setAttribute('gradientTransform', value)
            });
        }
        return this[PropertySymbol.gradientTransform];
    }
    /**
     * Returns spread method.
     *
     * @returns Spread method.
     */
    get spreadMethod() {
        if (!this[PropertySymbol.spreadMethod]) {
            this[PropertySymbol.spreadMethod] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('spreadMethod'),
                setAttribute: (value) => this.setAttribute('spreadMethod', value),
                values: ['pad', 'reflect', 'repeat'],
                defaultValue: 'pad'
            });
        }
        return this[PropertySymbol.spreadMethod];
    }
}
exports.default = SVGGradientElement;
//# sourceMappingURL=SVGGradientElement.cjs.map