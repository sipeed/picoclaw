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
const SVGTextContentElement_js_1 = __importDefault(require("../svg-text-content-element/SVGTextContentElement.cjs"));
const SVGAnimatedLengthList_js_1 = __importDefault(require("../../svg/SVGAnimatedLengthList.cjs"));
const SVGAnimatedNumberList_js_1 = __importDefault(require("../../svg/SVGAnimatedNumberList.cjs"));
/**
 * SVG Text Positioning Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextPositioningElement
 */
class SVGTextPositioningElement extends SVGTextContentElement_js_1.default {
    // Internal properties
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.dx] = null;
    [PropertySymbol.dy] = null;
    [PropertySymbol.rotate] = null;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedLengthList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x'),
                setAttribute: (value) => this.setAttribute('x', value)
            });
        }
        return this[PropertySymbol.x];
    }
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y() {
        if (!this[PropertySymbol.y]) {
            this[PropertySymbol.y] = new SVGAnimatedLengthList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
    /**
     * Returns dx.
     *
     * @returns DX.
     */
    get dx() {
        if (!this[PropertySymbol.dx]) {
            this[PropertySymbol.dx] = new SVGAnimatedLengthList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('dx'),
                setAttribute: (value) => this.setAttribute('dx', value)
            });
        }
        return this[PropertySymbol.dx];
    }
    /**
     * Returns dy.
     *
     * @returns DY.
     */
    get dy() {
        if (!this[PropertySymbol.dy]) {
            this[PropertySymbol.dy] = new SVGAnimatedLengthList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('dy'),
                setAttribute: (value) => this.setAttribute('dy', value)
            });
        }
        return this[PropertySymbol.dy];
    }
    /**
     * Returns rotate.
     *
     * @returns Rotate.
     */
    get rotate() {
        if (!this[PropertySymbol.rotate]) {
            this[PropertySymbol.rotate] = new SVGAnimatedNumberList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('rotate'),
                setAttribute: (value) => this.setAttribute('rotate', value)
            });
        }
        return this[PropertySymbol.rotate];
    }
}
exports.default = SVGTextPositioningElement;
//# sourceMappingURL=SVGTextPositioningElement.cjs.map